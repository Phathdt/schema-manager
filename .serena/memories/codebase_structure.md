# Codebase Structure

## Directory Layout
```
schema-manager/
├── cmd/                    # CLI command implementations
│   ├── commands.go        # Command registration and CLI app setup
│   ├── generate.go        # Migration generation command
│   ├── introspect.go      # Database introspection command  
│   ├── sync.go           # Bi-directional sync command
│   ├── validate.go       # Schema validation command
│   └── version.go        # Version information command
├── internal/             # Internal implementation packages
│   └── schema/          # Core schema processing logic
│       ├── generate.go   # Migration generation logic
│       ├── diff.go      # Schema comparison logic
│       ├── type_cast.go # Type conversion utilities
│       ├── source.go    # Source file handling
│       ├── parser_prisma.go      # Prisma schema parser
│       └── parser_migrations.go  # Migration file parser
├── .serena/             # Serena MCP server data
├── .claude/             # Claude Code configuration
├── main.go              # Application entry point
├── schema.prisma        # Example/test Prisma schema file
├── schema.go            # Generated schema definitions
├── go.mod               # Go module definition
├── go.sum               # Go module checksums
├── Makefile             # Build automation and development commands
├── format_go.sh         # Comprehensive Go formatting script
├── format_go_simple.sh  # Simple Go formatting script
├── README.md            # Comprehensive project documentation
└── .gitignore           # Git ignore patterns
```

## Key Components

### Entry Point
- `main.go`: Simple entry point that sets up CLI app using urfave/cli/v2

### Command Layer (`cmd/`)
- `commands.go`: Central command registration
- Individual command files handle CLI interface and orchestration
- Each command delegates business logic to internal packages

### Business Logic (`internal/schema/`)
- `parser_prisma.go`: Parses Prisma schema files
- `parser_migrations.go`: Parses existing migration files
- `diff.go`: Compares schemas to detect changes
- `generate.go`: Generates SQL migration files
- `type_cast.go`: Handles type conversions between Prisma and SQL types
- `source.go`: File system operations and source handling

### Configuration Files
- `schema.prisma`: Sample Prisma schema demonstrating project usage
- `Makefile`: Comprehensive build system with development, testing, and release targets

## Architecture Patterns
- **Clean Architecture**: Clear separation between CLI interface and business logic
- **Single Responsibility**: Each package has focused responsibility  
- **Command Pattern**: CLI commands implemented as separate modules
- **Parser Pattern**: Dedicated parsers for different file formats