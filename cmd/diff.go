package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/phathdt/schema-manager/internal/schema"
	"github.com/urfave/cli/v2"
)

func DiffCommand() *cli.Command {
	return &cli.Command{
		Name:  "diff",
		Usage: "Diff schema.prisma and schema.prisma.next, print Goose migration SQL",
		Action: func(c *cli.Context) error {
			ctx := context.Background()
			currentSource := &schema.PrismaFileSource{Path: "schema.prisma"}
			current, err := currentSource.LoadSchema(ctx)
			if err != nil {
				return cli.Exit("Failed to parse schema.prisma: "+err.Error(), 1)
			}
			if _, err := os.Stat("schema.prisma.next"); err != nil {
				return cli.Exit("schema.prisma.next not found", 1)
			}
			targetSource := &schema.PrismaFileSource{Path: "schema.prisma.next"}
			target, err := targetSource.LoadSchema(ctx)
			if err != nil {
				return cli.Exit("Failed to parse schema.prisma.next: "+err.Error(), 1)
			}
			diff := schema.DiffSchemas(current, target)
			up := schema.GenerateMigrationSQL(diff)
			fmt.Println("-- +goose Up\n" + up)
			fmt.Println("\n-- +goose Down\n")
			return nil
		},
	}
}
