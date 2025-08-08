package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/phathdt/schema-manager/internal/schema"
	"github.com/urfave/cli/v2"
)

type SchemaDiff struct {
	MissingInSchema []TableInfo
	MissingInDB     []*schema.Model
	ModifiedTables  []TableComparison
}

type TableComparison struct {
	TableName       string
	MissingInSchema []ColumnInfo
	MissingInDB     []schema.Field
	ModifiedColumns []ColumnComparison
}

type ColumnComparison struct {
	ColumnName  string
	DBColumn    ColumnInfo
	SchemaField schema.Field
	Changes     []string
}

func SyncCommand() *cli.Command {
	return &cli.Command{
		Name:        "sync",
		Usage:       "Sync database schema with schema.prisma (bi-directional)",
		Description: "Compare database schema with schema.prisma and sync differences",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "check",
				Usage: "Only show differences without making changes",
			},
			&cli.BoolFlag{
				Name:  "update-schema",
				Usage: "Update schema.prisma with database changes",
			},
			&cli.BoolFlag{
				Name:  "generate-migration",
				Usage: "Generate migration for schema.prisma changes",
			},
		},
		Action: func(ctx *cli.Context) error {
			check := ctx.Bool("check")
			updateSchema := ctx.Bool("update-schema")
			generateMigration := ctx.Bool("generate-migration")

			if check {
				return runSyncCheck()
			}

			if updateSchema {
				return runSyncUpdateSchema()
			}

			if generateMigration {
				return runSyncGenerateMigration()
			}

			return runSyncInteractive()
		},
	}
}

func runSyncCheck() error {
	fmt.Println("🔍 Checking differences between database and schema.prisma...")

	diff, err := compareSchemas()
	if err != nil {
		return fmt.Errorf("failed to compare schemas: %w", err)
	}

	if isDiffEmpty(diff) {
		fmt.Println("✅ Database and schema.prisma are in sync!")
		return nil
	}

	printDifferences(diff)
	return nil
}

func runSyncUpdateSchema() error {
	fmt.Println("📝 Updating schema.prisma from database...")

	diff, err := compareSchemas()
	if err != nil {
		return fmt.Errorf("failed to compare schemas: %w", err)
	}

	if len(diff.MissingInSchema) == 0 && len(diff.ModifiedTables) == 0 {
		fmt.Println("✅ schema.prisma is already up to date!")
		return nil
	}

	if err := updateSchemaFromDB(diff); err != nil {
		return fmt.Errorf("failed to update schema: %w", err)
	}

	if err := createConditionalMigration(diff.MissingInSchema); err != nil {
		return fmt.Errorf("failed to create conditional migration: %w", err)
	}

	fmt.Println("✅ Schema updated successfully!")
	fmt.Println("🚀 Run 'goose up' to apply the conditional migration")

	return nil
}

func runSyncGenerateMigration() error {
	fmt.Println("🔄 Generating migration from schema.prisma...")

	diff, err := compareSchemas()
	if err != nil {
		return fmt.Errorf("failed to compare schemas: %w", err)
	}

	if len(diff.MissingInDB) == 0 && len(diff.ModifiedTables) == 0 {
		fmt.Println("✅ Database is already up to date!")
		return nil
	}

	migrationName := fmt.Sprintf("sync_%s", time.Now().Format("20060102150405"))
	if err := generateMigrationFromDiff(diff, migrationName); err != nil {
		return fmt.Errorf("failed to generate migration: %w", err)
	}

	fmt.Printf("✅ Migration created: migrations/%s.sql\n", migrationName)
	fmt.Println("🚀 Run 'goose up' to apply the migration")

	return nil
}

func runSyncInteractive() error {
	fmt.Println("🤖 Interactive sync mode")
	fmt.Println("Analyzing differences...")

	diff, err := compareSchemas()
	if err != nil {
		return fmt.Errorf("failed to compare schemas: %w", err)
	}

	if isDiffEmpty(diff) {
		fmt.Println("✅ Database and schema.prisma are in sync!")
		return nil
	}

	printDifferences(diff)

	fmt.Println("\nChoose an action:")
	fmt.Println("1. Update schema.prisma from database")
	fmt.Println("2. Generate migration from schema.prisma")
	fmt.Println("3. Exit without changes")

	var choice string
	fmt.Print("Enter choice (1-3): ")
	fmt.Scanln(&choice)

	switch choice {
	case "1":
		return runSyncUpdateSchema()
	case "2":
		return runSyncGenerateMigration()
	case "3":
		fmt.Println("Exiting without changes.")
		return nil
	default:
		fmt.Println("Invalid choice. Exiting.")
		return nil
	}
}

func compareSchemas() (*SchemaDiff, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is required")
	}

	db, err := connectWithSSLFallback(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	dbTables, err := introspectDatabase(db)
	if err != nil {
		return nil, fmt.Errorf("failed to introspect database: %w", err)
	}

	if !fileExists("schema.prisma") {
		return &SchemaDiff{
			MissingInSchema: dbTables,
			MissingInDB:     []*schema.Model{},
			ModifiedTables:  []TableComparison{},
		}, nil
	}

	schemaResult, err := schema.ParsePrismaFileToSchema(context.Background(), "schema.prisma")
	if err != nil {
		return nil, fmt.Errorf("failed to parse schema.prisma: %w", err)
	}
	schemaModels := schemaResult.Models

	diff := &SchemaDiff{
		MissingInSchema: []TableInfo{},
		MissingInDB:     []*schema.Model{},
		ModifiedTables:  []TableComparison{},
	}

	dbTableMap := make(map[string]TableInfo)
	for _, table := range dbTables {
		dbTableMap[table.TableName] = table
	}

	schemaTableMap := make(map[string]*schema.Model)
	for _, model := range schemaModels {
		tableName := model.TableName
		if tableName == "" {
			tableName = strings.ToLower(model.Name)
		}
		schemaTableMap[tableName] = model
	}

	for _, table := range dbTables {
		if _, exists := schemaTableMap[table.TableName]; !exists {
			diff.MissingInSchema = append(diff.MissingInSchema, table)
		}
	}

	for _, model := range schemaModels {
		tableName := model.TableName
		if tableName == "" {
			tableName = strings.ToLower(model.Name)
		}
		if _, exists := dbTableMap[tableName]; !exists {
			diff.MissingInDB = append(diff.MissingInDB, model)
		}
	}

	return diff, nil
}

func isDiffEmpty(diff *SchemaDiff) bool {
	return len(diff.MissingInSchema) == 0 &&
		len(diff.MissingInDB) == 0 &&
		len(diff.ModifiedTables) == 0
}

func printDifferences(diff *SchemaDiff) {
	if len(diff.MissingInSchema) > 0 {
		fmt.Println("\n📊 Tables in database but not in schema.prisma:")
		for _, table := range diff.MissingInSchema {
			fmt.Printf("  - %s (%d columns)\n", table.TableName, len(table.Columns))
		}
	}

	if len(diff.MissingInDB) > 0 {
		fmt.Println("\n📋 Models in schema.prisma but not in database:")
		for _, model := range diff.MissingInDB {
			fmt.Printf("  - %s (%d fields)\n", model.Name, len(model.Fields))
		}
	}

	if len(diff.ModifiedTables) > 0 {
		fmt.Println("\n🔄 Tables with differences:")
		for _, table := range diff.ModifiedTables {
			fmt.Printf("  - %s (modified)\n", table.TableName)
		}
	}
}

func updateSchemaFromDB(diff *SchemaDiff) error {
	if len(diff.MissingInSchema) == 0 {
		return nil
	}

	var existingSchema string
	if fileExists("schema.prisma") {
		content, err := os.ReadFile("schema.prisma")
		if err != nil {
			return fmt.Errorf("failed to read existing schema: %w", err)
		}
		existingSchema = string(content)
	} else {
		existingSchema = `datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

generator client {
  provider = "schema-manager"
  output   = "./migrations"
}

`
	}

	for _, table := range diff.MissingInSchema {
		modelString := generateModelString(table)
		existingSchema += modelString
	}

	if err := os.WriteFile("schema.prisma", []byte(existingSchema), 0o644); err != nil {
		return fmt.Errorf("failed to write schema file: %w", err)
	}

	return nil
}

func generateModelString(table TableInfo) string {
	var model strings.Builder

	model.WriteString(fmt.Sprintf("model %s {\n", toPascalCase(table.TableName)))

	for _, col := range table.Columns {
		model.WriteString(fmt.Sprintf("  %s", toCamelCase(col.ColumnName)))

		prismaType := mapDataTypeToPrisma(col.DataType)
		if col.IsNullable && !col.IsPrimaryKey {
			prismaType += "?"
		}
		model.WriteString(fmt.Sprintf(" %s", prismaType))

		var attributes []string
		if col.IsPrimaryKey {
			attributes = append(attributes, "@id")
		}
		if col.IsAutoIncrement {
			attributes = append(attributes, "@default(autoincrement())")
		}
		if col.IsUnique && !col.IsPrimaryKey {
			attributes = append(attributes, "@unique")
		}
		if col.ColumnName != toCamelCase(col.ColumnName) {
			attributes = append(attributes, fmt.Sprintf("@map(\"%s\")", col.ColumnName))
		}

		if len(attributes) > 0 {
			model.WriteString(" " + strings.Join(attributes, " "))
		}

		model.WriteString("\n")
	}

	model.WriteString(fmt.Sprintf("\n  @@map(\"%s\")\n", table.TableName))
	model.WriteString("}\n\n")

	return model.String()
}

func createConditionalMigration(tables []TableInfo) error {
	if len(tables) == 0 {
		return nil
	}

	migrationContent := generateConditionalMigration(tables)
	timestamp := time.Now().Format("20060102150405")
	migrationFile := fmt.Sprintf("migrations/%s_sync_from_database.sql", timestamp)

	if err := createMigrationsDir(); err != nil {
		return fmt.Errorf("failed to create migrations directory: %w", err)
	}

	if err := os.WriteFile(migrationFile, []byte(migrationContent), 0o644); err != nil {
		return fmt.Errorf("failed to write migration file: %w", err)
	}

	fmt.Printf("✅ Created conditional migration: %s\n", migrationFile)
	return nil
}

func generateConditionalMigration(tables []TableInfo) string {
	var migration strings.Builder

	migration.WriteString("-- +goose Up\n")
	migration.WriteString("-- +goose StatementBegin\n")
	migration.WriteString("-- Conditional migration from database sync\n")
	migration.WriteString("-- Tables already exist in database\n\n")

	for _, table := range tables {
		migration.WriteString("DO $$\n")
		migration.WriteString("BEGIN\n")
		migration.WriteString(
			fmt.Sprintf(
				"    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = '%s') THEN\n",
				table.TableName,
			),
		)
		migration.WriteString(fmt.Sprintf("        CREATE TABLE %s (\n", table.TableName))

		var columnDefs []string
		for _, col := range table.Columns {
			colDef := fmt.Sprintf("            %s %s", col.ColumnName, mapDataTypeToSQL(col.DataType))

			if col.IsPrimaryKey {
				colDef += " PRIMARY KEY"
			}
			if col.IsAutoIncrement {
				colDef = strings.Replace(colDef, mapDataTypeToSQL(col.DataType), "SERIAL", 1)
			}
			if !col.IsNullable && !col.IsPrimaryKey {
				colDef += " NOT NULL"
			}
			if col.IsUnique && !col.IsPrimaryKey {
				colDef += " UNIQUE"
			}
			if col.DefaultValue.Valid && !col.IsAutoIncrement {
				colDef += fmt.Sprintf(" DEFAULT %s", col.DefaultValue.String)
			}

			columnDefs = append(columnDefs, colDef)
		}

		migration.WriteString(strings.Join(columnDefs, ",\n"))
		migration.WriteString("\n        );\n")
		migration.WriteString("    END IF;\n")
		migration.WriteString("END $$;\n\n")
	}

	migration.WriteString("-- +goose StatementEnd\n\n")
	migration.WriteString("-- +goose Down\n")
	migration.WriteString("-- +goose StatementBegin\n")
	migration.WriteString("-- Note: This migration represents existing tables\n")
	migration.WriteString("-- Dropping them might cause data loss\n")

	for i := len(tables) - 1; i >= 0; i-- {
		migration.WriteString(fmt.Sprintf("-- DROP TABLE IF EXISTS %s;\n", tables[i].TableName))
	}

	migration.WriteString("-- +goose StatementEnd\n")

	return migration.String()
}

func generateMigrationFromDiff(diff *SchemaDiff, migrationName string) error {
	if len(diff.MissingInDB) == 0 && len(diff.ModifiedTables) == 0 {
		return nil
	}

	var migration strings.Builder

	migration.WriteString("-- +goose Up\n")
	migration.WriteString("-- +goose StatementBegin\n")
	migration.WriteString("-- Migration generated from schema.prisma sync\n\n")

	for _, model := range diff.MissingInDB {
		migration.WriteString(fmt.Sprintf("-- Create table for model %s\n", model.Name))
		migration.WriteString("-- TODO: Implement table creation from schema model\n")
		migration.WriteString("-- This requires parsing Prisma model fields to SQL\n\n")
	}

	migration.WriteString("-- +goose StatementEnd\n\n")
	migration.WriteString("-- +goose Down\n")
	migration.WriteString("-- +goose StatementBegin\n")

	for i := len(diff.MissingInDB) - 1; i >= 0; i-- {
		model := diff.MissingInDB[i]
		tableName := model.TableName
		if tableName == "" {
			tableName = strings.ToLower(model.Name)
		}
		migration.WriteString(fmt.Sprintf("DROP TABLE IF EXISTS %s;\n", tableName))
	}

	migration.WriteString("-- +goose StatementEnd\n")

	migrationFile := fmt.Sprintf("migrations/%s.sql", migrationName)

	if err := createMigrationsDir(); err != nil {
		return fmt.Errorf("failed to create migrations directory: %w", err)
	}

	if err := os.WriteFile(migrationFile, []byte(migration.String()), 0o644); err != nil {
		return fmt.Errorf("failed to write migration file: %w", err)
	}

	return nil
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}
