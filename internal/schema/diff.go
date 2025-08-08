package schema

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
	// Normalize types for comparison (PostgreSQL types vs Prisma types)
	currentNormalizedType := NormalizeTypeForComparison(current.Type, current.Attributes)
	targetNormalizedType := NormalizeTypeForComparison(target.Type, target.Attributes)

	// Compare normalized types
	if currentNormalizedType != targetNormalizedType {
		return false
	}
	if current.IsOptional != target.IsOptional {
		return false
	}
	if current.IsArray != target.IsArray {
		return false
	}

	// Only compare schema-affecting attributes, not metadata attributes
	// The current schema (from migrations) won't have Prisma-specific attributes like @map, @unique, etc.
	// since those are handled at the SQL level in the migration files.
	// We should only care about attributes that affect the actual database schema structure.

	// For now, skip attribute comparison for fields that already exist in the database
	// since the migration parser doesn't capture Prisma metadata attributes.
	// TODO: In the future, we could compare specific schema-affecting attributes like constraints.

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
	default:
		// For Prisma types, return as-is
		return fieldType
	}
}
