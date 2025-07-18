# Tasks for Schema Manager CLI

## Core Functionality (Based on Schema-First Approach)
- [x] Parse Prisma schema file (schema.prisma)
- [x] Validate Prisma schema structure
- [x] Generate Goose-compatible migration SQL from schema changes
- [x] **FIXED**: Field changes detection (was missing, now works)
- [ ] **Database introspection** ✅ **NEW**: Import existing database structure into Prisma schema
- [ ] **Bi-directional sync** ✅ **NEW**: Sync database schema with schema.prisma (both ways)
- [ ] **Database connection** ✅ **NEW**: Connect to PostgreSQL database for introspection/sync
- [ ] ~~Apply migrations to the database~~ ❌ **REMOVE**: Let goose handle this
- [ ] ~~Rollback migrations~~ ❌ **REMOVE**: Let goose handle this
- [ ] ~~Show current schema state~~ ❌ **REMOVE**: Not core to diff generation

## CLI Implementation (Simplified)
- [ ] ~~Implement `init` command~~ ❌ **REMOVE**: Not needed
- [ ] ~~Implement `generate` command~~ ❌ **REMOVE**: Duplicate of migration create
- [x] Implement `validate` command ✅ **KEEP**: Useful for schema validation
- [ ] ~~Implement `show` command~~ ❌ **REMOVE**: Not core functionality
- [x] **Implement `migration create` command** ✅ **KEEP**: Core functionality - works perfectly
- [ ] ~~Implement `migration reset` command~~ ❌ **REMOVE**: Let goose handle this
- [ ] ~~Implement `migration status` command~~ ❌ **REMOVE**: Let goose handle this
- [ ] ~~Implement `migration resolve` command~~ ❌ **REMOVE**: Let goose handle this
- [ ] ~~Implement `push` command~~ ❌ **REMOVE**: Let goose handle this
- [ ] ~~Implement `rollback` command~~ ❌ **REMOVE**: Let goose handle this
- [ ] **Implement `diff` command** ✅ **KEEP**: Core functionality for comparing schemas
- [ ] **Implement `introspect` command** ✅ **NEW**: Import existing database structure into schema.prisma
- [ ] **Implement `sync` command** ✅ **NEW**: Bi-directional sync between database and schema.prisma

## Refactoring Completed ✅
- [x] **Refactored main.go** - Moved commands to separate files in cmd/ folder
- [x] **Clean Architecture** - Each command now has its own file
- [x] **Reduced main.go** - From 230 lines to 18 lines

## **RECOMMENDED SIMPLIFIED TOOL** 🎯
**Keep only these commands:**
1. `validate` - Validate schema.prisma syntax
2. `migration create --name <name>` - Generate migration from schema diff
3. `diff` - Show diff between schema.prisma and current migration state
4. **`introspect` - Import existing database structure into schema.prisma** ✅ **NEW**: For migrating from existing DB
5. **`sync` - Sync database schema with schema.prisma (bi-directional)** ✅ **NEW**: Core functionality for schema sync

**Remove these commands (let goose handle):**
- All migration execution: `reset`, `status`, `resolve`, `push`, `rollback`
- Schema operations: `init`, `generate`, `show`

## Testing & Quality
- [ ] Unit tests for schema parsing
- [ ] Unit tests for migration generation
- [ ] Integration tests for CLI commands
- [x] Error handling and user feedback (partially done)

## Documentation
- [x] Update README with usage examples
- [x] Document CLI commands and options
- [x] Add troubleshooting section

## **NEW USE CASES** 🎯

### Case 1: Migrating from Existing Database
**Problem**: User has existing database (created manually or with other tools), wants to use this tool for future migrations.

**Solution**: `introspect` command
```bash
schema-manager introspect --output schema.prisma
```

**Features**:
- Connect to existing database
- Analyze database structure (tables, columns, indexes, constraints)
- Generate schema.prisma file from database structure
- Create conditional baseline migration (compatible with Goose)
- Use IF NOT EXISTS for safe migration execution

**Workflow**:
```bash
# 1. Introspect existing database
schema-manager introspect --output schema.prisma

# 2. Apply baseline migration (safe with conditional SQL)
goose up

# 3. Future development works normally
schema-manager generate --name "add_new_feature"
goose up
```

### Case 2: Bi-directional Schema Sync
**Problem**: Database schema and schema.prisma might be out of sync (someone modified DB directly).

**Solution**: `sync` command
```bash
schema-manager sync
```

**Features**:
- Compare database schema with schema.prisma
- **If database has more**: Update schema.prisma + create conditional migration
- **If schema.prisma has more**: Generate migration to update database
- **Goose-compatible**: All migrations use conditional SQL (IF NOT EXISTS)
- Interactive mode to confirm changes

**Examples**:
```bash
# Database has additional fields -> Update schema.prisma + conditional migration
schema-manager sync --update-schema

# schema.prisma has additional fields -> Generate migration
schema-manager sync --generate-migration

# Interactive mode (default)
schema-manager sync
```

**Key Feature**: All migrations are **Goose-compatible** using conditional SQL:
```sql
-- Safe migration that won't fail if table exists
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables
                   WHERE table_name = 'users') THEN
        CREATE TABLE users (...);
    END IF;
END $$;
```

## **NEXT STEPS** 🚀
1. Remove unnecessary commands from cmd/ folder
2. Simplify main.go to only include core commands
3. **Implement `introspect` command** - Database to schema.prisma conversion
4. **Implement `sync` command** - Bi-directional schema synchronization
5. Update README to reflect new use cases
6. Add tests for core functionality
