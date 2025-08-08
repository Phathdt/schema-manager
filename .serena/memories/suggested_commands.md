# Suggested Commands for Schema Manager Development

## Build and Development Commands
```bash
# Build the CLI binary
make build

# Run the application
make run

# Run tests
make test

# Clean up binaries
make clean

# Show current version info
make version
```

## Code Quality Commands
```bash
# Format all Go files (comprehensive formatting)
make format

# Check if Go files are properly formatted
make format-check

# Run go mod tidy
make tidy
```

## Release Commands
```bash
# Build for multiple platforms
make build-release

# Complete release process (test + build + tag + push)
make release VERSION=v1.0.0

# Quick release
make quick-release VERSION=v1.0.0

# Create and push git tag
make tag VERSION=v1.0.0
make push-tags
```

## Git Commands
```bash
# Commit and push changes
make commit MSG="Add new feature"

# Check if working directory is clean
make check-clean

# List all tags
make list-tags
```

## Schema Manager CLI Commands
```bash
# Generate migration from schema changes
./schema-manager generate --name "add_product_table"

# Validate Prisma schema
./schema-manager validate

# Import existing database to schema.prisma
./schema-manager introspect --output schema.prisma

# Sync database and schema.prisma (bi-directional)
./schema-manager sync

# Check version
./schema-manager version
```

## System Commands (macOS)
```bash
# Standard Unix commands work on macOS Darwin
ls, cd, grep, find, git
```

## Help
```bash
# Show all available Make commands
make help
```