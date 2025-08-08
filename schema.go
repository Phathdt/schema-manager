package main

import (
	"errors"
	"io/ioutil"
	"os"
	"regexp"
	"sort"
	"strings"
)

type PrismaSchema struct {
	Datasource *Datasource
	Generator  *Generator
	Models     map[string]*Model
	Enums      map[string]*Enum
}

type Datasource struct {
	Name     string
	Provider string
	URL      string
}

type Generator struct {
	Provider string
	Output   string
}

type Model struct {
	Name       string
	TableName  string
	Fields     []*Field
	Attributes []*ModelAttribute
}

type Field struct {
	Name       string
	ColumnName string
	Type       string
	Attributes []*FieldAttribute
	IsOptional bool
	IsArray    bool
}

type FieldAttribute struct {
	Name string
	Args []string
}

type ModelAttribute struct {
	Name string
	Args []string
}

type Enum struct {
	Name   string
	Values []string
}

type PrismaDiff struct {
	ModelsAdded   []*Model
	ModelsRemoved []*Model
}

func ParsePrismaSchema(path string) (*PrismaSchema, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content := string(b)
	lines := strings.Split(content, "\n")
	ps := &PrismaSchema{Models: map[string]*Model{}, Enums: map[string]*Enum{}}
	var currentModel *Model
	var currentEnum *Enum
	for _, line := range lines {
		l := strings.TrimSpace(line)
		if l == "" || strings.HasPrefix(l, "//") {
			continue
		}
		if strings.HasPrefix(l, "datasource ") {
			ps.Datasource = &Datasource{}
			continue
		}
		if ps.Datasource != nil && strings.HasPrefix(l, "provider") {
			ps.Datasource.Provider = parseStringValue(l)
			continue
		}
		if ps.Datasource != nil && strings.HasPrefix(l, "url") {
			ps.Datasource.URL = parseStringValue(l)
			continue
		}
		if strings.HasPrefix(l, "generator ") {
			ps.Generator = &Generator{}
			continue
		}
		if ps.Generator != nil && strings.HasPrefix(l, "provider") {
			ps.Generator.Provider = parseStringValue(l)
			continue
		}
		if ps.Generator != nil && strings.HasPrefix(l, "output") {
			ps.Generator.Output = parseStringValue(l)
			continue
		}
		if strings.HasPrefix(l, "model ") {
			name := strings.Fields(l)[1]
			currentModel = &Model{Name: name, TableName: name}
			ps.Models[name] = currentModel
			continue
		}
		if strings.HasPrefix(l, "enum ") {
			name := strings.Fields(l)[1]
			currentEnum = &Enum{Name: name}
			ps.Enums[name] = currentEnum
			continue
		}
		if currentModel != nil && l == "}" {
			currentModel = nil
			continue
		}
		if currentEnum != nil && l == "}" {
			currentEnum = nil
			continue
		}
		if currentModel != nil {
			if strings.HasPrefix(l, "@@") {
				attr := parseModelAttribute(l)
				currentModel.Attributes = append(currentModel.Attributes, attr)
				if attr.Name == "map" && len(attr.Args) > 0 {
					currentModel.TableName = strings.Trim(attr.Args[0], "\"")
				}
				continue
			}
			f := parseField(l)
			if f != nil {
				currentModel.Fields = append(currentModel.Fields, f)
			}
			continue
		}
		if currentEnum != nil {
			if !strings.HasPrefix(l, "enum ") && l != "{" && l != "}" {
				currentEnum.Values = append(currentEnum.Values, l)
			}
			continue
		}
	}
	if ps.Datasource == nil {
		return nil, errors.New("missing datasource block")
	}
	return ps, nil
}

func ValidatePrismaSchema(schema *PrismaSchema) error {
	if schema == nil {
		return errors.New("schema is nil")
	}
	if schema.Datasource == nil {
		return errors.New("missing datasource block")
	}
	if schema.Generator == nil {
		return errors.New("missing generator block")
	}
	if len(schema.Models) == 0 {
		return errors.New("no model defined")
	}
	for name, model := range schema.Models {
		hasId := false
		for _, field := range model.Fields {
			for _, attr := range field.Attributes {
				if attr.Name == "id" {
					hasId = true
					break
				}
			}
		}
		for _, attr := range model.Attributes {
			if attr.Name == "id" {
				hasId = true
				break
			}
		}
		if !hasId {
			return errors.New("model " + name + " must have at least one @id field or @@id attribute")
		}
	}
	return nil
}

func DiffPrismaSchemas(current, target *PrismaSchema) *PrismaDiff {
	added := []*Model{}
	removed := []*Model{}
	for name, tModel := range target.Models {
		if _, ok := current.Models[name]; !ok {
			added = append(added, tModel)
		}
	}
	for name, cModel := range current.Models {
		if _, ok := target.Models[name]; !ok {
			removed = append(removed, cModel)
		}
	}
	return &PrismaDiff{ModelsAdded: added, ModelsRemoved: removed}
}

func ParseCurrentSchemaFromMigrations(migrationsDir string) (*PrismaSchema, error) {
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return nil, err
	}
	var migrationFiles []string
	for _, f := range files {
		if !f.IsDir() {
			migrationFiles = append(migrationFiles, f.Name())
		}
	}
	sort.Strings(migrationFiles)
	ps := &PrismaSchema{Models: map[string]*Model{}, Enums: map[string]*Enum{}}
	tableRe := regexp.MustCompile(`(?i)CREATE TABLE ([a-zA-Z0-9_]+) \(([^;]*)\);`)
	colRe := regexp.MustCompile(`(?m)^\s*([a-zA-Z0-9_]+) ([^,]+)`) // name type ...
	for _, fname := range migrationFiles {
		b, err := os.ReadFile(migrationsDir + "/" + fname)
		if err != nil {
			return nil, err
		}
		content := string(b)
		upStart := strings.Index(content, "-- +goose Up")
		if upStart < 0 {
			continue
		}
		upBlock := content[upStart:]
		stmts := strings.Split(upBlock, "-- +goose StatementBegin")
		for _, stmtBlock := range stmts {
			if !strings.Contains(stmtBlock, "CREATE TABLE") {
				continue
			}
			matches := tableRe.FindAllStringSubmatch(stmtBlock, -1)
			for _, mtab := range matches {
				table := mtab[1]
				colsBlock := mtab[2]
				model := &Model{Name: table, TableName: table}
				lines := strings.Split(colsBlock, ",")
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if line == "" {
						continue
					}
					colMatch := colRe.FindStringSubmatch(line)
					if len(colMatch) < 3 {
						continue
					}
					fname := colMatch[1]
					ftype := strings.Fields(colMatch[2])[0]
					model.Fields = append(model.Fields, &Field{Name: fname, ColumnName: fname, Type: ftype})
				}
				ps.Models[table] = model
			}
		}
	}
	return ps, nil
}

func parseIndexFields(args []string, fields []*Field) ([]string, string) {
	var cols []string
	var idxNameParts []string
	for _, a := range args {
		s := strings.Trim(a, "[] \"'")
		if s == "" {
			continue
		}
		for _, f := range fields {
			if f.Name == s {
				cols = append(cols, f.ColumnName)
				idxNameParts = append(idxNameParts, f.ColumnName)
			}
		}
	}
	return cols, strings.Join(idxNameParts, "_")
}

func GenerateMigrationSQL(diff *PrismaDiff) (string, string) {
	var upBlocks []string
	var downBlocks []string
	for _, m := range diff.ModelsAdded {
		table := m.TableName
		cols := []string{}
		var upStmts []string
		var downStmts []string
		for _, f := range m.Fields {
			isSerial := false
			col := f.ColumnName + " " + goTypeToSQLTypeWithAttributes(f.Type, f.Attributes)
			isUnique := false
			isPrimary := false
			for _, attr := range f.Attributes {
				if attr.Name == "id" {
					isPrimary = true
				}
				if attr.Name == "unique" {
					col += " UNIQUE"
					isUnique = true
				}
				if attr.Name == "default" && len(attr.Args) > 0 {
					if f.Type == "Int" && attr.Args[0] == "autoincrement()" && isPrimary {
						col = f.ColumnName + " SERIAL PRIMARY KEY"
						isSerial = true
					} else {
						col += " DEFAULT " + parseDefaultValue(attr.Args[0], f.Type)
					}
				}
			}
			if isPrimary && !isSerial {
				col += " PRIMARY KEY"
			}
			cols = append(cols, col)
			if isUnique {
				idxName := "idx_uniq_" + table + "_" + f.ColumnName
				upStmts = append(upStmts, "CREATE UNIQUE INDEX "+idxName+" ON "+table+"("+f.ColumnName+");")
				downStmts = append([]string{"DROP INDEX IF EXISTS " + idxName + ";"}, downStmts...)
			}
		}
		createTable := "CREATE TABLE " + table + " (\n  " + strings.Join(cols, ",\n  ") + "\n);"
		upStmts = append([]string{createTable}, upStmts...)
		for _, attr := range m.Attributes {
			if (attr.Name == "unique" || attr.Name == "index") && len(attr.Args) > 0 {
				fields, idxNamePart := parseIndexFields(attr.Args, m.Fields)
				if len(fields) == 0 || idxNamePart == "" {
					continue
				}
				if attr.Name == "unique" {
					idxName := "idx_uniq_" + table + "_" + idxNamePart
					upStmts = append(
						upStmts,
						"CREATE UNIQUE INDEX "+idxName+" ON "+table+"("+strings.Join(fields, ", ")+");",
					)
					downStmts = append([]string{"DROP INDEX IF EXISTS " + idxName + ";"}, downStmts...)
				} else {
					idxName := "idx_" + table + "_" + idxNamePart
					upStmts = append(upStmts, "CREATE INDEX "+idxName+" ON "+table+"("+strings.Join(fields, ", ")+");")
					downStmts = append([]string{"DROP INDEX IF EXISTS " + idxName + ";"}, downStmts...)
				}
			}
		}
		downStmts = append(downStmts, "DROP TABLE IF EXISTS "+table+";")
		upBlocks = append(upBlocks, wrapGoose(strings.Join(upStmts, "\n\n")))
		downBlocks = append([]string{wrapGoose(strings.Join(downStmts, "\n\n"))}, downBlocks...)
	}
	for _, m := range diff.ModelsRemoved {
		table := m.TableName
		cols := []string{}
		var upStmts []string
		var downStmts []string
		for _, f := range m.Fields {
			isSerial := false
			col := f.ColumnName + " " + goTypeToSQLTypeWithAttributes(f.Type, f.Attributes)
			isUnique := false
			isPrimary := false
			for _, attr := range f.Attributes {
				if attr.Name == "id" {
					isPrimary = true
				}
				if attr.Name == "unique" {
					col += " UNIQUE"
					isUnique = true
				}
				if attr.Name == "default" && len(attr.Args) > 0 {
					if f.Type == "Int" && attr.Args[0] == "autoincrement()" && isPrimary {
						col = f.ColumnName + " SERIAL PRIMARY KEY"
						isSerial = true
					} else {
						col += " DEFAULT " + parseDefaultValue(attr.Args[0], f.Type)
					}
				}
			}
			if isPrimary && !isSerial {
				col += " PRIMARY KEY"
			}
			cols = append(cols, col)
			if isUnique {
				idxName := "idx_uniq_" + table + "_" + f.ColumnName
				upStmts = append(upStmts, "CREATE UNIQUE INDEX "+idxName+" ON "+table+"("+f.ColumnName+");")
				downStmts = append([]string{"DROP INDEX IF EXISTS " + idxName + ";"}, downStmts...)
			}
		}
		createTable := "CREATE TABLE " + table + " (\n  " + strings.Join(cols, ",\n  ") + "\n);"
		upStmts = append([]string{createTable}, upStmts...)
		for _, attr := range m.Attributes {
			if (attr.Name == "unique" || attr.Name == "index") && len(attr.Args) > 0 {
				fields, idxNamePart := parseIndexFields(attr.Args, m.Fields)
				if len(fields) == 0 || idxNamePart == "" {
					continue
				}
				if attr.Name == "unique" {
					idxName := "idx_uniq_" + table + "_" + idxNamePart
					upStmts = append(
						upStmts,
						"CREATE UNIQUE INDEX "+idxName+" ON "+table+"("+strings.Join(fields, ", ")+");",
					)
					downStmts = append([]string{"DROP INDEX IF EXISTS " + idxName + ";"}, downStmts...)
				} else {
					idxName := "idx_" + table + "_" + idxNamePart
					upStmts = append(upStmts, "CREATE INDEX "+idxName+" ON "+table+"("+strings.Join(fields, ", ")+");")
					downStmts = append([]string{"DROP INDEX IF EXISTS " + idxName + ";"}, downStmts...)
				}
			}
		}
		downStmts = append(downStmts, "DROP TABLE IF EXISTS "+table+";")
		upBlocks = append([]string{wrapGoose(strings.Join(upStmts, "\n\n"))}, upBlocks...)
		downBlocks = append(downBlocks, wrapGoose(strings.Join(downStmts, "\n\n")))
	}
	return strings.Join(upBlocks, "\n\n"), strings.Join(downBlocks, "\n\n")
}

func wrapGoose(sql string) string {
	return "-- +goose StatementBegin\n" + sql + "\n-- +goose StatementEnd"
}

func goTypeToSQLType(t string) string {
	switch t {
	case "Int":
		return "INTEGER"
	case "BigInt":
		return "BIGINT"
	case "String":
		return "TEXT"
	case "DateTime":
		return "TIMESTAMP"
	case "Boolean":
		return "BOOLEAN"
	case "Float":
		return "FLOAT"
	default:
		return "TEXT"
	}
}

func goTypeToSQLTypeWithAttributes(t string, attributes []*FieldAttribute) string {
	// Check for @db type attributes first
	for _, attr := range attributes {
		if strings.HasPrefix(attr.Name, "db.") {
			dbType := strings.TrimPrefix(attr.Name, "db.")
			if dbType == "VarChar" && len(attr.Args) > 0 {
				return "VARCHAR(" + attr.Args[0] + ")"
			}
			if dbType == "Text" {
				return "TEXT"
			}
		}
	}

	return goTypeToSQLType(t)
}

func parseStringValue(line string) string {
	idx := strings.Index(line, "=")
	if idx < 0 {
		return ""
	}
	v := strings.TrimSpace(line[idx+1:])
	v = strings.Trim(v, "\"")
	return v
}

func parseField(line string) *Field {
	if strings.HasPrefix(line, "@@") || line == "{" || line == "}" {
		return nil
	}
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return nil
	}
	f := &Field{Name: parts[0], ColumnName: parts[0], Type: parts[1]}
	for _, p := range parts[2:] {
		if strings.HasPrefix(p, "@") {
			attr := parseFieldAttribute(p)
			f.Attributes = append(f.Attributes, attr)
			if attr.Name == "map" && len(attr.Args) > 0 {
				f.ColumnName = strings.Trim(attr.Args[0], "\"")
			}
		}
	}
	if strings.HasSuffix(f.Type, "?") {
		f.IsOptional = true
		f.Type = strings.TrimSuffix(f.Type, "?")
	}
	if strings.HasSuffix(f.Type, "[]") {
		f.IsArray = true
		f.Type = strings.TrimSuffix(f.Type, "[]")
	}
	return f
}

func parseFieldAttribute(token string) *FieldAttribute {
	token = strings.TrimPrefix(token, "@")
	name := token
	var args []string
	if i := strings.Index(token, "("); i >= 0 {
		name = token[:i]
		argsStr := strings.TrimSuffix(token[i+1:], ")")
		args = strings.Split(argsStr, ",")
		for i := range args {
			args[i] = strings.TrimSpace(args[i])
		}
	}
	return &FieldAttribute{Name: name, Args: args}
}

func parseModelAttribute(line string) *ModelAttribute {
	l := strings.TrimPrefix(line, "@@")
	l = strings.TrimSpace(l)
	name := l
	var args []string
	if i := strings.Index(l, "("); i >= 0 {
		name = l[:i]
		argsStr := strings.TrimSuffix(l[i+1:], ")")
		args = strings.Split(argsStr, ",")
		for i := range args {
			args[i] = strings.TrimSpace(args[i])
		}
	}
	return &ModelAttribute{Name: name, Args: args}
}

func parseDefaultValue(val, typ string) string {
	v := strings.Trim(val, "\"")
	switch typ {
	case "String":
		return "'" + v + "'"
	case "DateTime":
		if v == "now()" {
			return "CURRENT_TIMESTAMP"
		}
		return v
	case "Boolean":
		if v == "true" {
			return "TRUE"
		}
		return "FALSE"
	default:
		return v
	}
}
