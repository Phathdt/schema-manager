package schema

import (
	"context"
	"fmt"
	"os"
	"strings"
)

func ParsePrismaFileToSchema(ctx context.Context, path string) (*Schema, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content := string(b)
	lines := strings.Split(content, "\n")
	schema := &Schema{}
	var currentModel *Model
	var currentEnum *Enum
	for _, line := range lines {
		l := strings.TrimSpace(line)
		if l == "" || strings.HasPrefix(l, "//") {
			continue
		}
		if strings.HasPrefix(l, "model ") {
			name := strings.Fields(l)[1]
			currentModel = &Model{Name: name, TableName: name}
			schema.Models = append(schema.Models, currentModel)
			continue
		}
		if strings.HasPrefix(l, "enum ") {
			name := strings.Fields(l)[1]
			currentEnum = &Enum{Name: name}
			schema.Enums = append(schema.Enums, currentEnum)
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
	return schema, nil
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
	fmt.Printf("DEBUG: parseField line: '%s'\n", line)
	fmt.Printf("DEBUG: parseField parts: %v\n", parts)
	for _, p := range parts[2:] {
		fmt.Printf("DEBUG: parseField part: '%s'\n", p)
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
	fmt.Printf("DEBUG: parseFieldAttribute token: '%s'\n", token)
	token = strings.TrimPrefix(token, "@")
	name := token
	var args []string
	if i := strings.Index(token, "("); i >= 0 {
		name = token[:i]
		argsStr := strings.TrimSuffix(token[i+1:], ")")
		fmt.Printf("DEBUG: argsStr: '%s'\n", argsStr)
		// Handle complex args like "fields: [organizationId], references: [id]"
		if strings.Contains(argsStr, ":") {
			// Split by commas, but be careful with nested brackets
			parts := splitComplexArgs(argsStr)
			for _, part := range parts {
				args = append(args, strings.TrimSpace(part))
			}
		} else {
			args = strings.Split(argsStr, ",")
			for i := range args {
				args[i] = strings.TrimSpace(args[i])
			}
		}

		// Debug: print parsed args
		fmt.Printf("DEBUG: Parsed @%s args: %v\n", name, args)
	}
	return &FieldAttribute{Name: name, Args: args}
}

func splitComplexArgs(argsStr string) []string {
	var args []string
	var current strings.Builder
	inBrackets := 0

	// Debug: print input
	fmt.Printf("DEBUG: splitComplexArgs input: '%s'\n", argsStr)

	for _, char := range argsStr {
		if char == '[' {
			inBrackets++
		} else if char == ']' {
			inBrackets--
		} else if char == ',' && inBrackets == 0 {
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
			continue
		}
		current.WriteRune(char)
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	fmt.Printf("DEBUG: splitComplexArgs output: %v\n", args)
	return args
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
