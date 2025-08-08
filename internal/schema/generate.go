package schema

import (
	"fmt"
	"log"
	"strings"
)

func GenerateMigrationSQL(diff *SchemaDiff) string {
	var stmts []string

	// Generate ENUMs first
	for _, e := range diff.EnumsAdded {
		enumStmt := generateEnumSQL(e)
		stmts = append(stmts, wrapGooseStatement(enumStmt))
	}

	// Handle field additions
	for _, fieldChange := range diff.FieldsAdded {
		stmt := generateAddColumnSQL(fieldChange)
		if stmt != "" {
			stmts = append(stmts, wrapGooseStatement(stmt))
		}
	}

	// Handle field removals
	for _, fieldChange := range diff.FieldsRemoved {
		stmt := generateDropColumnSQL(fieldChange)
		if stmt != "" {
			warning := fmt.Sprintf("IRREVERSIBLE: Dropping column %s.%s - all data in this column will be lost!",
				fieldChange.ModelName, fieldChange.Field.ColumnName)
			stmts = append(stmts, wrapGooseStatementWithWarning(stmt, warning))
		}
	}

	// Handle field modifications
	for _, fieldChange := range diff.FieldsModified {
		stmt, warning := generateModifyColumnSQLWithWarning(fieldChange)
		if stmt != "" {
			if warning != "" {
				stmts = append(stmts, wrapGooseStatementWithWarning(stmt, warning))
			} else {
				stmts = append(stmts, wrapGooseStatement(stmt))
			}
		}
	}

	for _, m := range diff.ModelsAdded {
		cols := []string{}
		pkCols := []string{}
		indexes := []string{}
		uniqueIndexes := []string{}
		foreignKeys := []string{}

		// Check for composite primary key from model attributes
		compositePK := []string{}
		for _, attr := range m.Attributes {
			if attr.Name == "id" {
				compositePK = attr.Args
				break
			}
		}

		for _, f := range m.Fields {
			// Skip relation fields that don't have actual columns (array types and fields with @relation)
			if f.IsArray {
				continue
			}
			hasRelationAttr := false
			for _, attr := range f.Attributes {
				if attr.Name == "relation" {
					hasRelationAttr = true
					break
				}
			}
			if hasRelationAttr {
				continue
			}

			isPrimary := false
			isUnique := false
			isNotNull := !f.IsOptional
			var defaultVal string
			isAutoIncrement := false

			for _, attr := range f.Attributes {
				switch attr.Name {
				case "id":
					isPrimary = true
				case "unique":
					isUnique = true
				case "default":
					if len(attr.Args) > 0 {
						if attr.Args[0] == "autoincrement()" && f.Type == "Int" {
							isAutoIncrement = true
						} else {
							defaultVal = parseDefaultValue(attr.Args[0], f.Type)
						}
					}
				}
			}

			var col string
			if isPrimary && isAutoIncrement && len(compositePK) == 0 {
				col = f.ColumnName + " SERIAL PRIMARY KEY"
			} else {
				col = f.ColumnName + " " + goTypeToSQLType(f.Type, isAutoIncrement, f.Attributes)
				if defaultVal != "" {
					col += " DEFAULT " + defaultVal
				}
				if isNotNull {
					col += " NOT NULL"
				}
			}

			if isPrimary && !isAutoIncrement {
				pkCols = append(pkCols, f.ColumnName)
			}
			if isUnique {
				idxName := "idx_uniq_" + m.TableName + "_" + f.ColumnName
				uniqueIndexes = append(
					uniqueIndexes,
					"CREATE UNIQUE INDEX "+idxName+" ON "+m.TableName+"("+f.ColumnName+");",
				)
			}
			cols = append(cols, col)
		}

		// Generate foreign keys for relation fields
		for _, f := range m.Fields {
			for _, attr := range f.Attributes {
				if attr.Name == "relation" {
					// Debug: Print relation field processing
					fmt.Printf("Processing relation field: %s.%s (type: %s)\n", m.Name, f.Name, f.Type)
					// Find the foreign key field referenced by this relation
					referencedTable := strings.ToLower(f.Type)
					if !strings.HasSuffix(referencedTable, "s") {
						referencedTable += "s"
					}

					// Extract referenced column and foreign key field from relation args
					referencedColumn := "id" // default
					onDelete := ""
					var foreignKeyField *Field

					fmt.Printf("  Total relation args: %d\n", len(attr.Args))
					for i, relationArg := range attr.Args {
						relationArg = strings.TrimSpace(relationArg)
						fmt.Printf("  Processing relation arg[%d]: '%s'\n", i, relationArg)
						if strings.HasPrefix(relationArg, "fields:") {
							// Extract field name from fields: [fieldName]
							start := strings.Index(relationArg, "[")
							end := strings.Index(relationArg, "]")
							if start != -1 && end != -1 {
								fieldName := strings.TrimSpace(relationArg[start+1 : end])
								fmt.Printf("    Looking for field: %s\n", fieldName)
								for _, field := range m.Fields {
									fmt.Printf("      Available field: %s\n", field.Name)
									if field.Name == fieldName {
										foreignKeyField = field
										fmt.Printf("      Found FK field: %s\n", fieldName)
										break
									}
								}
							}
						} else if strings.HasPrefix(relationArg, "references:") {
							// Extract field name from references: [fieldName]
							start := strings.Index(relationArg, "[")
							end := strings.Index(relationArg, "]")
							if start != -1 && end != -1 {
								referencedColumn = strings.TrimSpace(relationArg[start+1 : end])
								fmt.Printf("    Referenced column: %s\n", referencedColumn)
							}
						} else if strings.HasPrefix(relationArg, "onDelete:") {
							parts := strings.Split(relationArg, ":")
							if len(parts) > 1 {
								onDelete = strings.TrimSpace(parts[1])
								fmt.Printf("    OnDelete: %s\n", onDelete)
							}
						}
					}

					if foreignKeyField != nil {
						fkName := "fk_" + m.TableName + "_" + foreignKeyField.ColumnName
						fkStmt := "CONSTRAINT " + fkName + " FOREIGN KEY (" + foreignKeyField.ColumnName + ") REFERENCES " + referencedTable + "(" + referencedColumn + ")"
						if onDelete != "" {
							fkStmt += " ON DELETE " + strings.ToUpper(onDelete)
						}
						foreignKeys = append(foreignKeys, fkStmt)
					}
					break
				}
			}
		}
		// Table-level unique/index
		for _, attr := range m.Attributes {
			switch attr.Name {
			case "unique":
				if len(attr.Args) > 0 {
					idxCols := parseIndexFields(attr.Args, m.Fields)
					idxName := "idx_uniq_" + m.TableName + "_" + strings.Join(idxCols, "_")
					uniqueIndexes = append(
						uniqueIndexes,
						"CREATE UNIQUE INDEX "+idxName+" ON "+m.TableName+"("+strings.Join(idxCols, ", ")+");",
					)
				}
			case "index":
				if len(attr.Args) > 0 {
					idxCols := parseIndexFields(attr.Args, m.Fields)
					idxName := "idx_" + m.TableName + "_" + strings.Join(idxCols, "_")
					indexes = append(
						indexes,
						"CREATE INDEX "+idxName+" ON "+m.TableName+"("+strings.Join(idxCols, ", ")+");",
					)
				}
			}
		}

		// Handle composite primary key or regular primary key
		if len(compositePK) > 0 {
			// Map field names to column names for composite PK
			compositePKCols := []string{}
			for _, fieldName := range compositePK {
				fieldName = strings.Trim(fieldName, "[] \"'")
				for _, f := range m.Fields {
					if f.Name == fieldName {
						compositePKCols = append(compositePKCols, f.ColumnName)
						break
					}
				}
			}
			if len(compositePKCols) > 0 {
				cols = append(cols, "PRIMARY KEY ("+strings.Join(compositePKCols, ", ")+")")
			}
		} else if len(pkCols) > 0 {
			cols = append(cols, "PRIMARY KEY ("+strings.Join(pkCols, ", ")+")")
		}

		// Foreign key constraints
		for _, fk := range foreignKeys {
			cols = append(cols, fk)
		}

		createTable := "CREATE TABLE " + m.TableName + " (\n  " + strings.Join(cols, ",\n  ") + "\n);"
		stmts = append(stmts, wrapGooseStatement(createTable))
		for _, idx := range uniqueIndexes {
			stmts = append(stmts, wrapGooseStatement(idx))
		}
		for _, idx := range indexes {
			stmts = append(stmts, wrapGooseStatement(idx))
		}
	}
	for _, m := range diff.ModelsRemoved {
		warning := fmt.Sprintf("IRREVERSIBLE: Dropping table %s - all data will be lost!", m.TableName)
		stmts = append(stmts, wrapGooseStatementWithWarning("DROP TABLE IF EXISTS "+m.TableName+";", warning))
	}
	return strings.Join(stmts, "\n\n")
}

func wrapGooseStatement(sql string) string {
	return "-- +goose StatementBegin\n" + sql + "\n-- +goose StatementEnd"
}

func wrapGooseStatementWithWarning(sql, warning string) string {
	return "-- +goose StatementBegin\n-- WARNING: " + warning + "\n" + sql + "\n-- +goose StatementEnd"
}

func GenerateDownMigrationSQL(diff *SchemaDiff) string {
	var stmts []string
	// For models added, we need to drop them in down migration
	for _, m := range diff.ModelsAdded {
		stmts = append(stmts, wrapGooseStatement("DROP TABLE IF EXISTS "+m.TableName+";"))
	}

	// For enums added, we need to drop them in down migration
	for _, e := range diff.EnumsAdded {
		stmts = append(stmts, wrapGooseStatement("DROP TYPE IF EXISTS "+e.Name+";"))
	}

	// For fields added, we need to drop them in down migration
	for _, fieldChange := range diff.FieldsAdded {
		stmt := generateDropColumnSQL(fieldChange)
		if stmt != "" {
			stmts = append(stmts, wrapGooseStatement(stmt))
		}
	}

	// For fields removed, we need to add them back in down migration
	for _, fieldChange := range diff.FieldsRemoved {
		stmt := generateAddColumnSQL(fieldChange)
		if stmt != "" {
			stmts = append(stmts, wrapGooseStatement(stmt))
		}
	}

	// For fields modified, we need to revert the changes in down migration
	for _, fieldChange := range diff.FieldsModified {
		stmt := generateReverseModifyColumnSQL(fieldChange)
		if stmt != "" {
			stmts = append(stmts, wrapGooseStatement(stmt))
		}
	}

	// For enums removed, we need to recreate them in down migration
	for _, e := range diff.EnumsRemoved {
		enumStmt := generateEnumSQL(e)
		stmts = append(stmts, wrapGooseStatement(enumStmt))
	}

	// For models removed, we need to recreate them in down migration
	for _, m := range diff.ModelsRemoved {
		cols := []string{}
		pkCols := []string{}
		indexes := []string{}
		uniqueIndexes := []string{}
		for _, f := range m.Fields {
			isPrimary := false
			isUnique := false
			isNotNull := !f.IsOptional
			var defaultVal string
			isAutoIncrement := false

			for _, attr := range f.Attributes {
				switch attr.Name {
				case "id":
					isPrimary = true
				case "unique":
					isUnique = true
				case "default":
					if len(attr.Args) > 0 {
						if attr.Args[0] == "autoincrement()" && f.Type == "Int" {
							isAutoIncrement = true
						} else {
							defaultVal = parseDefaultValue(attr.Args[0], f.Type)
						}
					}
				}
			}

			var col string
			if isPrimary && isAutoIncrement {
				col = f.ColumnName + " SERIAL PRIMARY KEY"
			} else {
				col = f.ColumnName + " " + goTypeToSQLType(f.Type, isAutoIncrement, f.Attributes)
				if defaultVal != "" {
					col += " DEFAULT " + defaultVal
				}
				if isNotNull {
					col += " NOT NULL"
				}
			}

			if isPrimary && !isAutoIncrement {
				pkCols = append(pkCols, f.ColumnName)
			}
			if isUnique {
				idxName := "idx_uniq_" + m.TableName + "_" + f.ColumnName
				uniqueIndexes = append(
					uniqueIndexes,
					"CREATE UNIQUE INDEX "+idxName+" ON "+m.TableName+"("+f.ColumnName+");",
				)
			}
			cols = append(cols, col)
		}
		// Table-level unique/index
		for _, attr := range m.Attributes {
			switch attr.Name {
			case "unique":
				if len(attr.Args) > 0 {
					idxCols := parseIndexFields(attr.Args, m.Fields)
					idxName := "idx_uniq_" + m.TableName + "_" + strings.Join(idxCols, "_")
					uniqueIndexes = append(
						uniqueIndexes,
						"CREATE UNIQUE INDEX "+idxName+" ON "+m.TableName+"("+strings.Join(idxCols, ", ")+");",
					)
				}
			case "index":
				if len(attr.Args) > 0 {
					idxCols := parseIndexFields(attr.Args, m.Fields)
					idxName := "idx_" + m.TableName + "_" + strings.Join(idxCols, "_")
					indexes = append(
						indexes,
						"CREATE INDEX "+idxName+" ON "+m.TableName+"("+strings.Join(idxCols, ", ")+");",
					)
				}
			}
		}
		// PRIMARY KEY
		if len(pkCols) > 0 {
			cols = append(cols, "PRIMARY KEY ("+strings.Join(pkCols, ", ")+")")
		}
		createTable := "CREATE TABLE " + m.TableName + " (\n  " + strings.Join(cols, ",\n  ") + "\n);"
		stmts = append(stmts, wrapGooseStatement(createTable))
		for _, idx := range uniqueIndexes {
			stmts = append(stmts, wrapGooseStatement(idx))
		}
		for _, idx := range indexes {
			stmts = append(stmts, wrapGooseStatement(idx))
		}
	}
	return strings.Join(stmts, "\n\n")
}

func goTypeToSQLType(t string, isAutoIncrement bool, attributes []*FieldAttribute) string {
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

	switch t {
	case "Int":
		if isAutoIncrement {
			return "SERIAL"
		}
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
		// Check if it's a custom enum type
		return t // Will be handled as enum type
	}
}

func generateEnumSQL(e *Enum) string {
	values := make([]string, len(e.Values))
	for i, v := range e.Values {
		values[i] = "'" + v + "'"
	}
	return "CREATE TYPE " + e.Name + " AS ENUM (" + strings.Join(values, ", ") + ");"
}

func isRelationField(field *Field) bool {
	for _, attr := range field.Attributes {
		if attr.Name == "relation" {
			return true
		}
	}
	// Also check if it's an array type or custom type (not basic types)
	if field.IsArray {
		return true
	}
	// Check if it's a custom model type
	if field.Type != "Int" && field.Type != "String" && field.Type != "DateTime" && field.Type != "Boolean" &&
		field.Type != "Float" {
		// Could be a custom model or enum - check if it has relation attributes
		return len(field.Attributes) == 0 // Relations usually have no attributes or only @relation
	}
	return false
}

func getRelationInfo(field *Field) (string, string, string) {
	// Returns: referencedTable, referencedColumn, onDelete
	var referencedTable, referencedColumn, onDelete string
	for _, attr := range field.Attributes {
		if attr.Name == "relation" {
			if len(attr.Args) >= 2 {
				// @relation(fields: [organizationId], references: [id])
				for _, arg := range attr.Args {
					arg = strings.TrimSpace(arg)
					if strings.HasPrefix(arg, "fields:") {
						// Skip - this is the local field
					} else if strings.HasPrefix(arg, "references:") {
						// Extract referenced column
						refPart := strings.TrimPrefix(arg, "references:")
						refPart = strings.Trim(refPart, " []")
						referencedColumn = refPart
					} else if strings.HasPrefix(arg, "onDelete:") {
						onDelete = strings.TrimPrefix(arg, "onDelete:")
						onDelete = strings.TrimSpace(onDelete)
					}
				}
			}
		}
	}

	// Extract referenced table from field type
	fieldType := field.Type
	if fieldType != "Int" && fieldType != "String" {
		referencedTable = strings.ToLower(fieldType) + "s" // Simple pluralization
	}

	if referencedColumn == "" {
		referencedColumn = "id" // Default reference column
	}

	return referencedTable, referencedColumn, onDelete
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
		if v == "autoincrement()" {
			return "" // This should be handled by SERIAL, so we return empty for default
		}
		return v
	}
}

func generateAddColumnSQL(fieldChange *FieldChange) string {
	f := fieldChange.Field

	// Skip relation fields that don't have actual columns (array types and fields with @relation)
	if f.IsArray {
		return ""
	}
	hasRelationAttr := false
	for _, attr := range f.Attributes {
		if attr.Name == "relation" {
			hasRelationAttr = true
			break
		}
	}
	if hasRelationAttr {
		return ""
	}

	isPrimary := false
	isUnique := false
	isNotNull := !f.IsOptional
	var defaultVal string
	isAutoIncrement := false

	for _, attr := range f.Attributes {
		switch attr.Name {
		case "id":
			isPrimary = true
		case "unique":
			isUnique = true
		case "default":
			if len(attr.Args) > 0 {
				if attr.Args[0] == "autoincrement()" && f.Type == "Int" {
					isAutoIncrement = true
				} else {
					defaultVal = parseDefaultValue(attr.Args[0], f.Type)
				}
			}
		}
	}

	var col string
	if isPrimary && isAutoIncrement {
		col = f.ColumnName + " SERIAL PRIMARY KEY"
	} else {
		col = f.ColumnName + " " + goTypeToSQLType(f.Type, isAutoIncrement, f.Attributes)
		if defaultVal != "" {
			col += " DEFAULT " + defaultVal
		}
		if isNotNull {
			col += " NOT NULL"
		}
	}

	stmt := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s;", fieldChange.ModelName, col)

	// Handle unique constraint separately
	if isUnique {
		idxName := "idx_uniq_" + fieldChange.ModelName + "_" + f.ColumnName
		stmt += fmt.Sprintf("\nCREATE UNIQUE INDEX %s ON %s(%s);", idxName, fieldChange.ModelName, f.ColumnName)
	}

	return stmt
}

func generateDropColumnSQL(fieldChange *FieldChange) string {
	f := fieldChange.Field

	// Skip relation fields that don't have actual columns
	if f.IsArray {
		return ""
	}
	hasRelationAttr := false
	for _, attr := range f.Attributes {
		if attr.Name == "relation" {
			hasRelationAttr = true
			break
		}
	}
	if hasRelationAttr {
		return ""
	}

	return fmt.Sprintf("ALTER TABLE %s DROP COLUMN IF EXISTS %s;", fieldChange.ModelName, f.ColumnName)
}

func parseIndexFields(args []string, fields []*Field) []string {
	var cols []string
	for _, a := range args {
		s := strings.Trim(a, "[] \"'")
		if s == "" {
			continue
		}
		for _, f := range fields {
			if f.Name == s {
				cols = append(cols, f.ColumnName)
			}
		}
	}
	return cols
}

func generateModifyColumnSQLWithWarning(fieldChange *FieldChange) (string, string) {
	currentField := fieldChange.CurrentField
	targetField := fieldChange.Field

	// Skip relation fields that don't have actual columns
	if targetField.IsArray {
		return "", ""
	}
	hasRelationAttr := false
	for _, attr := range targetField.Attributes {
		if attr.Name == "relation" {
			hasRelationAttr = true
			break
		}
	}
	if hasRelationAttr {
		return "", ""
	}

	// Now we can compare current vs target to determine exactly what changed
	var stmts []string
	var warnings []string

	// Check if type changed
	currentNormalizedType := NormalizeTypeForComparison(currentField.Type, currentField.Attributes)
	targetNormalizedType := NormalizeTypeForComparison(targetField.Type, targetField.Attributes)

	if currentNormalizedType != targetNormalizedType {
		// Type change - need casting
		newSQLType := goTypeToSQLType(targetField.Type, false, targetField.Attributes)
		castResult := CanCastType(currentNormalizedType, targetNormalizedType)

		if castResult.CanCast {
			if castResult.CastExpression != "" {
				// Use explicit casting
				stmt := fmt.Sprintf(
					"ALTER TABLE %s ALTER COLUMN %s TYPE %s USING %s%s;",
					fieldChange.ModelName,
					targetField.ColumnName,
					newSQLType,
					targetField.ColumnName,
					castResult.CastExpression,
				)
				stmts = append(stmts, stmt)
			} else {
				// Simple type change
				stmt := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s;",
					fieldChange.ModelName, targetField.ColumnName, newSQLType)
				stmts = append(stmts, stmt)
			}

			// Collect warnings for risky conversions
			if castResult.IsRisky || castResult.WarningMessage != "" {
				warning := fmt.Sprintf(
					"RISKY CONVERSION: %s.%s from %s to %s - %s. This cannot be safely rolled back!",
					fieldChange.ModelName,
					targetField.ColumnName,
					currentNormalizedType,
					targetNormalizedType,
					castResult.WarningMessage,
				)
				warnings = append(warnings, warning)
				LogTypeCastWarning(fieldChange.ModelName, targetField.ColumnName, castResult)
			}
		} else {
			// Cannot cast automatically
			log.Printf("ERROR: Cannot automatically convert column %s.%s - %s",
				fieldChange.ModelName, targetField.ColumnName, castResult.WarningMessage)
			stmts = append(stmts, fmt.Sprintf("-- ERROR: %s\n-- Manual migration required for %s.%s",
				castResult.WarningMessage, fieldChange.ModelName, targetField.ColumnName))
			warning := fmt.Sprintf("MANUAL INTERVENTION REQUIRED: %s", castResult.WarningMessage)
			warnings = append(warnings, warning)
		}
	}

	// Check if nullability changed
	if currentField.IsOptional != targetField.IsOptional {
		if targetField.IsOptional {
			// Make column nullable
			nullStmt := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP NOT NULL;",
				fieldChange.ModelName, targetField.ColumnName)
			stmts = append(stmts, nullStmt)
		} else {
			// Make column not nullable - this is risky
			nullStmt := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL;",
				fieldChange.ModelName, targetField.ColumnName)
			stmts = append(stmts, nullStmt)
			warning := fmt.Sprintf("RISKY: Making %s.%s NOT NULL - will fail if NULL values exist. Cannot be safely rolled back if data is modified!",
				fieldChange.ModelName, targetField.ColumnName)
			warnings = append(warnings, warning)
		}
	}

	if len(stmts) == 0 {
		// No actual changes detected
		return fmt.Sprintf("-- No changes detected for %s.%s", fieldChange.ModelName, targetField.ColumnName), ""
	}

	var combinedWarning string
	if len(warnings) > 0 {
		combinedWarning = strings.Join(warnings, " | ")
	}

	return strings.Join(stmts, "\n"), combinedWarning
}

func generateModifyColumnSQL(fieldChange *FieldChange) string {
	sql, _ := generateModifyColumnSQLWithWarning(fieldChange)
	return sql
}

func generateReverseModifyColumnSQL(fieldChange *FieldChange) string {
	currentField := fieldChange.CurrentField // What it was before
	targetField := fieldChange.Field         // What it became

	// Skip relation fields
	if targetField.IsArray {
		return ""
	}
	hasRelationAttr := false
	for _, attr := range targetField.Attributes {
		if attr.Name == "relation" {
			hasRelationAttr = true
			break
		}
	}
	if hasRelationAttr {
		return ""
	}

	var stmts []string

	// Reverse type changes
	currentNormalizedType := NormalizeTypeForComparison(currentField.Type, currentField.Attributes)
	targetNormalizedType := NormalizeTypeForComparison(targetField.Type, targetField.Attributes)

	if currentNormalizedType != targetNormalizedType {
		// Need to reverse the type change: target -> current
		originalSQLType := goTypeToSQLType(currentField.Type, false, currentField.Attributes)
		castResult := CanCastType(targetNormalizedType, currentNormalizedType)

		if castResult.CanCast && !castResult.IsRisky {
			// Safe to reverse
			if castResult.CastExpression != "" {
				stmt := fmt.Sprintf(
					"ALTER TABLE %s ALTER COLUMN %s TYPE %s USING %s%s;",
					fieldChange.ModelName,
					targetField.ColumnName,
					originalSQLType,
					targetField.ColumnName,
					castResult.CastExpression,
				)
				stmts = append(stmts, stmt)
			} else {
				stmt := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s;",
					fieldChange.ModelName, targetField.ColumnName, originalSQLType)
				stmts = append(stmts, stmt)
			}
		} else if castResult.CanCast && castResult.IsRisky {
			// Risky reversal - warn but allow
			stmt := fmt.Sprintf("-- WARNING: Risky type reversal from %s to %s\n-- %s\nALTER TABLE %s ALTER COLUMN %s TYPE %s USING %s%s;",
				targetNormalizedType, currentNormalizedType, castResult.WarningMessage,
				fieldChange.ModelName, targetField.ColumnName, originalSQLType, targetField.ColumnName, castResult.CastExpression)
			stmts = append(stmts, stmt)
		} else {
			// Cannot reverse automatically
			stmt := fmt.Sprintf("-- ERROR: Cannot automatically reverse type change for %s.%s\n-- From %s back to %s: %s\n-- Manual intervention required",
				fieldChange.ModelName, targetField.ColumnName, targetNormalizedType, currentNormalizedType, castResult.WarningMessage)
			stmts = append(stmts, stmt)
		}
	}

	// Reverse nullability changes
	if currentField.IsOptional != targetField.IsOptional {
		if currentField.IsOptional {
			// Original was nullable, target became not null -> reverse to nullable
			nullStmt := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP NOT NULL;",
				fieldChange.ModelName, targetField.ColumnName)
			stmts = append(stmts, nullStmt)
		} else {
			// Original was not null, target became nullable -> reverse to not null
			// This is potentially dangerous if NULL values were inserted
			nullStmt := fmt.Sprintf("-- WARNING: Setting NOT NULL may fail if NULL values exist\nALTER TABLE %s ALTER COLUMN %s SET NOT NULL;",
				fieldChange.ModelName, targetField.ColumnName)
			stmts = append(stmts, nullStmt)
		}
	}

	if len(stmts) == 0 {
		return fmt.Sprintf("-- No reverse changes needed for %s.%s", fieldChange.ModelName, targetField.ColumnName)
	}

	return strings.Join(stmts, "\n")
}
