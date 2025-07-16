# Schema Manager - Custom Migration Tool

## Overview

Schema Manager is a custom tool that provides a schema-first migration approach for Go applications, integrated with the Goose migration tool.

### Key Features

- **Schema-as-Code**: Define your database schema using Prisma Schema
- **Auto Migration Generation**: Automatically generate Goose migration files
- **Goose Integration**: Seamless integration with the existing Goose workflow

## Architecture

```
┌──────────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│  Prisma Schema File  │───▶│  Schema Manager  │───▶│  Goose Migration│
│   (schema.prisma)    │    │     (Go Tool)    │    │     (.sql)      │
└──────────────────────┘    └──────────────────┘    └─────────────────┘
```

## Schema Definition Format

### Prisma Schema Format (Native Support)

```prisma
// schema.prisma
datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

generator client {
  provider = "schema-manager"
  output   = "./migrations"
}

model Organization {
  id     Int    @id @default(autoincrement()) @map("id")
  name   String @map("name")
  apiKey String @unique @map("api_key")

  createdAt DateTime @default(now()) @map("created_at")
  updatedAt DateTime @updatedAt @map("updated_at")

  users User[]

  @@map("organizations")
}

model User {
  id             Int      @id @default(autoincrement())
  name           String   @db.VarChar(255)
  email          String   @unique @db.VarChar(255)
  organizationId Int      @map("organization_id")

  createdAt      DateTime @default(now()) @map("created_at")
  updatedAt      DateTime @updatedAt @map("updated_at")

  organization   Organization @relation(fields: [organizationId], references: [id], onDelete: Cascade)
  roles          UserRole[]

  @@index([email])
  @@index([createdAt])
  @@map("users")
}

model UserRole {
  userId Int    @map("user_id")
  role   String @db.VarChar(50)

  user   User   @relation(fields: [userId], references: [id], onDelete: Cascade)

  @@id([userId, role])
  @@map("user_roles")
}
```

## Core Components

### 1. Prisma Schema Parser

```go
type SchemaManager struct {
    currentSchema *PrismaSchema
    targetSchema  *PrismaSchema
    config        *Config
}

type PrismaSchema struct {
    Datasource *Datasource         `prisma:"datasource"`
    Generator  *Generator          `prisma:"generator"`
    Models     map[string]*Model   `prisma:"model"`
    Enums      map[string]*Enum    `prisma:"enum"`
}

type Datasource struct {
    Name     string `prisma:"name"`
    Provider string `prisma:"provider"`
    URL      string `prisma:"url"`
}

type Generator struct {
    Provider string `prisma:"provider"`
    Output   string `prisma:"output"`
}

type Model struct {
    Name       string             `prisma:"name"`
    Fields     []*Field           `prisma:"fields"`
    Attributes []*ModelAttribute  `prisma:"attributes"`
}

type Field struct {
    Name        string           `prisma:"name"`
    Type        string           `prisma:"type"`
    Attributes  []*FieldAttribute `prisma:"attributes"`
    IsOptional  bool             `prisma:"optional"`
    IsArray     bool             `prisma:"array"`
}

type FieldAttribute struct {
    Name string                 `prisma:"name"`
    Args []string               `prisma:"args"`
}

type ModelAttribute struct {
    Name string   `prisma:"name"`
    Args []string `prisma:"args"`
}
```

### 2. Migration Generator (Prisma-compatible)

```go
func (sm *SchemaManager) GenerateMigrationFromPrisma(diff *PrismaDiff) error {
    upSQL := sm.generateUpSQLFromPrisma(diff)
    downSQL := sm.generateDownSQLFromPrisma(diff)

    timestamp := time.Now().Format("20060102150405")
    filename := fmt.Sprintf("%s_%s.sql", timestamp, diff.Description)

    return sm.createGooseFile(filename, upSQL, downSQL)
}

func (sm *SchemaManager) generateUpSQLFromPrisma(diff *PrismaDiff) string {
    var statements []string

    // Handle model creation
    for _, model := range diff.ModelsAdded {
        statements = append(statements, sm.generateCreateTableFromModel(model))
    }

    // Handle field additions
    for _, change := range diff.FieldsAdded {
        statements = append(statements, sm.generateAddColumnFromField(change))
    }

    // Handle index creation from @@index
    for _, index := range diff.IndexesAdded {
        statements = append(statements, sm.generateCreateIndexFromAttribute(index))
    }

    // Handle relations
    for _, relation := range diff.RelationsAdded {
        statements = append(statements, sm.generateForeignKeyFromRelation(relation))
    }

    return strings.Join(statements, "\n\n")
}

func (sm *SchemaManager) generateCreateTableFromModel(model *Model) string {
    var columns []string
    var constraints []string

    for _, field := range model.Fields {
        if field.IsRelation() {
            continue // Skip relation fields
        }

        col := sm.generateColumnFromField(field)
        columns = append(columns, col)

        // Handle field attributes
        for _, attr := range field.Attributes {
            switch attr.Name {
            case "id":
                constraints = append(constraints, fmt.Sprintf("PRIMARY KEY (%s)", field.Name))
            case "unique":
                constraints = append(constraints, fmt.Sprintf("UNIQUE (%s)", field.Name))
            }
        }
    }

    // Handle model attributes
    for _, attr := range model.Attributes {
        switch attr.Name {
        case "id":
            // Composite primary key
            constraints = append(constraints, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(attr.Args, ", ")))
        case "index":
            // Will be handled separately
        }
    }

    tableName := sm.getTableNameFromModel(model)
    sql := fmt.Sprintf("CREATE TABLE %s (\n  %s", tableName, strings.Join(columns, ",\n  "))

    if len(constraints) > 0 {
        sql += ",\n  " + strings.Join(constraints, ",\n  ")
    }

    sql += "\n);"

    return sql
}
```

### 3. Prisma Schema Validator

```go
func (sm *SchemaManager) ValidatePrismaSchema(schema *PrismaSchema) error {
    var errors []error

    // Validate datasource
    if schema.Datasource == nil {
        errors = append(errors, fmt.Errorf("datasource block is required"))
    }

    // Validate models
    for name, model := range schema.Models {
        if err := sm.validateModel(name, model); err != nil {
            errors = append(errors, err)
        }
    }

    // Validate relations
    if err := sm.validateRelations(schema.Models); err != nil {
        errors = append(errors, err)
    }

    if len(errors) > 0 {
        return fmt.Errorf("schema validation failed: %v", errors)
    }

    return nil
}

func (sm *SchemaManager) validateModel(name string, model *Model) error {
    // Check if model has at least one @id field
    hasId := false
    for _, field := range model.Fields {
        for _, attr := range field.Attributes {
            if attr.Name == "id" {
                hasId = true
                break
            }
        }
    }

    // Check for @@id (composite key)
    for _, attr := range model.Attributes {
        if attr.Name == "id" {
            hasId = true
            break
        }
    }

    if !hasId {
        return fmt.Errorf("model %s must have at least one @id field or @@id attribute", name)
    }

    return nil
}
```

## CLI Commands

### Installation

```bash
go install github.com/phathdt/schema-manager@latest
```

### Basic Usage

```bash
# Initialize new schema with Prisma format
schema-manager init --prisma

# Generate migration from Prisma schema changes
schema-manager generate

# Validate Prisma schema
schema-manager validate

# Show current schema
schema-manager show

# Import existing database to Prisma schema
schema-manager db pull
```

### Manual Migration Commands

```bash
# Create empty migration file (like Prisma)
schema-manager migration create --name "add_custom_function"

# Create migration with custom SQL
schema-manager migration create --name "seed_data" --sql "INSERT INTO users..."

# Create migration from template
schema-manager migration create --name "add_indexes" --template "index"

# Reset migrations (like Prisma reset)
schema-manager migration reset

# Check migration status
schema-manager migration status

# Mark migration as applied without running
schema-manager migration resolve --applied 20231116123456

# Generate and apply in one command (like Prisma push)
schema-manager push --dev
```

### Advanced Usage

```bash
# Generate migration with custom name
schema-manager generate --name "add_user_indexes"

# Dry run - show what would be generated
schema-manager generate --dry-run

# Generate migration for specific environment
schema-manager generate --env production

# Rollback schema to previous version
schema-manager rollback --steps 1

# Prisma-like commands for rapid development
schema-manager db push --dev              # Push schema changes without migration
schema-manager db pull                    # Pull schema from database
schema-manager db reset                   # Reset database and run all migrations
schema-manager db migrate --dev           # Apply pending migrations
schema-manager db migrate deploy          # Deploy migrations to production

# Migration management
schema-manager migration diff             # Show diff between schema and database
schema-manager migration apply --up       # Apply specific migration
schema-manager migration apply --down     # Rollback specific migration
```

## Workflow

### 1. Development Workflow (Prisma-only)

```bash
# 1. Edit schema file
vim schema.prisma

# 2. Generate and apply migration
schema-manager db migrate --dev

# 3. If needed, create manual migration
schema-manager migration create --name "add_custom_logic"

# 4. Edit generated migration file
vim migrations/20231116123456_add_custom_logic.sql

# 5. Apply migration
schema-manager db migrate --dev
```

### 2. Production Workflow

```bash
# 1. Generate migration in development
schema-manager generate

# 2. Review and test migration
schema-manager generate --dry-run

# 3. Deploy to production
schema-manager db migrate deploy

# 4. Verify deployment
schema-manager migration status
```

### 3. Team Workflow

```bash
# 1. Pull latest schema changes
git pull origin main

# 2. Apply pending migrations
schema-manager db migrate --dev

# 3. Make schema changes
vim schema.prisma

# 4. Generate migration
schema-manager generate

# 5. Commit changes
git add schema.prisma migrations/
git commit -m "Add user profiles table"
```

## Prisma Schema Migration Examples

### 1. Basic Table Creation

**Before (empty schema.prisma):**
```prisma
datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

generator client {
  provider = "schema-manager"
  output   = "./migrations"
}
```

**After (adding Organization model):**
```prisma
datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

generator client {
  provider = "schema-manager"
  output   = "./migrations"
}

model Organization {
  id     Int    @id @default(autoincrement())
  name   String
  apiKey String @unique @map("api_key")

  createdAt DateTime @default(now()) @map("created_at")
  updatedAt DateTime @updatedAt @map("updated_at")

  @@map("organizations")
}
```

**Generated Migration:**
```sql
-- +goose Up
CREATE TABLE organizations (
  id SERIAL PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  api_key VARCHAR(255) UNIQUE NOT NULL,
  created_at TIMESTAMP DEFAULT now() NOT NULL,
  updated_at TIMESTAMP DEFAULT now() NOT NULL
);

-- +goose Down
DROP TABLE organizations;
```

### 2. Adding Relations

**Adding User model with Organization relation:**
```prisma
model User {
  id             Int      @id @default(autoincrement())
  name           String   @db.VarChar(255)
  email          String   @unique
  organizationId Int      @map("organization_id")

  organization   Organization @relation(fields: [organizationId], references: [id], onDelete: Cascade)

  @@map("users")
}

model Organization {
  id     Int    @id @default(autoincrement())
  name   String
  apiKey String @unique @map("api_key")

  createdAt DateTime @default(now()) @map("created_at")
  updatedAt DateTime @updatedAt @map("updated_at")

  users User[]  // Add relation

  @@map("organizations")
}
```

**Generated Migration:**
```sql
-- +goose Up
CREATE TABLE users (
  id SERIAL PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  email VARCHAR(255) UNIQUE NOT NULL,
  organization_id INTEGER NOT NULL,
  CONSTRAINT fk_users_organization
    FOREIGN KEY (organization_id)
    REFERENCES organizations(id)
    ON DELETE CASCADE
);

CREATE INDEX idx_users_email ON users(email);

-- +goose Down
DROP TABLE users;
```

### 3. Complex Schema Changes

**Adding indexes, enums, and composite keys:**
```prisma
enum UserStatus {
  ACTIVE
  INACTIVE
  PENDING
}

model User {
  id             Int        @id @default(autoincrement())
  name           String     @db.VarChar(255)
  email          String     @unique
  status         UserStatus @default(ACTIVE)
  organizationId Int        @map("organization_id")

  createdAt      DateTime   @default(now()) @map("created_at")
  updatedAt      DateTime   @updatedAt @map("updated_at")

  organization   Organization @relation(fields: [organizationId], references: [id])
  roles          UserRole[]

  @@index([organizationId])
  @@index([createdAt])
  @@index([status, organizationId])
  @@map("users")
}

model UserRole {
  userId    Int    @map("user_id")
  role      String @db.VarChar(50)
  grantedAt DateTime @default(now()) @map("granted_at")

  user      User   @relation(fields: [userId], references: [id], onDelete: Cascade)

  @@id([userId, role])
  @@map("user_roles")
}
```

## Troubleshooting

### Common Issues

1. **Schema Validation Errors**
   ```bash
   schema-manager validate --verbose
   ```

2. **Migration Generation Failures**
   ```bash
   schema-manager generate --debug
   ```

3. **Database Connection Issues**
   ```bash
   schema-manager test-connection
   ```

### Performance Optimization

- **Batch operations**: Group related schema changes
- **Index creation**: Create indexes concurrently when possible
- **Large table migrations**: Use pt-online-schema-change for MySQL

## Roadmap

### Phase 1 (MVP)
- [x] Basic schema parsing
- [x] Simple migration generation
- [x] Goose integration

### Phase 2 (Enhancement)
- [ ] Advanced migration strategies
- [ ] Data migration support
- [ ] Multiple database support

### Phase 3 (Advanced)
- [ ] Schema versioning
- [ ] Migration templates
- [ ] Plugin system
- [ ] Cloud deployment

## Contributing

### Development Setup

```bash
git clone https://github.com/phathdt/schema-manager
cd schema-manager
go mod download
make build
make test
```

### Testing

```bash
# Run all tests
make test

# Run integration tests
make test-integration

# Run with coverage
make test-coverage
```

## License

MIT License - see LICENSE file for details.
