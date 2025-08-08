package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/urfave/cli/v2"
)

func EmptyCommand() *cli.Command {
	return &cli.Command{
		Name:  "empty",
		Usage: "Create an empty migration file for manual SQL writing",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "name", Usage: "Migration name", Required: true},
		},
		Action: func(c *cli.Context) error {
			name := c.String("name")
			ts := time.Now().Format("20060102150405")

			// Create migrations directory if it doesn't exist
			os.MkdirAll("migrations", 0o755)

			filename := "migrations/" + ts + "_" + name + ".sql"
			f, err := os.Create(filename)
			if err != nil {
				return cli.Exit("Failed to create migration file: "+err.Error(), 1)
			}
			defer f.Close()

			// Write empty goose template
			template := `-- +goose Up
-- +goose StatementBegin
-- Write your SQL here (e.g., CREATE INDEX, TRIGGER, FUNCTION, etc.)

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Write the rollback SQL here

-- +goose StatementEnd
`
			f.WriteString(template)
			fmt.Println("Created empty migration:", filename)
			fmt.Println("You can now edit this file to add your custom SQL statements.")
			return nil
		},
	}
}
