# Task Completion Workflow

## Commands to Run When Task is Completed

### 1. Code Quality Checks
```bash
# Format all Go files with comprehensive formatting
make format

# Check if Go files are properly formatted
make format-check

# Run tests to ensure nothing is broken
make test

# Tidy up Go modules
make tidy
```

### 2. Build Verification
```bash
# Build the binary to ensure it compiles
make build

# Optional: Test the built binary
./schema-manager validate
```

### 3. Pre-Commit Checks
```bash
# Check if working directory is clean
make check-clean

# Run pre-release checks (includes clean check and tests)
make pre-release
```

### 4. Git Workflow
```bash
# Commit and push changes (if ready)
make commit MSG="Descriptive commit message"
```

## Development Best Practices

### After Making Changes:
1. **Always run formatting**: `make format`
2. **Run tests**: `make test` 
3. **Build to verify**: `make build`
4. **Test the CLI**: `./schema-manager validate` or relevant command

### Before Committing:
1. **Format check**: `make format-check`
2. **Clean working directory**: `make check-clean`
3. **Final test run**: `make test`

### For Releases:
1. **Pre-release checks**: `make pre-release`
2. **Quick release**: `make quick-release VERSION=vX.Y.Z`
3. **Full release**: `make release VERSION=vX.Y.Z`

## Important Notes
- The project uses a comprehensive formatting pipeline that includes gofmt, goimports, golines, and gofumpt
- Always run `make format` after making code changes
- Tests should pass before any commit
- Clean working directory is enforced for releases