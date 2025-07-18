package cmd

import "github.com/urfave/cli/v2"

func GetAllCommands() []*cli.Command {
	return []*cli.Command{
		InitCommand(),
		GenerateCommand(),
		ValidateCommand(),
		ShowCommand(),
		DbCommand(),
		MigrationCommand(),
		PushCommand(),
		RollbackCommand(),
		DiffCommand(),
	}
}
