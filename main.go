package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/phathdt/schema-manager/internal/schema"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "schema-manager",
		Usage: "Schema-first migration tool for Go applications (Prisma schema only)",
		Commands: []*cli.Command{
			{
				Name:  "init",
				Usage: "Initialize new schema with Prisma format",
				Action: func(c *cli.Context) error {
					fmt.Println("init called")
					return nil
				},
			},
			{
				Name:  "generate",
				Usage: "Generate migration from Prisma schema changes",
				Action: func(c *cli.Context) error {
					fmt.Println("generate called")
					return nil
				},
			},
			{
				Name:  "validate",
				Usage: "Validate Prisma schema",
				Action: func(c *cli.Context) error {
					ctx := context.Background()
					prismaSource := &schema.PrismaFileSource{Path: "schema.prisma"}
					_, err := prismaSource.LoadSchema(ctx)
					if err != nil {
						return cli.Exit("Failed to parse schema.prisma: "+err.Error(), 1)
					}
					fmt.Println("Schema valid")
					return nil
				},
			},
			{
				Name:  "show",
				Usage: "Show current schema",
				Action: func(c *cli.Context) error {
					fmt.Println("show called")
					return nil
				},
			},
			{
				Name:  "db",
				Usage: "Database operations",
				Subcommands: []*cli.Command{
					{
						Name:  "pull",
						Usage: "Import existing database to Prisma schema",
						Action: func(c *cli.Context) error {
							fmt.Println("db pull called")
							return nil
						},
					},
				},
			},
			{
				Name:  "migration",
				Usage: "Migration management",
				Subcommands: []*cli.Command{
					{
						Name:  "create",
						Usage: "Create migration file",
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
					},
					{
						Name:  "reset",
						Usage: "Reset migrations",
						Action: func(c *cli.Context) error {
							fmt.Println("migration reset called")
							return nil
						},
					},
					{
						Name:  "status",
						Usage: "Check migration status",
						Action: func(c *cli.Context) error {
							fmt.Println("migration status called")
							return nil
						},
					},
					{
						Name:  "resolve",
						Usage: "Mark migration as applied without running",
						Action: func(c *cli.Context) error {
							fmt.Println("migration resolve called")
							return nil
						},
					},
				},
			},
			{
				Name:  "push",
				Usage: "Generate and apply migration in one command",
				Action: func(c *cli.Context) error {
					fmt.Println("push called")
					return nil
				},
			},
			{
				Name:  "rollback",
				Usage: "Rollback schema to previous version",
				Action: func(c *cli.Context) error {
					fmt.Println("rollback called")
					return nil
				},
			},
			{
				Name:  "diff",
				Usage: "Diff schema.prisma and schema.prisma.next, print Goose migration SQL",
				Action: func(c *cli.Context) error {
					ctx := context.Background()
					currentSource := &schema.PrismaFileSource{Path: "schema.prisma"}
					current, err := currentSource.LoadSchema(ctx)
					if err != nil {
						return cli.Exit("Failed to parse schema.prisma: "+err.Error(), 1)
					}
					if _, err := os.Stat("schema.prisma.next"); err != nil {
						return cli.Exit("schema.prisma.next not found", 1)
					}
					targetSource := &schema.PrismaFileSource{Path: "schema.prisma.next"}
					target, err := targetSource.LoadSchema(ctx)
					if err != nil {
						return cli.Exit("Failed to parse schema.prisma.next: "+err.Error(), 1)
					}
					diff := schema.DiffSchemas(current, target)
					up := schema.GenerateMigrationSQL(diff)
					fmt.Println("-- +goose Up\n" + up)
					fmt.Println("\n-- +goose Down\n")
					return nil
				},
			},
		},
	}
	app.Run(os.Args)
}
