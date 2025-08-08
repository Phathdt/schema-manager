package schema

import (
	"context"
	"os"
	"regexp"
	"sort"
	"strings"
)

func ParseMigrationsToSchema(ctx context.Context, dir string) (*Schema, error) {
	files, err := os.ReadDir(dir)
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
	schema := &Schema{}

	// Track tables and enums
	tables := make(map[string]*Model)
	enums := make(map[string]*Enum)

	tableRe := regexp.MustCompile(`(?is)CREATE TABLE ([a-zA-Z0-9_]+) \((.*?)\);`)
	enumRe := regexp.MustCompile(`(?i)CREATE TYPE ([a-zA-Z0-9_]+) AS ENUM \(([^;]*)\);`)
	dropTableRe := regexp.MustCompile(`(?i)DROP TABLE IF EXISTS ([a-zA-Z0-9_]+);`)
	dropTypeRe := regexp.MustCompile(`(?i)DROP TYPE IF EXISTS ([a-zA-Z0-9_]+);`)
	addColumnRe := regexp.MustCompile(`(?i)ALTER TABLE ([a-zA-Z0-9_]+) ADD COLUMN ([a-zA-Z0-9_]+) ([^;]+);`)
	dropColumnRe := regexp.MustCompile(`(?i)ALTER TABLE ([a-zA-Z0-9_]+) DROP COLUMN IF EXISTS ([a-zA-Z0-9_]+);`)
	colRe := regexp.MustCompile(`(?m)^\s*([a-zA-Z0-9_]+) ([^,\n]+)`) // name type ...

	for _, fname := range migrationFiles {
		b, err := os.ReadFile(dir + "/" + fname)
		if err != nil {
			return nil, err
		}
		content := string(b)
		upStart := strings.Index(content, "-- +goose Up")
		downStart := strings.Index(content, "-- +goose Down")

		if upStart < 0 {
			continue
		}

		var upBlock string
		if downStart > upStart {
			upBlock = content[upStart:downStart]
		} else {
			upBlock = content[upStart:]
		}

		stmts := strings.Split(upBlock, "-- +goose StatementBegin")
		for _, stmtBlock := range stmts {
			// Handle CREATE TABLE
			if strings.Contains(stmtBlock, "CREATE TABLE") {
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
						// Skip constraints and other non-column definitions
						if strings.HasPrefix(strings.ToUpper(line), "PRIMARY KEY") ||
							strings.HasPrefix(strings.ToUpper(line), "UNIQUE") ||
							strings.HasPrefix(strings.ToUpper(line), "CONSTRAINT") ||
							strings.HasPrefix(strings.ToUpper(line), "FOREIGN KEY") {
							continue
						}
						colMatch := colRe.FindStringSubmatch(line)
						if len(colMatch) < 3 {
							continue
						}
						fname := colMatch[1]
						ftype := strings.Fields(colMatch[2])[0]

						// Check if field is nullable by looking for NOT NULL constraint or PRIMARY KEY
						// In SQL, columns are nullable by default unless NOT NULL is specified
						// PRIMARY KEY also implies NOT NULL
						columnDef := strings.ToUpper(colMatch[2])
						isOptional := !strings.Contains(columnDef, "NOT NULL") &&
							!strings.Contains(columnDef, "PRIMARY KEY")

						model.Fields = append(model.Fields, &Field{
							Name:       fname,
							ColumnName: fname,
							Type:       ftype,
							IsOptional: isOptional,
						})
					}
					tables[table] = model
				}
			}

			// Handle CREATE TYPE (enum)
			if strings.Contains(stmtBlock, "CREATE TYPE") {
				matches := enumRe.FindAllStringSubmatch(stmtBlock, -1)
				for _, match := range matches {
					enumName := match[1]
					valuesStr := match[2]
					enum := &Enum{Name: enumName}
					// Parse enum values
					values := strings.Split(valuesStr, ",")
					for _, v := range values {
						v = strings.TrimSpace(v)
						v = strings.Trim(v, "'\"")
						if v != "" {
							enum.Values = append(enum.Values, v)
						}
					}
					enums[enumName] = enum
				}
			}

			// Handle DROP TABLE
			if strings.Contains(stmtBlock, "DROP TABLE") {
				matches := dropTableRe.FindAllStringSubmatch(stmtBlock, -1)
				for _, match := range matches {
					table := match[1]
					delete(tables, table)
				}
			}

			// Handle DROP TYPE
			if strings.Contains(stmtBlock, "DROP TYPE") {
				matches := dropTypeRe.FindAllStringSubmatch(stmtBlock, -1)
				for _, match := range matches {
					enumName := match[1]
					delete(enums, enumName)
				}
			}

			// Handle ALTER TABLE ADD COLUMN
			if strings.Contains(stmtBlock, "ALTER TABLE") && strings.Contains(stmtBlock, "ADD COLUMN") {
				matches := addColumnRe.FindAllStringSubmatch(stmtBlock, -1)
				for _, match := range matches {
					tableName := match[1]
					columnName := match[2]
					columnDef := match[3]
					columnType := strings.Fields(columnDef)[0] // Get the first word as type

					// Check if field is nullable by looking for NOT NULL constraint or PRIMARY KEY
					columnDefUpper := strings.ToUpper(columnDef)
					isOptional := !strings.Contains(columnDefUpper, "NOT NULL") &&
						!strings.Contains(columnDefUpper, "PRIMARY KEY")

					// Find or create the model for this table
					if model, exists := tables[tableName]; exists {
						// Add the new field to the existing model
						model.Fields = append(model.Fields, &Field{
							Name:       columnName,
							ColumnName: columnName,
							Type:       columnType,
							IsOptional: isOptional,
						})
					}
				}
			}

			// Handle ALTER TABLE DROP COLUMN
			if strings.Contains(stmtBlock, "ALTER TABLE") && strings.Contains(stmtBlock, "DROP COLUMN") {
				matches := dropColumnRe.FindAllStringSubmatch(stmtBlock, -1)
				for _, match := range matches {
					tableName := match[1]
					columnName := match[2]

					// Find the model and remove the field
					if model, exists := tables[tableName]; exists {
						newFields := []*Field{}
						for _, field := range model.Fields {
							if field.ColumnName != columnName {
								newFields = append(newFields, field)
							}
						}
						model.Fields = newFields
					}
				}
			}
		}
	}

	// Convert maps to slices
	for _, model := range tables {
		schema.Models = append(schema.Models, model)
	}
	for _, enum := range enums {
		schema.Enums = append(schema.Enums, enum)
	}

	return schema, nil
}
