package schema

type SchemaDiff struct {
	ModelsAdded   []*Model
	ModelsRemoved []*Model
	EnumsAdded    []*Enum
	EnumsRemoved  []*Enum
	// ... các thay đổi khác (sau này)
}

func DiffSchemas(current, target *Schema) *SchemaDiff {
	// Models diff - use TableName for comparison since that's what matters for SQL
	modelsAdded := []*Model{}
	modelsRemoved := []*Model{}
	currentModelMap := map[string]*Model{}
	targetModelMap := map[string]*Model{}
	for _, m := range current.Models {
		currentModelMap[m.TableName] = m
	}
	for _, m := range target.Models {
		targetModelMap[m.TableName] = m
	}
	for tableName, tModel := range targetModelMap {
		if _, ok := currentModelMap[tableName]; !ok {
			modelsAdded = append(modelsAdded, tModel)
		}
	}
	for tableName, cModel := range currentModelMap {
		if _, ok := targetModelMap[tableName]; !ok {
			modelsRemoved = append(modelsRemoved, cModel)
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
	}
}
