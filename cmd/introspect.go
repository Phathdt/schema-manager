package cmd

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/urfave/cli/v2"
)

type TableInfo struct {
	TableName   string
	Columns     []ColumnInfo
	Indexes     []IndexInfo
	Constraints []ConstraintInfo
}

type ColumnInfo struct {
	ColumnName      string
	DataType        string
	IsNullable      bool
	DefaultValue    sql.NullString
	IsAutoIncrement bool
	IsPrimaryKey    bool
	IsUnique        bool
	IsCompositePK   bool
}

type IndexInfo struct {
	IndexName  string
	ColumnName string
	IsUnique   bool
}

type ConstraintInfo struct {
	ConstraintName string
	ConstraintType string
	ColumnName     string
}

func IntrospectCommand() *cli.Command {
	return &cli.Command{
		Name:        "introspect",
		Usage:       "Import existing database structure into schema.prisma",
		Description: "Connect to existing database and generate schema.prisma file with conditional baseline migration",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output schema file path",
				Value:   "schema.prisma",
			},
		},
		Action: func(ctx *cli.Context) error {
			outputFile := ctx.String("output")
			return runIntrospect(outputFile)
		},
	}
}

func runIntrospect(outputFile string) error {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL environment variable is required")
	}

	db, err := connectWithSSLFallback(databaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	fmt.Println("✅ Connected to database successfully")

	tables, err := introspectDatabase(db)
	if err != nil {
		return fmt.Errorf("failed to introspect database: %w", err)
	}

	if len(tables) == 0 {
		fmt.Println("⚠️  No tables found in database")
		return nil
	}

	fmt.Printf("📊 Found %d tables in database\n", len(tables))

	schemaContent := generatePrismaSchema(tables)
	if err := writeSchemaFile(outputFile, schemaContent); err != nil {
		return fmt.Errorf("failed to write schema file: %w", err)
	}

	fmt.Printf("✅ Generated schema.prisma at %s\n", outputFile)

	migrationContent := generateBaselineMigration(tables)
	timestamp := time.Now().Format("20060102150405")
	migrationFile := fmt.Sprintf("migrations/%s_baseline_from_database.sql", timestamp)

	if err := createMigrationsDir(); err != nil {
		return fmt.Errorf("failed to create migrations directory: %w", err)
	}

	if err := writeMigrationFile(migrationFile, migrationContent); err != nil {
		return fmt.Errorf("failed to write migration file: %w", err)
	}

	fmt.Printf("✅ Generated baseline migration at %s\n", migrationFile)
	fmt.Println("🚀 Run 'goose up' to apply the baseline migration")

	return nil
}

func connectWithSSLFallback(databaseURL string) (*sql.DB, error) {
	// First, try to connect with the original URL
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		db.Close()

		// Check if it's an SSL-related error
		if strings.Contains(err.Error(), "SSL is not enabled") || strings.Contains(err.Error(), "ssl") {
			fmt.Println("⚠️  SSL connection failed, retrying with SSL disabled...")

			// Add sslmode=disable if not present
			fallbackURL := databaseURL
			if !strings.Contains(databaseURL, "sslmode=") {
				separator := "?"
				if strings.Contains(databaseURL, "?") {
					separator = "&"
				}
				fallbackURL = databaseURL + separator + "sslmode=disable"
			}

			// Try connecting with SSL disabled
			db, err = sql.Open("postgres", fallbackURL)
			if err != nil {
				return nil, err
			}

			if err := db.Ping(); err != nil {
				db.Close()
				return nil, fmt.Errorf(
					"connection failed even with SSL disabled. Please check your database connection settings: %w",
					err,
				)
			}

			fmt.Println("✅ Connected successfully with SSL disabled")
		} else {
			return nil, fmt.Errorf("database connection failed: %w", err)
		}
	}

	return db, nil
}

func introspectDatabase(db *sql.DB) ([]TableInfo, error) {
	query := `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = 'public'
		AND table_type = 'BASE TABLE'
		AND table_name != 'goose_db_version'
		ORDER BY table_name
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []TableInfo
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}

		table := TableInfo{TableName: tableName}

		columns, err := getTableColumns(db, tableName)
		if err != nil {
			return nil, fmt.Errorf("failed to get columns for table %s: %w", tableName, err)
		}
		table.Columns = columns

		indexes, err := getTableIndexes(db, tableName)
		if err != nil {
			return nil, fmt.Errorf("failed to get indexes for table %s: %w", tableName, err)
		}
		table.Indexes = indexes

		constraints, err := getTableConstraints(db, tableName)
		if err != nil {
			return nil, fmt.Errorf("failed to get constraints for table %s: %w", tableName, err)
		}
		table.Constraints = constraints

		// Get primary key columns for composite key detection
		primaryKeys, err := getTablePrimaryKeys(db, tableName)
		if err != nil {
			return nil, fmt.Errorf("failed to get primary keys for table %s: %w", tableName, err)
		}

		// Mark composite primary key flag
		for i := range table.Columns {
			if table.Columns[i].IsPrimaryKey {
				table.Columns[i].IsCompositePK = len(primaryKeys) > 1
			}
		}

		tables = append(tables, table)
	}

	return tables, nil
}

func getTableColumns(db *sql.DB, tableName string) ([]ColumnInfo, error) {
	query := `
		SELECT
			column_name,
			data_type,
			is_nullable,
			column_default,
			CASE
				WHEN column_default LIKE 'nextval%' THEN true
				ELSE false
			END as is_auto_increment
		FROM information_schema.columns
		WHERE table_name = $1
		AND table_schema = 'public'
		ORDER BY ordinal_position
	`

	rows, err := db.Query(query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		var isNullable string

		if err := rows.Scan(&col.ColumnName, &col.DataType, &isNullable, &col.DefaultValue, &col.IsAutoIncrement); err != nil {
			return nil, err
		}

		col.IsNullable = isNullable == "YES"

		isPK, err := isColumnPrimaryKey(db, tableName, col.ColumnName)
		if err != nil {
			return nil, err
		}
		col.IsPrimaryKey = isPK

		isUnique, err := isColumnUnique(db, tableName, col.ColumnName)
		if err != nil {
			return nil, err
		}
		col.IsUnique = isUnique

		columns = append(columns, col)
	}

	return columns, nil
}

func getTableIndexes(db *sql.DB, tableName string) ([]IndexInfo, error) {
	query := `
		SELECT
			i.indexname,
			a.attname,
			i.indexdef LIKE '%UNIQUE%' as is_unique
		FROM pg_indexes i
		JOIN pg_class c ON c.relname = i.tablename
		JOIN pg_index ix ON ix.indexrelid = (
			SELECT oid FROM pg_class WHERE relname = i.indexname
		)
		JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum = ANY(ix.indkey)
		WHERE i.tablename = $1
		AND i.schemaname = 'public'
		AND NOT ix.indisprimary
		ORDER BY i.indexname, a.attnum
	`

	rows, err := db.Query(query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []IndexInfo
	for rows.Next() {
		var idx IndexInfo
		if err := rows.Scan(&idx.IndexName, &idx.ColumnName, &idx.IsUnique); err != nil {
			return nil, err
		}
		indexes = append(indexes, idx)
	}

	return indexes, nil
}

func getTableConstraints(db *sql.DB, tableName string) ([]ConstraintInfo, error) {
	query := `
		SELECT
			tc.constraint_name,
			tc.constraint_type,
			ccu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.constraint_column_usage ccu
			ON tc.constraint_name = ccu.constraint_name
		WHERE tc.table_name = $1
		AND tc.table_schema = 'public'
		ORDER BY tc.constraint_name
	`

	rows, err := db.Query(query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var constraints []ConstraintInfo
	for rows.Next() {
		var constraint ConstraintInfo
		if err := rows.Scan(&constraint.ConstraintName, &constraint.ConstraintType, &constraint.ColumnName); err != nil {
			return nil, err
		}
		constraints = append(constraints, constraint)
	}

	return constraints, nil
}

func isColumnPrimaryKey(db *sql.DB, tableName, columnName string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.table_constraints tc
			JOIN information_schema.constraint_column_usage ccu
				ON tc.constraint_name = ccu.constraint_name
			WHERE tc.table_name = $1
			AND tc.constraint_type = 'PRIMARY KEY'
			AND ccu.column_name = $2
		)
	`

	var exists bool
	err := db.QueryRow(query, tableName, columnName).Scan(&exists)
	return exists, err
}

func isColumnUnique(db *sql.DB, tableName, columnName string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.table_constraints tc
			JOIN information_schema.constraint_column_usage ccu
				ON tc.constraint_name = ccu.constraint_name
			WHERE tc.table_name = $1
			AND tc.constraint_type = 'UNIQUE'
			AND ccu.column_name = $2
		)
	`

	var exists bool
	err := db.QueryRow(query, tableName, columnName).Scan(&exists)
	return exists, err
}

func getTablePrimaryKeys(db *sql.DB, tableName string) ([]string, error) {
	query := `
		SELECT ccu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.constraint_column_usage ccu
			ON tc.constraint_name = ccu.constraint_name
		WHERE tc.table_name = $1
		AND tc.constraint_type = 'PRIMARY KEY'
		ORDER BY ccu.column_name
	`

	rows, err := db.Query(query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var primaryKeys []string
	for rows.Next() {
		var columnName string
		if err := rows.Scan(&columnName); err != nil {
			return nil, err
		}
		primaryKeys = append(primaryKeys, columnName)
	}

	return primaryKeys, nil
}

func generatePrismaSchema(tables []TableInfo) string {
	var schema strings.Builder

	schema.WriteString(`datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

generator client {
  provider = "schema-manager"
  output   = "./migrations"
}

`)

	for _, table := range tables {
		schema.WriteString(fmt.Sprintf("model %s {\n", toPascalCase(table.TableName)))

		// Collect primary key fields for composite primary key
		var primaryKeyFields []string

		for _, col := range table.Columns {
			schema.WriteString(fmt.Sprintf("  %s", toCamelCase(col.ColumnName)))

			prismaType := mapDataTypeToPrisma(col.DataType)
			if col.IsNullable && !col.IsPrimaryKey {
				prismaType += "?"
			}
			schema.WriteString(fmt.Sprintf(" %s", prismaType))

			var attributes []string
			// Only add @id for single primary keys, not composite ones
			if col.IsPrimaryKey && !col.IsCompositePK {
				attributes = append(attributes, "@id")
			}
			if col.IsAutoIncrement {
				attributes = append(attributes, "@default(autoincrement())")
			}
			if col.IsUnique && !col.IsPrimaryKey {
				attributes = append(attributes, "@unique")
			}
			if col.ColumnName != toCamelCase(col.ColumnName) {
				attributes = append(attributes, fmt.Sprintf("@map(\"%s\")", col.ColumnName))
			}

			if len(attributes) > 0 {
				schema.WriteString(" " + strings.Join(attributes, " "))
			}

			schema.WriteString("\n")

			// Collect primary key fields for composite key
			if col.IsPrimaryKey {
				primaryKeyFields = append(primaryKeyFields, toCamelCase(col.ColumnName))
			}
		}

		schema.WriteString("\n")

		// Add composite primary key if there are multiple primary key fields
		if len(primaryKeyFields) > 1 {
			schema.WriteString(fmt.Sprintf("  @@id([%s])\n", strings.Join(primaryKeyFields, ", ")))
		}

		schema.WriteString(fmt.Sprintf("  @@map(\"%s\")\n", table.TableName))
		schema.WriteString("}\n\n")
	}

	return schema.String()
}

func generateBaselineMigration(tables []TableInfo) string {
	var migration strings.Builder

	migration.WriteString("-- +goose Up\n")
	migration.WriteString("-- +goose StatementBegin\n")
	migration.WriteString("-- Baseline migration from existing database\n")
	migration.WriteString("-- All tables use conditional creation (IF NOT EXISTS)\n\n")

	for _, table := range tables {
		migration.WriteString("DO $$\n")
		migration.WriteString("BEGIN\n")
		migration.WriteString(
			fmt.Sprintf(
				"    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = '%s') THEN\n",
				table.TableName,
			),
		)
		migration.WriteString(fmt.Sprintf("        CREATE TABLE %s (\n", table.TableName))

		var columnDefs []string
		for _, col := range table.Columns {
			colDef := fmt.Sprintf("            %s %s", col.ColumnName, mapDataTypeToSQL(col.DataType))

			if col.IsPrimaryKey {
				colDef += " PRIMARY KEY"
			}
			if col.IsAutoIncrement {
				colDef = strings.Replace(colDef, mapDataTypeToSQL(col.DataType), "SERIAL", 1)
			}
			if !col.IsNullable && !col.IsPrimaryKey {
				colDef += " NOT NULL"
			}
			if col.IsUnique && !col.IsPrimaryKey {
				colDef += " UNIQUE"
			}
			if col.DefaultValue.Valid && !col.IsAutoIncrement {
				colDef += fmt.Sprintf(" DEFAULT %s", col.DefaultValue.String)
			}

			columnDefs = append(columnDefs, colDef)
		}

		migration.WriteString(strings.Join(columnDefs, ",\n"))
		migration.WriteString("\n        );\n")
		migration.WriteString("    END IF;\n")
		migration.WriteString("END $$;\n\n")
	}

	migration.WriteString("-- +goose StatementEnd\n\n")
	migration.WriteString("-- +goose Down\n")
	migration.WriteString("-- +goose StatementBegin\n")

	for i := len(tables) - 1; i >= 0; i-- {
		migration.WriteString(fmt.Sprintf("DROP TABLE IF EXISTS %s;\n", tables[i].TableName))
	}

	migration.WriteString("-- +goose StatementEnd\n")

	return migration.String()
}

func mapDataTypeToPrisma(sqlType string) string {
	switch strings.ToLower(sqlType) {
	case "integer", "int4", "serial":
		return "Int"
	case "bigint", "int8", "bigserial":
		return "BigInt"
	case "varchar", "text", "char", "character varying":
		return "String"
	case "boolean", "bool":
		return "Boolean"
	case "timestamp", "timestamptz", "timestamp with time zone", "timestamp without time zone":
		return "DateTime"
	case "date":
		return "DateTime"
	case "decimal", "numeric":
		return "Decimal"
	case "real", "float4":
		return "Float"
	case "double precision", "float8":
		return "Float"
	case "json", "jsonb":
		return "Json"
	case "uuid":
		return "String"
	default:
		return "String"
	}
}

func mapDataTypeToSQL(sqlType string) string {
	switch strings.ToLower(sqlType) {
	case "integer", "int4":
		return "INTEGER"
	case "bigint", "int8":
		return "BIGINT"
	case "varchar", "character varying":
		return "VARCHAR(255)"
	case "text":
		return "TEXT"
	case "boolean", "bool":
		return "BOOLEAN"
	case "timestamp", "timestamp without time zone":
		return "TIMESTAMP"
	case "timestamptz", "timestamp with time zone":
		return "TIMESTAMP WITH TIME ZONE"
	case "date":
		return "DATE"
	case "decimal", "numeric":
		return "DECIMAL"
	case "real", "float4":
		return "REAL"
	case "double precision", "float8":
		return "DOUBLE PRECISION"
	case "json":
		return "JSON"
	case "jsonb":
		return "JSONB"
	case "uuid":
		return "UUID"
	default:
		return "TEXT"
	}
}

func toPascalCase(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		parts[i] = strings.Title(part)
	}
	result := strings.Join(parts, "")
	return singularize(result)
}

func singularize(s string) string {
	if len(s) == 0 {
		return s
	}

	// Handle common plural patterns
	switch {
	case strings.HasSuffix(s, "ies"):
		// categories -> category, companies -> company
		return s[:len(s)-3] + "y"
	case strings.HasSuffix(s, "ses"):
		// addresses -> address, processes -> process
		return s[:len(s)-2]
	case strings.HasSuffix(s, "s") && !strings.HasSuffix(s, "ss"):
		// users -> user, wallets -> wallet (but not address -> addres)
		return s[:len(s)-1]
	default:
		return s
	}
}

func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += strings.Title(parts[i])
	}
	return result
}

func writeSchemaFile(filename, content string) error {
	return os.WriteFile(filename, []byte(content), 0o644)
}

func writeMigrationFile(filename, content string) error {
	return os.WriteFile(filename, []byte(content), 0o644)
}

func createMigrationsDir() error {
	dir := filepath.Dir("migrations/")
	return os.MkdirAll(dir, 0o755)
}
