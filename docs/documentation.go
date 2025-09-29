package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/simulot/immich-go/app/cmd"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// generate markdown documentation for environment variables
func main() {
	rootCmd, app := cmd.RootImmichGoCommand(context.Background())
	err := app.Config.ProcessCommand(rootCmd)
	if err != nil {
		panic(err)
	}

	// Generate environment variables documentation
	envVars := map[string]string{}
	collectEnvVars(rootCmd, []string{}, envVars)

	f, err := os.Create("docs/environment.md")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	fmt.Fprintln(f, "# Environment Variables")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "The following environment variables can be used to configure `immich-go`.")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "| Variable | Flag | Description |")
	fmt.Fprintln(f, "|----------|------|-------------|")

	keys := make([]string, 0, len(envVars))
	for k := range envVars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, ev := range keys {
		desc := envVars[ev]
		flag := "--" + strings.ReplaceAll(strings.ToLower(strings.TrimPrefix(ev, "IMMICH_GO_")), "_", "-")
		fmt.Fprintf(f, "| `%s` | `%s` | %s |\n", ev, flag, desc)
	}

	// Generate configuration file examples
	f, err = os.Create("docs/configuration.md")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	fmt.Fprintln(f, "# Configuration File")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "The configuration file is a TOML file. By default, `immich-go` looks for a file named `immich-go.toml` in the current directory.")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "## Global settings")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "```toml")
	fmt.Fprintln(f, "# Immich server URL")
	fmt.Fprintln(f, "server = \"http://immich:2283\"")
	fmt.Fprintln(f, "# Immich API key")
	fmt.Fprintln(f, "api-key = \"...\"")
	fmt.Fprintln(f, "# Log level (DEBUG|INFO|WARN|ERROR)")
	fmt.Fprintln(f, "log-level = \"INFO\"")
	fmt.Fprintln(f, "```")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "## Command specific settings")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "Settings for specific commands are nested under keys corresponding to the command path.")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "### `upload` command")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "```toml")
	fmt.Fprintln(f, "[upload]")
	fmt.Fprintln(f, "# Number of concurrent upload workers")
	fmt.Fprintln(f, "concurrent = 2")
	fmt.Fprintln(f, "# Create albums for assets")
	fmt.Fprintln(f, "create-albums = true")
	fmt.Fprintln(f, "```")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "### `upload from-folder` command")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "```toml")
	fmt.Fprintln(f, "[upload.from-folder]")
	fmt.Fprintln(f, "# Use the folder name as the album name")
	fmt.Fprintln(f, "folder-as-album = \"FOLDER\"")
	fmt.Fprintln(f, "```")
}

func collectEnvVars(cmd *cobra.Command, path []string, envVars map[string]string) {
	if cmd.HasFlags() {
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			if f.Name != "config" && f.Name != "help" {
				varName := "IMMICH_GO_"
				if len(path) > 0 {
					varName += strings.ToUpper(strings.ReplaceAll(strings.Join(path, "_"), "-", "_")) + "_"
				}
				varName += strings.ToUpper(strings.ReplaceAll(f.Name, "-", "_"))
				envVars[varName] = f.Usage
			}
		})
	}

	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() {
			continue
		}
		collectEnvVars(c, append(path, c.Name()), envVars)
	}
}
