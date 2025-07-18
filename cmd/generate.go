package cmd

import (
	"context"
	"fmt"
	"os"
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
			fmt.Printf("Diff: %d models added, %d models removed, %d enums added, %d enums removed, %d fields added, %d fields removed\n",
				len(diff.ModelsAdded), len(diff.ModelsRemoved), len(diff.EnumsAdded), len(diff.EnumsRemoved), len(diff.FieldsAdded), len(diff.FieldsRemoved))

			if diff == nil || (len(diff.ModelsAdded) == 0 && len(diff.EnumsAdded) == 0 && len(diff.FieldsAdded) == 0 && len(diff.FieldsRemoved) == 0) {
				fmt.Println("No changes detected.")
				return nil
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
