package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/phathdt/schema-manager/internal/schema"
	"github.com/urfave/cli/v2"
)

func GenerateCommand() *cli.Command {
	return &cli.Command{
		Name:  "generate",
		Usage: "Generate migration from Prisma schema changes",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "name", Usage: "Migration name", Required: true},
		},
		Action: func(c *cli.Context) error {
			ctx := context.Background()
			prismaSource := &schema.PrismaFileSource{Path: "schema.prisma"}
			migrationsSource := &schema.MigrationsFolderSource{Dir: "migrations"}
			targetSchema, err := prismaSource.LoadSchema(ctx)
			if err != nil {
				return cli.Exit("Failed to parse schema.prisma: "+err.Error(), 1)
			}
			entries, err := os.ReadDir("migrations")
			if err != nil || len(entries) == 0 {
				// Initial migration
				diff := &schema.SchemaDiff{}
				for _, m := range targetSchema.Models {
					diff.ModelsAdded = append(diff.ModelsAdded, m)
				}
				for _, e := range targetSchema.Enums {
					diff.EnumsAdded = append(diff.EnumsAdded, e)
				}
				up := schema.GenerateMigrationSQL(diff)
				down := schema.GenerateDownMigrationSQL(diff)
				ts := time.Now().Format("20060102150405")
				name := c.String("name")
				os.MkdirAll("migrations", 0755)
				filename := "migrations/" + ts + "_" + name + ".sql"
				f, err := os.Create(filename)
				if err != nil {
					return cli.Exit("Failed to create migration file: "+err.Error(), 1)
				}
				defer f.Close()
				f.WriteString("-- +goose Up\n" + up + "\n\n-- +goose Down\n" + down)
				fmt.Println("Created migration:", filename)
				return nil
			}
			currentSchema, err := migrationsSource.LoadSchema(ctx)
			if err != nil {
				return cli.Exit("Failed to parse current schema from migrations: "+err.Error(), 1)
			}

			// Debug: Print current schema
			fmt.Printf("Current schema has %d models, %d enums\n", len(currentSchema.Models), len(currentSchema.Enums))
			for _, m := range currentSchema.Models {
				fmt.Printf("  - Model: %s (table: %s)\n", m.Name, m.TableName)
			}
			for _, e := range currentSchema.Enums {
				fmt.Printf("  - Enum: %s\n", e.Name)
			}

			fmt.Printf("Target schema has %d models, %d enums\n", len(targetSchema.Models), len(targetSchema.Enums))
			for _, m := range targetSchema.Models {
				fmt.Printf("  - Model: %s (table: %s)\n", m.Name, m.TableName)
			}
			for _, e := range targetSchema.Enums {
				fmt.Printf("  - Enum: %s\n", e.Name)
			}

			diff := schema.DiffSchemas(currentSchema, targetSchema)
			fmt.Printf("Diff: %d models added, %d models removed, %d enums added, %d enums removed, %d fields added, %d fields removed, %d fields modified\n",
				len(diff.ModelsAdded), len(diff.ModelsRemoved), len(diff.EnumsAdded), len(diff.EnumsRemoved), len(diff.FieldsAdded), len(diff.FieldsRemoved), len(diff.FieldsModified))

			if diff == nil || (len(diff.ModelsAdded) == 0 && len(diff.EnumsAdded) == 0 && len(diff.FieldsAdded) == 0 && len(diff.FieldsRemoved) == 0 && len(diff.FieldsModified) == 0) {
				fmt.Println("No changes detected.")
				return nil
			}

			// Check for risky operations before generating
			risks := analyzeRiskyOperations(diff)
			if len(risks) > 0 {
				fmt.Println("\n⚠️  WARNING: The following operations cannot be automatically rolled back:")
				for _, risk := range risks {
					fmt.Printf("  • %s\n", risk)
				}
				fmt.Print("\nDo you want to continue? This will generate the migration with warnings. (y/N): ")

				reader := bufio.NewReader(os.Stdin)
				response, err := reader.ReadString('\n')
				if err != nil {
					return cli.Exit("Failed to read user input: "+err.Error(), 1)
				}

				response = strings.ToLower(strings.TrimSpace(response))
				if response != "y" && response != "yes" {
					fmt.Println("Migration generation cancelled.")
					return nil
				}

				fmt.Println("Proceeding with risky migration...")
			}
			up := schema.GenerateMigrationSQL(diff)
			down := schema.GenerateDownMigrationSQL(diff)
			ts := time.Now().Format("20060102150405")
			name := c.String("name")
			filename := "migrations/" + ts + "_" + name + ".sql"
			f, err := os.Create(filename)
			if err != nil {
				return cli.Exit("Failed to create migration file: "+err.Error(), 1)
			}
			defer f.Close()
			f.WriteString("-- +goose Up\n" + up + "\n\n-- +goose Down\n" + down)
			fmt.Println("Created migration:", filename)
			return nil
		},
	}
}

// analyzeRiskyOperations checks for operations that cannot be safely rolled back
func analyzeRiskyOperations(diff *schema.SchemaDiff) []string {
	var risks []string

	// Check field modifications for risky type changes
	for _, fieldChange := range diff.FieldsModified {
		currentField := fieldChange.CurrentField
		targetField := fieldChange.Field

		currentNormalizedType := schema.NormalizeTypeForComparison(currentField.Type, currentField.Attributes)
		targetNormalizedType := schema.NormalizeTypeForComparison(targetField.Type, targetField.Attributes)

		if currentNormalizedType != targetNormalizedType {
			// Check forward conversion (UP migration)
			forwardCastResult := schema.CanCastType(currentNormalizedType, targetNormalizedType)
			// Check reverse conversion (DOWN migration rollback)
			reverseCastResult := schema.CanCastType(targetNormalizedType, currentNormalizedType)

			if forwardCastResult.IsRisky {
				risk := fmt.Sprintf("Field %s.%s: %s → %s (%s)",
					fieldChange.ModelName, targetField.ColumnName,
					currentNormalizedType, targetNormalizedType, forwardCastResult.WarningMessage)
				risks = append(risks, risk)
			} else if !forwardCastResult.CanCast {
				risk := fmt.Sprintf("Field %s.%s: %s → %s (Cannot be automatically cast - manual intervention required)",
					fieldChange.ModelName, targetField.ColumnName,
					currentNormalizedType, targetNormalizedType)
				risks = append(risks, risk)
			}

			// Also check if the rollback would be risky
			if reverseCastResult.IsRisky {
				risk := fmt.Sprintf("Field %s.%s: %s → %s (ROLLBACK RISK: %s)",
					fieldChange.ModelName, targetField.ColumnName,
					currentNormalizedType, targetNormalizedType, reverseCastResult.WarningMessage)
				risks = append(risks, risk)
			} else if !reverseCastResult.CanCast {
				risk := fmt.Sprintf("Field %s.%s: %s → %s (ROLLBACK IMPOSSIBLE: Cannot reverse this conversion)",
					fieldChange.ModelName, targetField.ColumnName,
					currentNormalizedType, targetNormalizedType)
				risks = append(risks, risk)
			}
		}

		// Check for nullability changes that could be problematic
		if !currentField.IsOptional && targetField.IsOptional {
			// Making a field nullable is generally safe
		} else if currentField.IsOptional && !targetField.IsOptional {
			// Making a field NOT NULL is risky if there are existing NULL values
			risk := fmt.Sprintf("Field %s.%s: Making nullable field NOT NULL (may fail if NULL values exist)",
				fieldChange.ModelName, targetField.ColumnName)
			risks = append(risks, risk)
		}
	}

	// Check for model/table drops - these can't be easily rolled back with data
	for _, model := range diff.ModelsRemoved {
		risk := fmt.Sprintf("Table %s: Being dropped (all data will be lost)", model.TableName)
		risks = append(risks, risk)
	}

	// Check for field removals - data will be lost
	for _, fieldChange := range diff.FieldsRemoved {
		risk := fmt.Sprintf("Field %s.%s: Being removed (column data will be lost)",
			fieldChange.ModelName, fieldChange.Field.ColumnName)
		risks = append(risks, risk)
	}

	// Check for enum removals
	for _, enum := range diff.EnumsRemoved {
		risk := fmt.Sprintf("Enum %s: Being dropped (may affect dependent fields)", enum.Name)
		risks = append(risks, risk)
	}

	return risks
}
