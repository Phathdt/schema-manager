# Schema Manager - Simplified Migration Tool

## Overview

Schema Manager is a **focused** tool that compares your Prisma schema with existing migrations and generates missing migration files. It integrates seamlessly with Goose for migration execution.

### Key Features

- **Schema Diff**: Compare `schema.prisma` with existing migrations
- **Migration Generation**: Generate only missing migration files
- **Goose Integration**: Let Goose handle migration execution
- **Clean Architecture**: Focused on core functionality

## Philosophy

This tool follows the **Unix philosophy**: do one thing and do it well.

- **Schema Manager**: Compare schemas and generate migrations
- **Goose**: Execute migrations on database

## Architecture

```
┌──────────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│  Prisma Schema File  │───▶│  Schema Manager  │───▶│  Migration Files│
│   (schema.prisma)    │    │   (Diff & Gen)   │    │     (.sql)      │
└──────────────────────┘    └──────────────────┘    └─────────────────┘
                                                              │
                                                              ▼
                                                     ┌─────────────────┐
                                                     │      Goose      │
                                                     │   (Execute)     │
                                                     └─────────────────┘
```

## Schema Definition Format

### Prisma Schema Format

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

model Product {
  id        Int      @id @default(autoincrement())
  name      String
  slug      String   @unique
  description String?
  price     Int
  status    String
  createdAt DateTime @default(now()) @map("created_at")
  updatedAt DateTime @updatedAt @map("updated_at")

  @@index([name])
  @@map("products")
}
```

## Installation

```bash
go install github.com/phathdt/schema-manager@latest
```

## CLI Commands

### Core Commands

```bash
# Generate migration from schema changes
schema-manager generate --name "add_product_table"

# Validate Prisma schema
schema-manager validate

# Import existing database to schema.prisma (with baseline migration)
schema-manager introspect --output schema.prisma

# Sync database and schema.prisma (bi-directional)
schema-manager sync
```

### Goose Integration

```bash
# Execute migrations (using Goose)
goose up

# Rollback migrations (using Goose)
goose down

# Check migration status (using Goose)
goose status
```

## Workflow

### 1. Development Workflow (New Project)

```bash
# 1. Edit schema file
vim schema.prisma

# 2. Generate migration
schema-manager generate --name "add_new_fields"

# 3. Review generated migration
cat migrations/20231116123456_add_new_fields.sql

# 4. Apply migration with Goose
goose up
```

### 2. Migrating from Existing Database

```bash
# 1. Connect to existing database and generate schema.prisma
schema-manager introspect --output schema.prisma

# 2. Apply baseline migration (safe with conditional SQL)
goose up

# 3. Future development works normally
schema-manager generate --name "add_new_feature"
goose up
```

### 3. Sync Workflow (Database Modified Outside)

```bash
# 1. Someone added table/fields directly to database
# Check differences
schema-manager sync --check

# 2. Update schema.prisma with database changes
schema-manager sync --update-schema

# 3. Apply conditional migration (safe)
goose up

# 4. Continue normal development
schema-manager generate --name "add_more_features"
```

### 4. Team Workflow

```bash
# 1. Pull latest changes
git pull origin main

# 2. Apply pending migrations
goose up

# 3. Make schema changes
vim schema.prisma

# 4. Generate migration
schema-manager generate --name "update_user_table"

# 5. Test migration
goose up
goose down
goose up

# 6. Commit changes
git add schema.prisma migrations/
git commit -m "Update user table schema"
```

## Migration Examples

### 1. Baseline Migration (From Existing Database)

**Existing Database:**
```sql
-- Table already exists in database
CREATE TABLE users (
  id SERIAL PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  email VARCHAR(255) UNIQUE NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**Generated schema.prisma (from introspect):**
```prisma
model User {
  id        Int      @id @default(autoincrement())
  name      String
  email     String   @unique
  createdAt DateTime @default(now()) @map("created_at")

  @@map("users")
}
```

**Generated Baseline Migration (Goose-compatible):**
```sql
-- +goose Up
-- +goose StatementBegin
-- Baseline migration from existing database
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables
                   WHERE table_name = 'users') THEN
        CREATE TABLE users (
            id SERIAL PRIMARY KEY,
            name VARCHAR(255) NOT NULL,
            email VARCHAR(255) UNIQUE NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        );
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS users;
-- +goose StatementEnd
```

### 2. Initial Schema

**schema.prisma:**
```prisma
model Product {
  id        Int      @id @default(autoincrement())
  name      String
  createdAt DateTime @default(now()) @map("created_at")
  updatedAt DateTime @updatedAt @map("updated_at")

  @@map("products")
}
```

**Generated Migration:**
```sql
-- +goose Up
-- +goose StatementBegin
CREATE TABLE products (
  id SERIAL PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS products;
-- +goose StatementEnd
```

### 2. Adding Fields

**Updated schema.prisma:**
```prisma
model Product {
  id        Int      @id @default(autoincrement())
  name      String
  slug      String   @unique
  description String?
  price     Int
  status    String
  createdAt DateTime @default(now()) @map("created_at")
  updatedAt DateTime @updatedAt @map("updated_at")

  @@index([name])
  @@map("products")
}
```

**Generated Migration:**
```sql
-- +goose Up
-- +goose StatementBegin
ALTER TABLE products ADD COLUMN slug VARCHAR(255) NOT NULL;
CREATE UNIQUE INDEX idx_uniq_products_slug ON products(slug);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE products ADD COLUMN description VARCHAR(255);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE products ADD COLUMN price INTEGER NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE products ADD COLUMN status VARCHAR(255) NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE products DROP COLUMN IF EXISTS slug;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE products DROP COLUMN IF EXISTS description;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE products DROP COLUMN IF EXISTS price;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE products DROP COLUMN IF EXISTS status;
-- +goose StatementEnd
```

## Command Reference

### `generate`

Generate migration files from schema changes.

```bash
schema-manager generate --name "migration_name"
```

**Features:**
- Compares `schema.prisma` with existing migrations
- Generates only missing changes
- Creates Goose-compatible migration files
- Handles: tables, columns, indexes, constraints

### `validate`

Validate Prisma schema syntax.

```bash
schema-manager validate
```

**Features:**
- Checks Prisma schema syntax
- Validates required fields and attributes
- Reports parsing errors

### `introspect`

Import existing database structure into schema.prisma.

```bash
schema-manager introspect --output schema.prisma
```

**Features:**
- Connects to existing database
- Analyzes database structure (tables, columns, indexes, constraints)
- Generates schema.prisma from database structure
- Creates conditional baseline migration (Goose-compatible)
- Uses IF NOT EXISTS for safe migration execution
- **Automatically handles SSL connection issues** - Falls back to `sslmode=disable` if SSL connection fails

**SSL Configuration:**
```bash
# For local development (most common)
export DATABASE_URL="postgresql://user:password@localhost/dbname?sslmode=disable"

# For production with SSL
export DATABASE_URL="postgresql://user:password@host/dbname?sslmode=require"
```

### `sync`

Sync database schema with schema.prisma (bi-directional).

```bash
schema-manager sync                    # Interactive mode
schema-manager sync --check           # Show differences only
schema-manager sync --update-schema   # Update schema.prisma from DB
schema-manager sync --generate-migration # Generate migration from schema
```

**Features:**
- Compares database schema with schema.prisma
- **If database has more**: Updates schema.prisma + creates conditional migration
- **If schema.prisma has more**: Generates migration to update database
- **Goose-compatible**: All migrations use conditional SQL (IF NOT EXISTS)
- Interactive mode to confirm changes
- **Automatically handles SSL connection issues** - Falls back to `sslmode=disable` if SSL connection fails

## Best Practices

### 1. Migration Naming

```bash
# Good: descriptive names
schema-manager generate --name "add_user_email_index"
schema-manager generate --name "create_orders_table"

# Bad: generic names
schema-manager generate --name "update"
schema-manager generate --name "fix"
```

### 2. Schema Evolution

```bash
# Always generate migrations for schema changes
vim schema.prisma
schema-manager generate --name "add_user_avatar"

# Test migrations before committing
goose up
goose down
goose up
```

### 3. Team Collaboration

```bash
# Pull before making changes
git pull origin main
goose up

# Make changes and generate migration
vim schema.prisma
schema-manager generate --name "add_product_categories"

# Test and commit
goose up
git add .
git commit -m "Add product categories"
```

## Troubleshooting

### Common Issues

1. **Schema Validation Errors**
   ```bash
   schema-manager validate
   ```

2. **No Changes Detected**
   - Check if migrations are up to date
   - Verify schema.prisma has actual changes

3. **Migration Conflicts**
   - Use Goose to resolve conflicts
   - Ensure migrations are applied in order

4. **Database Connection Issues**
   ```bash
   # Check database connection
   export DATABASE_URL="postgresql://user:password@localhost/dbname"
   schema-manager introspect --output schema.prisma
   ```

5. **SSL Connection Issues**
   **Problem**: Error message "SSL is not enabled on the server"

   **Solution**: Schema Manager automatically handles SSL connection issues by falling back to `sslmode=disable`. However, you can also explicitly set it:

   ```bash
   # For local development (SSL disabled)
   export DATABASE_URL="postgresql://user:password@localhost/dbname?sslmode=disable"

   # For production (SSL required)
   export DATABASE_URL="postgresql://user:password@host/dbname?sslmode=require"

   # Common SSL modes:
   # - sslmode=disable    (no SSL)
   # - sslmode=require    (SSL required)
   # - sslmode=prefer     (SSL preferred, fallback to non-SSL)
   ```

6. **Table Already Exists (After Sync)**
   - Conditional migrations prevent this issue
   - All migrations use IF NOT EXISTS
   - Safe to run goose up multiple times

7. **Schema Out of Sync**
   ```bash
   # Check differences
   schema-manager sync --check

   # Fix schema.prisma
   schema-manager sync --update-schema

   # Or generate migration
   schema-manager sync --generate-migration
   ```

## Integration with Goose

### Setup Goose

```bash
# Install Goose
go install github.com/pressly/goose/v3/cmd/goose@latest

# Initialize Goose
goose create initial sql
```

### Using Together

```bash
# Generate migration with Schema Manager
schema-manager generate --name "add_users_table"

# Apply migration with Goose
goose up

# Check status with Goose
goose status
```

## Roadmap

### ✅ **Completed Features**

#### **Core Migration System**
- [x] Parse Prisma schema file (schema.prisma)
- [x] Validate Prisma schema structure
- [x] Generate Goose-compatible migration SQL from schema changes
- [x] Field changes detection (CREATE, ALTER, DROP)
- [x] Clean architecture with separate command files

#### **Database Integration**
- [x] **Introspect command** - Import existing database structure into schema.prisma
- [x] **Sync command** - Bi-directional sync between database and schema.prisma
- [x] **Database connection** - Connect to PostgreSQL for introspection/sync
- [x] **Conditional migrations** - Safe IF NOT EXISTS SQL for existing databases

#### **CLI Implementation**
- [x] `validate` command - Validate schema.prisma syntax
- [x] `generate` command - Generate migration from schema changes
- [x] `introspect` command - Import database structure to schema.prisma
- [x] `sync` command - Bi-directional schema synchronization

#### **Goose Integration**
- [x] **Compatible approach** - All migrations use Goose-compatible format
- [x] **Conditional SQL** - Safe migrations with IF NOT EXISTS
- [x] **Single source of truth** - Goose handles migration execution and tracking

### 🎯 **Current Focus**

#### **Simplified Tool Philosophy**
- **Schema Manager**: Compare schemas and generate migrations
- **Goose**: Execute migrations on database
- **Unix Philosophy**: Do one thing and do it well

#### **Supported Workflows**
1. **New Project**: Edit schema → Generate migration → Apply with Goose
2. **Existing Database**: Introspect → Baseline migration → Future development
3. **Schema Out of Sync**: Sync check → Update schema or generate migration
4. **Team Collaboration**: Pull → Apply migrations → Make changes → Generate → Commit

### 🚀 **Future Enhancements**

#### **Phase 1: Core Improvements**
- [ ] **Enhanced type mapping** - More PostgreSQL data types support
- [ ] **Relationship detection** - Foreign key constraints in introspection
- [ ] **Index optimization** - Better index handling in migrations
- [ ] **Error handling** - More detailed error messages and validation

#### **Phase 2: Advanced Features**
- [ ] **Multi-database support** - MySQL, SQLite support
- [ ] **Schema versioning** - Track schema changes over time
- [ ] **Migration rollback** - Generate reverse migrations
- [ ] **Performance optimization** - Faster introspection for large databases

#### **Phase 3: Developer Experience**
- [ ] **Interactive CLI** - Better prompts and confirmations
- [ ] **Configuration file** - Project-specific settings
- [ ] **Migration templates** - Custom migration templates
- [ ] **Integration tests** - Automated testing with real databases

#### **Phase 4: Advanced Sync**
- [ ] **Column modifications** - Handle column type changes in sync
- [ ] **Data migration** - Handle data transformations
- [ ] **Conflict resolution** - Better handling of schema conflicts
- [ ] **Batch operations** - Optimize large schema synchronizations

### 🎨 **Design Decisions**

#### **What We Keep**
- **Schema-first approach** - Prisma schema as source of truth
- **Goose integration** - Compatible with existing migration tools
- **Conditional migrations** - Safe for existing databases
- **Clean CLI** - Simple, focused command interface

#### **What We Removed**
- **Migration execution** - Let Goose handle this
- **Database state management** - Let Goose handle this
- **Complex diff workflows** - Use generate command instead
- **Redundant commands** - Removed diff command

### 🧪 **Testing Strategy**

#### **Current Testing**
- [x] Manual testing with PostgreSQL
- [x] Command-line interface testing
- [x] Migration generation testing

#### **Planned Testing**
- [ ] Unit tests for schema parsing
- [ ] Integration tests with real databases
- [ ] Migration generation tests
- [ ] CLI command tests
- [ ] Cross-platform compatibility tests

### 📋 **Known Limitations**

#### **Current Limitations**
- PostgreSQL only (MySQL, SQLite planned)
- Basic relationship detection
- Manual foreign key handling
- Limited custom type support

#### **Planned Improvements**
- Multi-database support
- Advanced relationship detection
- Custom type mapping configuration
- Performance optimizations for large schemas

## Contributing

### Development Setup

```bash
git clone https://github.com/phathdt/schema-manager
cd schema-manager
go mod download
go build -o schema-manager main.go
```

### Testing

```bash
# Build and test
go build -o schema-manager main.go
./schema-manager validate
./schema-manager generate --name "test_migration"
```

### Contributing Guidelines

1. **Focus on simplicity** - Keep the Unix philosophy
2. **Maintain Goose compatibility** - Don't break existing workflows
3. **Add comprehensive tests** - Test with real databases
4. **Document new features** - Update README and examples
5. **Follow Go best practices** - Clean, readable code

## License

MIT License - see LICENSE file for details.
