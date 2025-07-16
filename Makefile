build:
	go build -o schema-manager

run:
	go run main.go

test:
	go test ./...

tidy:
	go mod tidy

clean:
	rm -f schema-manager

help:
	@echo "Available targets:"
	@echo "  build   - Build the CLI binary"
	@echo "  run     - Run the CLI app"
	@echo "  test    - Run all tests"
	@echo "  tidy    - Clean up go.mod and go.sum"
	@echo "  clean   - Remove built binaries"
	@echo "  help    - Show this help message"
