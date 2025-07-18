package cmd

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func PushCommand() *cli.Command {
	return &cli.Command{
		Name:  "push",
		Usage: "Generate and apply migration in one command",
		Action: func(c *cli.Context) error {
			fmt.Println("push called")
			return nil
		},
	}
}
