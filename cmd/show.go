package cmd

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func ShowCommand() *cli.Command {
	return &cli.Command{
		Name:  "show",
		Usage: "Show current schema",
		Action: func(c *cli.Context) error {
			fmt.Println("show called")
			return nil
		},
	}
}
