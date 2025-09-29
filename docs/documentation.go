package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/app/cmd"
	"github.com/simulot/immich-go/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

type EnvVarInfo struct {
	Path  string
	Flag  string
	Usage string
}

// generate markdown documentation for environment variables
func main() {
	rootCmd, app := cmd.RootImmichGoCommand(context.Background())
	err := app.Config.ProcessCommand(rootCmd)
	if err != nil {
		panic(err)
	}

	// Generate environment variables documentation
	envVars := map[string]EnvVarInfo{}
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

	// Group by path
	varsByPath := map[string][]struct {
		Name string
		Info EnvVarInfo
	}{}

	keys := make([]string, 0, len(envVars))
	for k := range envVars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		info := envVars[k]
		path := info.Path
		if path == "" {
			path = "Global"
		}
		varsByPath[path] = append(varsByPath[path], struct {
			Name string
			Info EnvVarInfo
		}{k, info})
	}

	// Get sorted paths
	paths := make([]string, 0, len(varsByPath))
	for p := range varsByPath {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	for _, p := range paths {
		fmt.Fprintf(f, "## %s\n\n", p)
		fmt.Fprintln(f, "| Variable | Flag | Description |")
		fmt.Fprintln(f, "|----------|------|-------------|")
		for _, v := range varsByPath[p] {
			fmt.Fprintf(f, "| `%s` | `--%s` | %s |\n", v.Name, v.Info.Flag, v.Info.Usage)
		}
		fmt.Fprintln(f, "")
	}

	// Generate configuration file examples
	generateConfigurationFileExamples(rootCmd, app)
}

func generateConfigurationFileExamples(rootCmd *cobra.Command, app *app.Application) {
	cfg := NewConfigFrom(rootCmd)
	if err := app.Config.Save("docs/config-example.toml"); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save config example: %v\n", err)
	}
	f, err := os.Create("docs/configuration.md")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	fmt.Fprintln(f, "# Configuration File")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "The configuration file can be a `TOML`, `YAML` or `JSON` file. By default, `immich-go` looks for a file named `immich-go.toml` in the current directory.")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "## Configuration file structure")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "````")
	fmt.Fprintln(f, "---")
	fmt.Fprintln(f, "title: TOML")
	fmt.Fprintln(f, "---")
	fmt.Fprintln(f, "```toml")
	b, err := toml.Marshal(cfg)
	if err != nil {
		panic(err)
	}
	fmt.Fprint(f, string(b))
	fmt.Fprintln(f, "```")
	fmt.Fprintln(f, "````")

	fmt.Fprintln(f, "````")
	fmt.Fprintln(f, "---")
	fmt.Fprintln(f, "title: YAML")
	fmt.Fprintln(f, "---")
	fmt.Fprintln(f, "```yaml")
	out := bytes.NewBuffer(nil)
	encoder := yaml.NewEncoder(out)
	encoder.SetIndent(2)
	err = encoder.Encode(cfg)
	if err != nil {
		panic(err)
	}
	fmt.Fprint(f, out.String())
	fmt.Fprintln(f, "```")
	fmt.Fprintln(f, "````")

	fmt.Fprintln(f, "````")
	fmt.Fprintln(f, "---")
	fmt.Fprintln(f, "title: JSON")
	fmt.Fprintln(f, "---")
	fmt.Fprintln(f, "```json")
	out.Reset()
	encoderJ := json.NewEncoder(out)
	encoderJ.SetIndent("", "  ")
	err = encoderJ.Encode(cfg)
	if err != nil {
		panic(err)
	}
	fmt.Fprint(f, out.String())
	fmt.Fprintln(f, "```")
	fmt.Fprintln(f, "````")
}

func NewConfigFrom(cmd *cobra.Command) any {
	m := config.ToMap(cmd)
	return m
}

func collectEnvVars(cmd *cobra.Command, path []string, envVars map[string]EnvVarInfo) {
	flags := cmd.Flags()
	if cmd.HasPersistentFlags() {
		flags.AddFlagSet(cmd.PersistentFlags())
	}
	if flags.HasFlags() {
		flags.VisitAll(func(f *pflag.Flag) {
			if f.Name != "config" && f.Name != "help" {
				varName := "IMMICH_GO_"
				if len(path) > 0 {
					varName += strings.ToUpper(strings.ReplaceAll(strings.Join(path, "_"), "-", "_")) + "_"
				}
				varName += strings.ToUpper(strings.ReplaceAll(f.Name, "-", "_"))
				current, ok := envVars[varName]
				if !ok || len(current.Path) > len(strings.Join(path, " ")) {
					envVars[varName] = EnvVarInfo{
						Path:  strings.Join(path, " "),
						Flag:  f.Name,
						Usage: f.Usage,
					}
				}
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
