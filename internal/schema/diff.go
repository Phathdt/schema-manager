package schema

type FieldChange struct {
	ModelName string
	Field     *Field
	Type      string // "added", "removed", "modified"
}

type SchemaDiff struct {
	ModelsAdded   []*Model
	ModelsRemoved []*Model
	EnumsAdded    []*Enum
	EnumsRemoved  []*Enum
	FieldsAdded   []*FieldChange
	FieldsRemoved []*FieldChange
	// ... các thay đổi khác (sau này)
}

func DiffSchemas(current, target *Schema) *SchemaDiff {
	// Models diff - use TableName for comparison since that's what matters for SQL
	modelsAdded := []*Model{}
	modelsRemoved := []*Model{}
	fieldsAdded := []*FieldChange{}
	fieldsRemoved := []*FieldChange{}

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
		ModelsAdded:   modelsAdded,
		ModelsRemoved: modelsRemoved,
		EnumsAdded:    enumsAdded,
		EnumsRemoved:  enumsRemoved,
		FieldsAdded:   fieldsAdded,
		FieldsRemoved: fieldsRemoved,
	}
}
