package schema

import (
	"strings"
)

type FieldChange struct {
	ModelName    string
	Field        *Field // Target field
	CurrentField *Field // Current field (for modifications)
	Type         string // "added", "removed", "modified"
}

type SchemaDiff struct {
	ModelsAdded    []*Model
	ModelsRemoved  []*Model
	EnumsAdded     []*Enum
	EnumsRemoved   []*Enum
	FieldsAdded    []*FieldChange
	FieldsRemoved  []*FieldChange
	FieldsModified []*FieldChange
}

func DiffSchemas(current, target *Schema) *SchemaDiff {
	// Models diff - use TableName for comparison since that's what matters for SQL
	modelsAdded := []*Model{}
	modelsRemoved := []*Model{}
	fieldsAdded := []*FieldChange{}
	fieldsRemoved := []*FieldChange{}
	fieldsModified := []*FieldChange{}

	currentModelMap := map[string]*Model{}
	targetModelMap := map[string]*Model{}
	for _, m := range current.Models {
		currentModelMap[m.TableName] = m
	}
	for _, m := range target.Models {
		targetModelMap[m.TableName] = m
	}

	// Check for models added
	for tableName, tModel := range targetModelMap {
		if _, ok := currentModelMap[tableName]; !ok {
			modelsAdded = append(modelsAdded, tModel)
		}
	}

	// Check for models removed
	for tableName, cModel := range currentModelMap {
		if _, ok := targetModelMap[tableName]; !ok {
			modelsRemoved = append(modelsRemoved, cModel)
		}
	}

	// Check for field changes within existing models
	for tableName, tModel := range targetModelMap {
		if cModel, ok := currentModelMap[tableName]; ok {
			// Model exists in both, check for field changes

			currentFieldMap := map[string]*Field{}
			targetFieldMap := map[string]*Field{}

			for _, f := range cModel.Fields {
				currentFieldMap[f.ColumnName] = f
			}
			for _, f := range tModel.Fields {
				targetFieldMap[f.ColumnName] = f
			}

			// Check for fields added
			for columnName, tField := range targetFieldMap {
				if _, ok := currentFieldMap[columnName]; !ok {
					fieldsAdded = append(fieldsAdded, &FieldChange{
						ModelName: tModel.TableName,
						Field:     tField,
						Type:      "added",
					})
				}
			}

			// Check for fields removed
			for columnName, cField := range currentFieldMap {
				if _, ok := targetFieldMap[columnName]; !ok {
					fieldsRemoved = append(fieldsRemoved, &FieldChange{
						ModelName: cModel.TableName,
						Field:     cField,
						Type:      "removed",
					})
				}
			}

			// Check for fields modified
			for columnName, tField := range targetFieldMap {
				if cField, ok := currentFieldMap[columnName]; ok {
					// Field exists in both, check if it's been modified

					if !fieldsEqual(cField, tField) {
						fieldsModified = append(fieldsModified, &FieldChange{
							ModelName:    tModel.TableName,
							Field:        tField,
							CurrentField: cField,
							Type:         "modified",
						})
					}
				}
			}
		}
	}

	// Enums diff
	enumsAdded := []*Enum{}
	enumsRemoved := []*Enum{}
	currentEnumMap := map[string]*Enum{}
	targetEnumMap := map[string]*Enum{}
	for _, e := range current.Enums {
		currentEnumMap[e.Name] = e
	}
	for _, e := range target.Enums {
		targetEnumMap[e.Name] = e
	}
	for name, tEnum := range targetEnumMap {
		if _, ok := currentEnumMap[name]; !ok {
			enumsAdded = append(enumsAdded, tEnum)
		}
	}
	for name, cEnum := range currentEnumMap {
		if _, ok := targetEnumMap[name]; !ok {
			enumsRemoved = append(enumsRemoved, cEnum)
		}
	}

	return &SchemaDiff{
		ModelsAdded:    modelsAdded,
		ModelsRemoved:  modelsRemoved,
		EnumsAdded:     enumsAdded,
		EnumsRemoved:   enumsRemoved,
		FieldsAdded:    fieldsAdded,
		FieldsRemoved:  fieldsRemoved,
		FieldsModified: fieldsModified,
	}
}

// fieldsEqual compares two fields to see if they are equivalent
func fieldsEqual(current, target *Field) bool {
	// Both schemas now use consistent internal representation from SQL parsing
	// Compare the SQL types directly - this handles DECIMAL precision/scale automatically
	currentSQL := GetSQLTypeForField(current)
	targetSQL := GetSQLTypeForField(target)

	if currentSQL != targetSQL {
		return false
	}

	if current.IsOptional != target.IsOptional {
		return false
	}

	if current.IsArray != target.IsArray {
		return false
	}

	// No need for complex attribute comparison since migration parser produces clean schema
	return true
}

// NormalizeTypeForComparison converts both PostgreSQL and Prisma types to a common format for comparison
func NormalizeTypeForComparison(fieldType string, attributes []*FieldAttribute) string {
	// Handle PostgreSQL types from migrations - convert to Prisma equivalent
	switch fieldType {
	case "TEXT":
		return "String"
	case "INTEGER":
		return "Int"
	case "BIGINT":
		return "BigInt"
	case "SERIAL":
		// SERIAL is PostgreSQL's auto-increment integer - equivalent to Int with @id @default(autoincrement())
		return "Int"
	case "TIMESTAMP":
		return "DateTime"
	case "BOOLEAN":
		return "Boolean"
	case "DOUBLE PRECISION", "FLOAT":
		return "Float"
	case "JSONB", "JSON":
		return "Json"
	case "NUMERIC":
		return "Decimal"
	default:
		// Handle DECIMAL(precision, scale) types
		if strings.HasPrefix(fieldType, "DECIMAL(") {
			return "Decimal"
		}

		// For Prisma types with @db attributes, normalize to the base type
		if fieldType == "Decimal" {
			return "Decimal"
		}

		// For Prisma types, return as-is
		return fieldType
	}
}

// getSQLTypeForField returns the SQL type for a field, considering @db attributes
func GetSQLTypeForField(field *Field) string {
	// Check for @db type attributes first
	for _, attr := range field.Attributes {
		if strings.HasPrefix(attr.Name, "db.") {
			dbType := strings.TrimPrefix(attr.Name, "db.")
			if dbType == "VarChar" && len(attr.Args) > 0 {
				return "VARCHAR(" + attr.Args[0] + ")"
			}
			if dbType == "Text" {
				return "TEXT"
			}
			if dbType == "Decimal" && len(attr.Args) >= 2 {
				return "DECIMAL(" + attr.Args[0] + "," + attr.Args[1] + ")"
			}
		}
	}

	// If field type is already a SQL type (from migrations), normalize and return
	// Handle case-insensitive DECIMAL types from migrations
	upperType := strings.ToUpper(field.Type)
	if strings.HasPrefix(upperType, "DECIMAL(") {
		// Normalize to uppercase for consistency
		return upperType
	}

	// Handle other SQL types from migrations (normalize to uppercase)
	switch strings.ToUpper(field.Type) {
	case "TEXT":
		return "TEXT"
	case "INTEGER":
		return "INTEGER"
	case "BIGINT":
		return "BIGINT"
	case "SERIAL":
		// SERIAL from migrations should be treated as INTEGER for comparison purposes
		// since it's functionally equivalent to Int @default(autoincrement())
		return "INTEGER"
	case "NUMERIC":
		return "NUMERIC"
	case "TIMESTAMP":
		return "TIMESTAMP"
	case "BOOLEAN":
		return "BOOLEAN"
	}

	// Map Prisma types to SQL types
	switch field.Type {
	case "String":
		return "TEXT"
	case "Int":
		// Check if this Int field has autoincrement - if so, it's equivalent to SERIAL
		// For comparison purposes, we normalize both to INTEGER
		return "INTEGER"
	case "BigInt":
		return "BIGINT"
	case "Float":
		return "DOUBLE PRECISION"
	case "Decimal":
		return "NUMERIC"
	case "Boolean":
		return "BOOLEAN"
	case "DateTime":
		return "TIMESTAMP"
	case "Json":
		return "JSONB"
	default:
		return strings.ToUpper(field.Type)
	}
}
