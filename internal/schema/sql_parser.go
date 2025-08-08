package schema

import (
	"context"
	"os"
	"regexp"
	"sort"
	"strings"
)

// SQLStatement represents a parsed SQL statement that can be applied to a schema
type SQLStatement interface {
	Apply(schema *Schema) error
	String() string
}

// ColumnDefinition represents a column in a table
type ColumnDefinition struct {
	Name          string
	Type          string
	NotNull       bool
	Default       string
	PrimaryKey    bool
	AutoIncrement bool
}

// CreateTableStatement represents a CREATE TABLE SQL statement
type CreateTableStatement struct {
	TableName string
	Columns   []ColumnDefinition
}

func (c *CreateTableStatement) Apply(schema *Schema) error {
	model := &Model{
		Name:      c.TableName,
		TableName: c.TableName,
		Fields:    make([]*Field, 0, len(c.Columns)),
	}

	for _, col := range c.Columns {
		field := &Field{
			Name:       col.Name,
			ColumnName: col.Name,
			Type:       col.Type,
			IsOptional: !col.NotNull && !col.PrimaryKey,
		}
		model.Fields = append(model.Fields, field)
	}

	schema.Models = append(schema.Models, model)
	return nil
}

func (c *CreateTableStatement) String() string {
	return "CREATE TABLE " + c.TableName
}

// AlterTableStatement represents various ALTER TABLE operations
type AlterTableStatement struct {
	TableName string
	Operation AlterOperation
}

type AlterOperation interface {
	Apply(model *Model) error
	String() string
}

// AddColumnOperation represents ALTER TABLE ADD COLUMN
type AddColumnOperation struct {
	Column ColumnDefinition
}

func (a *AddColumnOperation) Apply(model *Model) error {
	field := &Field{
		Name:       a.Column.Name,
		ColumnName: a.Column.Name,
		Type:       a.Column.Type,
		IsOptional: !a.Column.NotNull && !a.Column.PrimaryKey,
	}
	model.Fields = append(model.Fields, field)
	return nil
}

func (a *AddColumnOperation) String() string {
	return "ADD COLUMN " + a.Column.Name
}

// DropColumnOperation represents ALTER TABLE DROP COLUMN
type DropColumnOperation struct {
	ColumnName string
}

func (d *DropColumnOperation) Apply(model *Model) error {
	newFields := make([]*Field, 0, len(model.Fields))
	for _, field := range model.Fields {
		if field.ColumnName != d.ColumnName {
			newFields = append(newFields, field)
		}
	}
	model.Fields = newFields
	return nil
}

func (d *DropColumnOperation) String() string {
	return "DROP COLUMN " + d.ColumnName
}

// AlterColumnTypeOperation represents ALTER TABLE ALTER COLUMN TYPE
type AlterColumnTypeOperation struct {
	ColumnName string
	NewType    string
}

func (a *AlterColumnTypeOperation) Apply(model *Model) error {
	for _, field := range model.Fields {
		if field.ColumnName == a.ColumnName {
			field.Type = a.NewType
			break
		}
	}
	return nil
}

func (a *AlterColumnTypeOperation) String() string {
	return "ALTER COLUMN " + a.ColumnName + " TYPE " + a.NewType
}

func (a *AlterTableStatement) Apply(schema *Schema) error {
	// Find the model to alter
	for _, model := range schema.Models {
		if model.TableName == a.TableName {
			return a.Operation.Apply(model)
		}
	}
	return nil // Table not found - could be an error but we'll be permissive
}

func (a *AlterTableStatement) String() string {
	return "ALTER TABLE " + a.TableName + " " + a.Operation.String()
}

// MinifySQL takes raw SQL content and returns clean, normalized statements
func MinifySQL(sql string) []string {
	// Remove SQL comments
	sql = removeComments(sql)

	// Normalize whitespace
	sql = normalizeWhitespace(sql)

	// Split by semicolons
	statements := strings.Split(sql, ";")

	// Clean and filter statements
	var result []string
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt != "" {
			result = append(result, stmt)
		}
	}

	return result
}

// removeComments removes both -- and /* */ style comments from SQL
func removeComments(sql string) string {
	// Remove -- comments (single line)
	lines := strings.Split(sql, "\n")
	var cleanLines []string

	for _, line := range lines {
		// Find -- comment start (but not inside quotes)
		inQuote := false
		quoteChar := byte(0)
		commentPos := -1

		for i := 0; i < len(line); i++ {
			char := line[i]

			if !inQuote && (char == '\'' || char == '"') {
				inQuote = true
				quoteChar = char
			} else if inQuote && char == quoteChar {
				inQuote = false
			} else if !inQuote && i < len(line)-1 && line[i:i+2] == "--" {
				commentPos = i
				break
			}
		}

		if commentPos >= 0 {
			line = line[:commentPos]
		}

		cleanLines = append(cleanLines, line)
	}

	sql = strings.Join(cleanLines, "\n")

	// Remove /* */ comments (multi-line)
	blockCommentRegex := regexp.MustCompile(`/\*.*?\*/`)
	sql = blockCommentRegex.ReplaceAllString(sql, "")

	return sql
}

// normalizeWhitespace collapses multiple whitespace characters into single spaces
func normalizeWhitespace(sql string) string {
	// Replace multiple whitespace (including newlines) with single spaces
	whitespaceRegex := regexp.MustCompile(`\s+`)
	return strings.TrimSpace(whitespaceRegex.ReplaceAllString(sql, " "))
}

// ParseSQLStatement parses a single SQL statement into a SQLStatement interface
func ParseSQLStatement(sql string) (SQLStatement, error) {
	sql = strings.TrimSpace(strings.ToUpper(sql))

	if strings.HasPrefix(sql, "CREATE TABLE") {
		return parseCreateTable(sql)
	} else if strings.HasPrefix(sql, "ALTER TABLE") {
		return parseAlterTable(sql)
	}

	// Ignore other statements (CREATE TYPE, DROP TABLE, etc. for now)
	return nil, nil
}

// parseCreateTable parses CREATE TABLE statements
func parseCreateTable(sql string) (*CreateTableStatement, error) {
	// Extract table name
	tableNameRegex := regexp.MustCompile(`CREATE TABLE\s+([a-zA-Z0-9_]+)\s*\(`)
	matches := tableNameRegex.FindStringSubmatch(sql)
	if len(matches) < 2 {
		return nil, nil // Skip malformed statements
	}

	tableName := strings.ToLower(matches[1])

	// Extract column definitions - find content between parentheses
	parenStart := strings.Index(sql, "(")
	parenEnd := strings.LastIndex(sql, ")")
	if parenStart == -1 || parenEnd == -1 || parenEnd <= parenStart {
		return nil, nil
	}

	columnsStr := sql[parenStart+1 : parenEnd]
	columns := parseColumnDefinitions(columnsStr)

	return &CreateTableStatement{
		TableName: tableName,
		Columns:   columns,
	}, nil
}

// parseAlterTable parses ALTER TABLE statements
func parseAlterTable(sql string) (*AlterTableStatement, error) {
	// Extract table name
	tableNameRegex := regexp.MustCompile(`ALTER TABLE\s+([a-zA-Z0-9_]+)\s+(.+)`)
	matches := tableNameRegex.FindStringSubmatch(sql)
	if len(matches) < 3 {
		return nil, nil
	}

	tableName := strings.ToLower(matches[1])
	operation := strings.TrimSpace(matches[2])

	var op AlterOperation

	if strings.HasPrefix(operation, "ADD COLUMN") {
		op = parseAddColumn(operation)
	} else if strings.HasPrefix(operation, "DROP COLUMN") {
		op = parseDropColumn(operation)
	} else if strings.HasPrefix(operation, "ALTER COLUMN") && strings.Contains(operation, "TYPE") {
		op = parseAlterColumnType(operation)
	}

	if op == nil {
		return nil, nil // Unsupported operation
	}

	return &AlterTableStatement{
		TableName: tableName,
		Operation: op,
	}, nil
}

// parseColumnDefinitions parses the column definitions inside CREATE TABLE
func parseColumnDefinitions(columnsStr string) []ColumnDefinition {
	var columns []ColumnDefinition

	// Split by commas, but be careful about commas inside types like DECIMAL(10, 2)
	parts := smartSplitColumns(columnsStr)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || isConstraint(part) {
			continue // Skip empty parts and constraints
		}

		col := parseColumnDefinition(part)
		if col.Name != "" {
			columns = append(columns, col)
		}
	}

	return columns
}

// smartSplitColumns splits column definitions by comma, handling parentheses properly
func smartSplitColumns(s string) []string {
	var parts []string
	var current strings.Builder
	parenDepth := 0

	for _, char := range s {
		if char == '(' {
			parenDepth++
		} else if char == ')' {
			parenDepth--
		} else if char == ',' && parenDepth == 0 {
			parts = append(parts, current.String())
			current.Reset()
			continue
		}

		current.WriteRune(char)
	}

	// Add the last part
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// isConstraint checks if a part is a table constraint rather than a column
func isConstraint(part string) bool {
	part = strings.ToUpper(strings.TrimSpace(part))
	return strings.HasPrefix(part, "PRIMARY KEY") ||
		strings.HasPrefix(part, "UNIQUE") ||
		strings.HasPrefix(part, "CONSTRAINT") ||
		strings.HasPrefix(part, "FOREIGN KEY") ||
		strings.HasPrefix(part, "CHECK")
}

// parseColumnDefinition parses a single column definition
func parseColumnDefinition(def string) ColumnDefinition {
	parts := strings.Fields(def)
	if len(parts) < 2 {
		return ColumnDefinition{}
	}

	col := ColumnDefinition{
		Name: strings.ToLower(parts[0]),
		Type: extractTypeFromParts(parts[1:]),
	}

	// Check for constraints
	defUpper := strings.ToUpper(def)
	col.NotNull = strings.Contains(defUpper, "NOT NULL")
	col.PrimaryKey = strings.Contains(defUpper, "PRIMARY KEY")
	col.AutoIncrement = strings.Contains(defUpper, "SERIAL") || strings.Contains(defUpper, "AUTO_INCREMENT")

	return col
}

// extractTypeFromParts extracts the type from column definition parts, handling complex types
func extractTypeFromParts(parts []string) string {
	if len(parts) == 0 {
		return ""
	}

	// For types with parentheses (like DECIMAL(10,2)), find the complete type
	typeStr := parts[0]

	if strings.Contains(typeStr, "(") && !strings.Contains(typeStr, ")") {
		// Multi-part type, find the closing parenthesis
		for i := 1; i < len(parts); i++ {
			typeStr += " " + parts[i]
			if strings.Contains(parts[i], ")") {
				break
			}
		}
	}

	// Clean up and normalize the type
	typeStr = strings.ToLower(typeStr)
	typeStr = strings.ReplaceAll(typeStr, " ", "") // Remove spaces within type

	return typeStr
}

// parseAddColumn parses ADD COLUMN operations
func parseAddColumn(operation string) *AddColumnOperation {
	// Extract column definition after "ADD COLUMN"
	addColumnRegex := regexp.MustCompile(`ADD COLUMN\s+(.+)`)
	matches := addColumnRegex.FindStringSubmatch(operation)
	if len(matches) < 2 {
		return nil
	}

	colDef := parseColumnDefinition(matches[1])
	if colDef.Name == "" {
		return nil
	}

	return &AddColumnOperation{Column: colDef}
}

// parseDropColumn parses DROP COLUMN operations
func parseDropColumn(operation string) *DropColumnOperation {
	dropColumnRegex := regexp.MustCompile(`DROP COLUMN\s+(?:IF EXISTS\s+)?([a-zA-Z0-9_]+)`)
	matches := dropColumnRegex.FindStringSubmatch(operation)
	if len(matches) < 2 {
		return nil
	}

	return &DropColumnOperation{ColumnName: strings.ToLower(matches[1])}
}

// parseAlterColumnType parses ALTER COLUMN TYPE operations
func parseAlterColumnType(operation string) *AlterColumnTypeOperation {
	alterColumnRegex := regexp.MustCompile(`ALTER COLUMN\s+([a-zA-Z0-9_]+)\s+TYPE\s+(.+)`)
	matches := alterColumnRegex.FindStringSubmatch(operation)
	if len(matches) < 3 {
		return nil
	}

	columnName := strings.ToLower(matches[1])
	newType := strings.ToLower(strings.TrimSpace(matches[2]))

	return &AlterColumnTypeOperation{
		ColumnName: columnName,
		NewType:    newType,
	}
}

// ApplyMigrationsFromDir reads and applies all migrations from a directory
func ApplyMigrationsFromDir(ctx context.Context, dir string) (*Schema, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var migrationFiles []string
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".sql") {
			migrationFiles = append(migrationFiles, f.Name())
		}
	}

	// Sort files to apply in chronological order
	sort.Strings(migrationFiles)

	schema := &Schema{
		Models: make([]*Model, 0),
		Enums:  make([]*Enum, 0),
	}

	for _, fname := range migrationFiles {
		if err := applyMigrationFile(schema, dir+"/"+fname); err != nil {
			return nil, err
		}
	}

	return schema, nil
}

// applyMigrationFile applies a single migration file to the schema
func applyMigrationFile(schema *Schema, filepath string) error {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}

	sql := string(content)

	// Extract only the "UP" section of goose migrations
	upStart := strings.Index(sql, "-- +goose Up")
	downStart := strings.Index(sql, "-- +goose Down")

	if upStart >= 0 {
		if downStart > upStart {
			sql = sql[upStart:downStart]
		} else {
			sql = sql[upStart:]
		}
	}

	// Minify and parse statements
	statements := MinifySQL(sql)

	for _, stmt := range statements {
		sqlStmt, err := ParseSQLStatement(stmt)
		if err != nil {
			continue // Skip malformed statements
		}

		if sqlStmt != nil {
			if err := sqlStmt.Apply(schema); err != nil {
				return err
			}
		}
	}

	return nil
}
