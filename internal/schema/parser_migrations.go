package schema

import (
	"context"
)

func ParseMigrationsToSchema(ctx context.Context, dir string) (*Schema, error) {
	// Use the new SQL parser-based approach
	return ApplyMigrationsFromDir(ctx, dir)
}

// These legacy functions are no longer needed with the new SQL parser
