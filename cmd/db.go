package cmd

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func DbCommand() *cli.Command {
	return &cli.Command{
		Name:  "db",
		Usage: "Database operations",
		Subcommands: []*cli.Command{
			{
				Name:  "pull",
				Usage: "Import existing database to Prisma schema",
				Action: func(c *cli.Context) error {
					fmt.Println("db pull called")
					return nil
				},
			},
		},
	}
}
