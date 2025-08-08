# Variables
VERSION ?= $(shell git describe --tags --always --dirty)
COMMIT := $(shell git rev-parse --short HEAD)
DATE := $(shell date +%Y-%m-%d)

# Build configuration
BINARY_NAME = schema-manager
LDFLAGS = -ldflags "-X github.com/phathdt/schema-manager/cmd.Version=$(VERSION) -X github.com/phathdt/schema-manager/cmd.Commit=$(COMMIT) -X github.com/phathdt/schema-manager/cmd.Date=$(DATE)"

# Build targets
build:
	go build $(LDFLAGS) -o $(BINARY_NAME)

build-release:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe

# Development targets
run:
	go run main.go

test:
	go test ./...

tidy:
	go mod tidy

format:
	@echo "🎨 Formatting all Go files..."
	@find . -name "*.go" -type f -exec gofmt -w {} \;
	@echo "📦 Organizing imports..."
	@goimports -w .
	@echo "📏 Formatting line lengths..."
	@golines -w -m 120 .
	@echo "✨ Applying gofumpt formatting..."
	@gofumpt -extra -w .
	@echo "✅ Go files formatted successfully!"

format-go: format

format-check:
	@echo "🔍 Checking Go file formatting..."
	@if find . -name "*.go" -type f -exec gofmt -l {} \; | grep -q .; then \
		echo "❌ Some Go files are not properly formatted:"; \
		find . -name "*.go" -type f -exec gofmt -l {} \; | sed 's/^/  /'; \
		echo "Run 'make format' to fix formatting issues"; \
		exit 1; \
	else \
		echo "✅ All Go files are properly formatted"; \
	fi

clean:
	rm -f $(BINARY_NAME)*

# Git and versioning targets
version:
	@echo "Current version: $(VERSION)"
	@echo "Current commit: $(COMMIT)"
	@echo "Build date: $(DATE)"

# Create a new tag (usage: make tag VERSION=v1.0.0)
tag:
	@if [ -z "$(VERSION)" ]; then \
		echo "Usage: make tag VERSION=v1.0.0"; \
		exit 1; \
	fi
	@echo "Creating tag $(VERSION)..."
	git tag -a $(VERSION) -m "Release $(VERSION)"
	@echo "Tag $(VERSION) created successfully"

# Push tags to GitHub
push-tags:
	@echo "Pushing tags to GitHub..."
	git push origin --tags
	@echo "Tags pushed successfully"

# Delete a tag (usage: make delete-tag VERSION=v1.0.0)
delete-tag:
	@if [ -z "$(VERSION)" ]; then \
		echo "Usage: make delete-tag VERSION=v1.0.0"; \
		exit 1; \
	fi
	@echo "Deleting tag $(VERSION)..."
	git tag -d $(VERSION)
	git push origin :refs/tags/$(VERSION)
	@echo "Tag $(VERSION) deleted successfully"

# List all tags
list-tags:
	@echo "Available tags:"
	git tag -l --sort=-version:refname

# Complete release process
release: clean test build build-release tag push-tags
	@echo "Release $(VERSION) completed successfully!"
	@echo "Binaries built:"
	@ls -la $(BINARY_NAME)*

# Quick release (usage: make quick-release VERSION=v1.0.0)
quick-release:
	@if [ -z "$(VERSION)" ]; then \
		echo "Usage: make quick-release VERSION=v1.0.0"; \
		exit 1; \
	fi
	@echo "Creating quick release $(VERSION)..."
	make clean
	make test
	make build
	make tag VERSION=$(VERSION)
	make push-tags
	@echo "Quick release $(VERSION) completed!"

# Commit and push changes
commit:
	@if [ -z "$(MSG)" ]; then \
		echo "Usage: make commit MSG='commit message'"; \
		exit 1; \
	fi
	git add -A
	git commit -m "$(MSG)"
	git push origin main

# Check if working directory is clean
check-clean:
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "Working directory is not clean. Please commit changes first."; \
		git status --short; \
		exit 1; \
	fi
	@echo "Working directory is clean"

# Pre-release checks
pre-release: check-clean test
	@echo "Pre-release checks passed"

help:
	@echo "Available targets:"
	@echo ""
	@echo "Build targets:"
	@echo "  build         - Build the CLI binary"
	@echo "  build-release - Build binaries for multiple platforms"
	@echo "  run           - Run the CLI app"
	@echo "  clean         - Remove built binaries"
	@echo ""
	@echo "Development targets:"
	@echo "  test          - Run all tests"
	@echo "  tidy          - Run go mod tidy"
	@echo "  format        - Format all Go files"
	@echo "  format-go     - Alias for format"
	@echo "  format-check  - Check if Go files are properly formatted"
	@echo ""
	@echo "Git and versioning targets:"
	@echo "  version       - Show current version info"
	@echo "  tag           - Create a new tag (make tag VERSION=v1.0.0)"
	@echo "  push-tags     - Push tags to GitHub"
	@echo "  delete-tag    - Delete a tag (make delete-tag VERSION=v1.0.0)"
	@echo "  list-tags     - List all tags"
	@echo ""
	@echo "Release targets:"
	@echo "  release       - Complete release process (test + build + tag + push)"
	@echo "  quick-release - Quick release (make quick-release VERSION=v1.0.0)"
	@echo "  pre-release   - Run pre-release checks"
	@echo ""
	@echo "Git targets:"
	@echo "  commit        - Commit and push changes (make commit MSG='message')"
	@echo "  check-clean   - Check if working directory is clean"
	@echo ""
	@echo "Examples:"
	@echo "  make tag VERSION=v1.0.0"
	@echo "  make quick-release VERSION=v1.2.3"
	@echo "  make commit MSG='Add new feature'"
