# Code Style and Conventions

## Go Code Style
- **Formatting**: Uses comprehensive formatting pipeline:
  - `gofmt` for standard Go formatting
  - `goimports` for import organization
  - `golines` for line length formatting (max 120 characters)
  - `gofumpt` with extra formatting rules
- **Package Structure**: Clean separation with `cmd/` for commands and `internal/` for implementation
- **Naming**: Standard Go conventions (CamelCase for exported, camelCase for unexported)
- **Error Handling**: Standard Go error handling patterns

## Project Structure
```
├── cmd/                    # CLI command implementations
│   ├── commands.go        # Command registration
│   ├── generate.go        # Generate command
│   ├── introspect.go      # Introspect command
│   ├── sync.go           # Sync command
│   ├── validate.go       # Validate command
│   └── version.go        # Version command
├── internal/             # Internal packages
│   └── schema/          # Schema parsing and generation
├── main.go              # Entry point
├── schema.prisma        # Example Prisma schema
├── go.mod              # Go module definition
└── Makefile            # Build automation
```

## CLI Framework
- Uses `urfave/cli/v2` for command-line interface
- Commands follow consistent structure with context handling
- Version information embedded at build time via ldflags

## Database Integration
- PostgreSQL support via `lib/pq` driver
- Conditional SQL generation for safe migrations
- Automatic SSL fallback handling

## Build and Release
- Make-based build system
- Multi-platform binary generation
- Semantic versioning with git tags
- Automated formatting and testing pipeline

## Documentation
- Comprehensive README with examples
- Command-line help integration
- Inline code documentation following Go conventions