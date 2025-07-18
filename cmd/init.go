package cmd

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func InitCommand() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "Initialize new schema with Prisma format",
		Action: func(c *cli.Context) error {
			fmt.Println("init called")
			return nil
		},
	}
}
