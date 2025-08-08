package cmd

import (
	"github.com/phathdt/schema-manager/internal/logger"
	"github.com/urfave/cli/v2"
)

func GetAllCommands() []*cli.Command {
	return []*cli.Command{
		GenerateCommand(),
		EmptyCommand(),
		ValidateCommand(),
		IntrospectCommand(),
		SyncCommand(),
		VersionCommand(),
	}
}

func SetupGlobalFlags(c *cli.Context) error {
	if c.Bool("verbose") {
		logger.SetVerbose(true)
	}
	return nil
}
