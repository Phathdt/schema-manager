package cmd

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/urfave/cli/v2"
)

// Version information (set during build or detected at runtime)
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

func init() {
	// If version is still "dev", try to get it from build info
	if Version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok {
			// Try to get version from module info
			if info.Main.Version != "" && info.Main.Version != "(devel)" {
				Version = info.Main.Version
			}

			// Try to get commit from build settings
			for _, setting := range info.Settings {
				switch setting.Key {
				case "vcs.revision":
					if len(setting.Value) >= 7 {
						Commit = setting.Value[:7]
					}
				case "vcs.time":
					if setting.Value != "" {
						// Parse time and format as date
						if strings.Contains(setting.Value, "T") {
							Date = strings.Split(setting.Value, "T")[0]
						}
					}
				}
			}
		}
	}
}

func VersionCommand() *cli.Command {
	return &cli.Command{
		Name:  "version",
		Usage: "Show version information",
		Action: func(ctx *cli.Context) error {
			fmt.Printf("schema-manager version %s\n", Version)
			fmt.Printf("Git commit: %s\n", Commit)
			fmt.Printf("Build date: %s\n", Date)
			fmt.Printf("Go version: %s\n", runtime.Version())
			fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
			return nil
		},
	}
}
