package cmd

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func RollbackCommand() *cli.Command {
	return &cli.Command{
		Name:  "rollback",
		Usage: "Rollback schema to previous version",
		Action: func(c *cli.Context) error {
			fmt.Println("rollback called")
			return nil
		},
	}
}
