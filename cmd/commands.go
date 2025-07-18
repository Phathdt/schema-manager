package cmd

import "github.com/urfave/cli/v2"

func GetAllCommands() []*cli.Command {
	return []*cli.Command{
		GenerateCommand(),
		ValidateCommand(),
		IntrospectCommand(),
		SyncCommand(),
	}
}
