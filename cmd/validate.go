package cmd

import (
	"context"
	"fmt"

	"github.com/phathdt/schema-manager/internal/schema"
	"github.com/urfave/cli/v2"
)

func ValidateCommand() *cli.Command {
	return &cli.Command{
		Name:  "validate",
		Usage: "Validate Prisma schema",
		Action: func(c *cli.Context) error {
			ctx := context.Background()
			prismaSource := &schema.PrismaFileSource{Path: "schema.prisma"}
			_, err := prismaSource.LoadSchema(ctx)
			if err != nil {
				return cli.Exit("Failed to parse schema.prisma: "+err.Error(), 1)
			}
			fmt.Println("Schema valid")
			return nil
		},
	}
}
