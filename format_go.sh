#!/bin/bash

set -e

echo "🔍 Finding all Go files..."
GO_FILES=$(find . -name "*.go" -type f)

if [ -z "$GO_FILES" ]; then
    echo "❌ No Go files found in the current directory and subdirectories."
    exit 1
fi

echo "📁 Found $(echo "$GO_FILES" | wc -l) Go files:"
echo "$GO_FILES" | sed 's/^/  /'

echo ""
echo "🎨 Formatting Go files with gofmt..."

FORMATTED_COUNT=0
ERROR_COUNT=0

for file in $GO_FILES; do
    echo "  Formatting: $file"

    if gofmt -w "$file"; then
        echo "    ✅ Successfully formatted"
        ((FORMATTED_COUNT++))
    else
        echo "    ❌ Error formatting $file"
        ((ERROR_COUNT++))
    fi
done

echo ""
echo "📊 Summary:"
echo "  ✅ Successfully formatted: $FORMATTED_COUNT files"
echo "  ❌ Errors: $ERROR_COUNT files"

if [ $ERROR_COUNT -eq 0 ]; then
    echo "🎉 All Go files have been formatted successfully!"
    exit 0
else
    echo "⚠️  Some files had formatting errors. Please check the output above."
    exit 1
fi
