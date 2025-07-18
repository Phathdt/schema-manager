package main

import (
	"os"

	"github.com/phathdt/schema-manager/cmd"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:     "schema-manager",
		Usage:    "Schema-first migration tool for Go applications (Prisma schema only)",
		Version:  cmd.Version,
		Commands: cmd.GetAllCommands(),
	}
	app.Run(os.Args)
}
