package schema

import (
	"fmt"
	"log"
)

// TypeCastResult represents the result of a type cast operation
type TypeCastResult struct {
	CanCast        bool
	CastExpression string
	IsRisky        bool
	WarningMessage string
}

// GetPostgreSQLType maps Prisma types to PostgreSQL types
func GetPostgreSQLType(prismaType string) string {
	typeMap := map[string]string{
		"String":   "TEXT",
		"Int":      "INTEGER",
		"BigInt":   "BIGINT",
		"Float":    "DOUBLE PRECISION",
		"Decimal":  "NUMERIC",
		"Boolean":  "BOOLEAN",
		"DateTime": "TIMESTAMP",
		"Json":     "JSONB",
	}

	if pgType, ok := typeMap[prismaType]; ok {
		return pgType
	}
	return prismaType // fallback to original type
}

// CanCastType determines if a type can be cast from source to target
func CanCastType(sourceType, targetType string) TypeCastResult {
	sourcePG := GetPostgreSQLType(sourceType)
	targetPG := GetPostgreSQLType(targetType)

	// Same type - no casting needed
	if sourcePG == targetPG {
		return TypeCastResult{
			CanCast:        true,
			CastExpression: "",
			IsRisky:        false,
		}
	}

	// Define casting compatibility matrix
	castingRules := map[string]map[string]TypeCastResult{
		"BIGINT": {
			"INTEGER": {
				CanCast:        true,
				CastExpression: "::INTEGER",
				IsRisky:        true,
				WarningMessage: "Converting BIGINT to INTEGER may fail if values exceed INTEGER range (-2,147,483,648 to 2,147,483,647)",
			},
			"TEXT": {
				CanCast:        true,
				CastExpression: "::TEXT",
				IsRisky:        false,
			},
			"DOUBLE PRECISION": {
				CanCast:        true,
				CastExpression: "::DOUBLE PRECISION",
				IsRisky:        false,
			},
		},
		"INTEGER": {
			"BIGINT": {
				CanCast:        true,
				CastExpression: "::BIGINT",
				IsRisky:        false,
			},
			"TEXT": {
				CanCast:        true,
				CastExpression: "::TEXT",
				IsRisky:        false,
			},
			"DOUBLE PRECISION": {
				CanCast:        true,
				CastExpression: "::DOUBLE PRECISION",
				IsRisky:        false,
			},
			"BOOLEAN": {
				CanCast:        true,
				CastExpression: "::BOOLEAN",
				IsRisky:        false,
				WarningMessage: "Converting INTEGER to BOOLEAN: 0 = false, any other value = true",
			},
		},
		"TEXT": {
			"INTEGER": {
				CanCast:        true,
				CastExpression: "::INTEGER",
				IsRisky:        true,
				WarningMessage: "Converting TEXT to INTEGER may fail if text contains non-numeric values",
			},
			"BIGINT": {
				CanCast:        true,
				CastExpression: "::BIGINT",
				IsRisky:        true,
				WarningMessage: "Converting TEXT to BIGINT may fail if text contains non-numeric values",
			},
			"DOUBLE PRECISION": {
				CanCast:        true,
				CastExpression: "::DOUBLE PRECISION",
				IsRisky:        true,
				WarningMessage: "Converting TEXT to DOUBLE PRECISION may fail if text contains non-numeric values",
			},
			"BOOLEAN": {
				CanCast:        true,
				CastExpression: "::BOOLEAN",
				IsRisky:        true,
				WarningMessage: "Converting TEXT to BOOLEAN may fail if text is not 't', 'f', 'true', 'false', '1', or '0'",
			},
			"TIMESTAMP": {
				CanCast:        true,
				CastExpression: "::TIMESTAMP",
				IsRisky:        true,
				WarningMessage: "Converting TEXT to TIMESTAMP may fail if text is not in valid timestamp format",
			},
			"JSONB": {
				CanCast:        true,
				CastExpression: "::JSONB",
				IsRisky:        true,
				WarningMessage: "Converting TEXT to JSONB may fail if text is not valid JSON",
			},
		},
		"DOUBLE PRECISION": {
			"INTEGER": {
				CanCast:        true,
				CastExpression: "::INTEGER",
				IsRisky:        true,
				WarningMessage: "Converting DOUBLE PRECISION to INTEGER will truncate decimal places",
			},
			"BIGINT": {
				CanCast:        true,
				CastExpression: "::BIGINT",
				IsRisky:        true,
				WarningMessage: "Converting DOUBLE PRECISION to BIGINT will truncate decimal places",
			},
			"TEXT": {
				CanCast:        true,
				CastExpression: "::TEXT",
				IsRisky:        false,
			},
		},
		"BOOLEAN": {
			"TEXT": {
				CanCast:        true,
				CastExpression: "::TEXT",
				IsRisky:        false,
			},
			"INTEGER": {
				CanCast:        true,
				CastExpression: "CASE WHEN %s THEN 1 ELSE 0 END",
				IsRisky:        false,
				WarningMessage: "Converting BOOLEAN to INTEGER: true = 1, false = 0",
			},
		},
		"TIMESTAMP": {
			"TEXT": {
				CanCast:        true,
				CastExpression: "::TEXT",
				IsRisky:        false,
			},
		},
		"JSONB": {
			"TEXT": {
				CanCast:        true,
				CastExpression: "::TEXT",
				IsRisky:        false,
			},
		},
	}

	if sourceRules, ok := castingRules[sourcePG]; ok {
		if result, ok := sourceRules[targetPG]; ok {
			return result
		}
	}

	// No casting rule found
	return TypeCastResult{
		CanCast: false,
		WarningMessage: fmt.Sprintf(
			"No automatic casting available from %s to %s. Manual SQL migration required.",
			sourcePG,
			targetPG,
		),
	}
}

// LogTypeCastWarning logs warnings for risky type casts
func LogTypeCastWarning(tableName, columnName string, result TypeCastResult) {
	if result.IsRisky && result.WarningMessage != "" {
		log.Printf("WARNING: Type cast for %s.%s - %s", tableName, columnName, result.WarningMessage)
	}
}
