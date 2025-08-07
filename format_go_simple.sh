#!/bin/bash

echo "Formatting all Go files..."
find . -name "*.go" -type f -exec gofmt -w {} \;
echo "✅ Go files formatted successfully!"
