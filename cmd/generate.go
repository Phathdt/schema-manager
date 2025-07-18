package cmd

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func GenerateCommand() *cli.Command {
	return &cli.Command{
		Name:  "generate",
		Usage: "Generate migration from Prisma schema changes",
		Action: func(c *cli.Context) error {
			fmt.Println("generate called")
			return nil
		},
	}
}
