package schema

import (
	"context"
)

type Model struct {
	Name       string
	TableName  string
	Fields     []*Field
	Attributes []*ModelAttribute
}

type Enum struct {
	Name   string
	Values []string
}

type Field struct {
	Name       string
	ColumnName string
	Type       string
	Attributes []*FieldAttribute
	IsOptional bool
	IsArray    bool
}

type FieldAttribute struct {
	Name string
	Args []string
}

type ModelAttribute struct {
	Name string
	Args []string
}

type Schema struct {
	Models []*Model
	Enums  []*Enum
}

type SchemaSource interface {
	LoadSchema(ctx context.Context) (*Schema, error)
	SourceName() string
}

type PrismaFileSource struct {
	Path string
}

func (p *PrismaFileSource) LoadSchema(ctx context.Context) (*Schema, error) {
	return ParsePrismaFileToSchema(ctx, p.Path)
}

func (p *PrismaFileSource) SourceName() string {
	return "PrismaFileSource: " + p.Path
}

type MigrationsFolderSource struct {
	Dir string
}

func (m *MigrationsFolderSource) LoadSchema(ctx context.Context) (*Schema, error) {
	return ParseMigrationsToSchema(ctx, m.Dir)
}

func (m *MigrationsFolderSource) SourceName() string {
	return "MigrationsFolderSource: " + m.Dir
}
