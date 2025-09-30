package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/simulot/immich-go/app/cmd"
	"github.com/simulot/immich-go/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

const (
	exampleTimeout = "20m"
)

type EnvVarInfo struct {
	Path  string
	Flag  string
	Usage string
}

// collectEnvVars recursively collects environment variable information from cobra commands
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

// NewConfigFrom creates a configuration map from cobra command flags
func NewConfigFrom(cmd *cobra.Command) any {
	m := config.ToMap(cmd)
	return m
}

// generateEnvVarsDoc generates markdown documentation for environment variables
func generateEnvVarsDoc(rootCmd *cobra.Command) {
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
}

// generateConfigurationFileExamples generates markdown documentation with configuration file examples
func generateConfigurationFileExamples(rootCmd *cobra.Command) {
	cfg := NewConfigFrom(rootCmd)

	// Set example values for documentation
	if configMap, ok := cfg.(map[string]interface{}); ok {
		// Set server and API key in upload section
		if upload, ok := configMap["upload"].(map[string]interface{}); ok {
			upload["server"] = "https://immich.app"
			upload["api-key"] = "YOUR-API-KEY"
			upload["client-timeout"] = exampleTimeout
			upload["device-uuid"] = "HOSTNAME"
		}
		// Set server and API key in stack section
		if stack, ok := configMap["stack"].(map[string]interface{}); ok {
			stack["server"] = "https://immich.app"
			stack["api-key"] = "YOUR-API-KEY"
			stack["client-timeout"] = exampleTimeout
			stack["device-uuid"] = "HOSTNAME"
		}
		// Set server and API key in archive.from-immich section
		if archive, ok := configMap["archive"].(map[string]interface{}); ok {
			if fromImmich, ok := archive["from-immich"].(map[string]interface{}); ok {
				fromImmich["from-server"] = "https://old.immich.app"
				fromImmich["from-api-key"] = "OLD-API-KEY"
				fromImmich["from-client-timeout"] = exampleTimeout
			}
		}
		// Set timeout in upload.from-immich section
		if upload, ok := configMap["upload"].(map[string]interface{}); ok {
			if fromImmich, ok := upload["from-immich"].(map[string]interface{}); ok {
				fromImmich["from-client-timeout"] = exampleTimeout
				fromImmich["from-server"] = "https://old.immich.app"
				fromImmich["from-api-key"] = "OLD-API-KEY"
			}
		}

		// Set date ranges to example value
		setDateRanges(configMap, "2024-01-15,2024-03-31")
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

// setDateRanges recursively sets all date-range fields in the configuration map
func setDateRanges(m map[string]interface{}, value string) {
	for k, v := range m {
		if k == "date-range" || k == "from-date-range" {
			// Try to set the DateRange value by calling Set on it
			if dr, ok := v.(interface{ Set(string) error }); ok {
				if err := dr.Set(value); err != nil {
					log.Printf("Error setting date range value: %v", err)
				}
			}
		} else if subMap, ok := v.(map[string]interface{}); ok {
			setDateRanges(subMap, value)
		}
	}
}

// main generates documentation for environment variables and configuration files
func main() {
	rootCmd, _ := cmd.RootImmichGoCommand(context.Background())
	err := rootCmd.Execute()
	if err != nil {
		panic(err)
	}

	generateEnvVarsDoc(rootCmd)
	generateConfigurationFileExamples(rootCmd)
}
