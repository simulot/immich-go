// Package config provides configuration management for the immich-go application.
// It integrates Viper for configuration file handling, environment variables, and Cobra for CLI flags.
// The ConfigurationManager handles flag registration, binding, and origin tracking.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	// OriginCLI indicates the value came from command line flags
	OriginCLI = "cli"
	// OriginEnvironment indicates the value came from environment variables
	OriginEnvironment = "environment"
	// OriginConfigFile indicates the value came from a configuration file
	OriginConfigFile = "config file"
	// OriginDefault indicates the value is the default
	OriginDefault = "default"
)

// ConfigurationManager manages application configuration using Viper and Cobra.
// It handles flag registration, binding to configuration sources, and tracks the origin
// of configuration values (CLI, environment, config file, or default).
type ConfigurationManager struct {
	v         *viper.Viper      // Viper instance for configuration handling
	command   *cobra.Command    // Root command being processed
	processed bool              // Whether the command has been processed
	origins   map[string]string // Maps configuration keys to their origin source
}

// New creates a new ConfigurationManager instance.
// It initializes the Viper instance and internal maps for origins.
func New() *ConfigurationManager {
	return &ConfigurationManager{
		v:       viper.New(),
		origins: make(map[string]string),
	}
}

// GlobalConfigDir returns the path to the global configuration directory (~/.config/immich-go).
func GlobalConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".config", "immich-go"), nil
}

// GlobalConfigPath returns the full path to the global config file.
func GlobalConfigPath() (string, error) {
	dir, err := GlobalConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// Init initializes the configuration manager with the specified config file.
// Loading order (later overrides earlier):
// 1. Global config (~/.config/immich-go/config.yaml)
// 2. Local config (./immich-go.*)
// 3. Explicit --config file
// 4. Environment variables
// 5. CLI flags (handled in ProcessCommand)
func (cm *ConfigurationManager) Init(cfgFile string) error {
	cm.v.SetEnvPrefix("IMMICH_GO")
	cm.v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	cm.v.AutomaticEnv()

	// Load global config first (lowest file priority)
	if globalPath, err := GlobalConfigPath(); err == nil {
		if _, statErr := os.Stat(globalPath); statErr == nil {
			cm.v.SetConfigFile(globalPath)
			if err := cm.v.ReadInConfig(); err != nil {
				return fmt.Errorf("reading global config: %w", err)
			}
		}
	}

	// Merge local/explicit config on top
	if cfgFile != "" {
		cm.v.SetConfigFile(cfgFile)
	} else {
		cm.v.AddConfigPath(".")
		cm.v.SetConfigName("immich-go")
	}

	if err := cm.v.MergeInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}
	return nil
}

// SaveGlobal saves the given key-value pairs to the global config file.
// It uses a separate Viper instance to avoid polluting the runtime config.
// The file is written atomically with mode 0600 to protect API keys.
func SaveGlobal(values map[string]string) error {
	configDir, err := GlobalConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	configPath, _ := GlobalConfigPath()

	// Use a separate Viper to read-modify-write the global config
	gv := viper.New()
	gv.SetConfigFile(configPath)
	gv.SetConfigType("yaml")
	_ = gv.ReadInConfig() // ignore error if file doesn't exist yet

	for k, v := range values {
		gv.Set(k, v)
	}

	// Write to temp file with .yaml extension (Viper needs it), then rename
	tmpPath := configPath + ".new.yaml"
	if err := gv.WriteConfigAs(tmpPath); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("setting config permissions: %w", err)
	}
	if err := os.Rename(tmpPath, configPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("renaming config: %w", err)
	}
	return nil
}

// ProcessCommand processes the given command and its subcommands.
// It registers flags, binds them to Viper, applies configuration values,
// and tracks the origin of each configuration value.
// This method should be called once per root command.
func (cm *ConfigurationManager) ProcessCommand(cmd *cobra.Command) error {
	if cm.processed {
		return nil
	}
	cm.command = cmd
	err := cm.processCommand(cmd)
	cm.processed = true
	return err
}

// processCommand recursively processes a command and its subcommands.
// It defines flags, binds them to Viper, applies configuration values from various sources,
// and determines the origin of each configuration value.
func (cm *ConfigurationManager) processCommand(cmd *cobra.Command) error {
	// First, record CLI origins
	origins := make(map[string]string)
	recordOrigins := func(f *pflag.Flag) {
		key := getViperKey(cmd, f)
		if f.Changed {
			origins[key] = OriginCLI
		}
	}
	cmd.Flags().VisitAll(recordOrigins)
	cmd.PersistentFlags().VisitAll(recordOrigins)

	var err error
	// Bind and apply viper values
	if flagErr := cm.processFlagSet(cmd, cmd.Flags(), origins); flagErr != nil {
		err = errors.Join(err, flagErr)
	}
	if flagErr := cm.processFlagSet(cmd, cmd.PersistentFlags(), origins); flagErr != nil {
		err = errors.Join(err, flagErr)
	}

	// Set origins
	for k, v := range origins {
		cm.origins[k] = v
	}

	// Recurse for subcommands
	for _, c := range cmd.Commands() {
		err = errors.Join(err, cm.processCommand(c))
	}
	return err
}

// processFlagSet binds flags to Viper and applies configuration values for a given flag set.
func (cm *ConfigurationManager) processFlagSet(cmd *cobra.Command, fs *pflag.FlagSet, origins map[string]string) error {
	var err error
	fs.VisitAll(func(f *pflag.Flag) {
		key := getViperKey(cmd, f)
		_ = cm.v.BindPFlag(key, f) // can't fail in this context

		// For subcommand flags (e.g. "sync.down.server"), also check the flat key
		// ("server") as a fallback. This allows global config values to apply to
		// all subcommands without requiring per-command keys.
		effectiveKey := key
		if !f.Changed && !cm.v.IsSet(key) && strings.Contains(key, ".") {
			if cm.v.IsSet(f.Name) {
				effectiveKey = f.Name
			}
		}

		if !f.Changed && cm.v.IsSet(effectiveKey) {
			val := cm.v.Get(effectiveKey)

			err = errors.Join(fs.Set(f.Name, fmt.Sprintf("%v", val)))
			// Determine origin
			envKey := "IMMICH_GO_" + strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(key, ".", "_"), "-", "_"))
			if os.Getenv(envKey) != "" {
				origins[key] = OriginEnvironment
			} else {
				origins[key] = OriginConfigFile
			}
		} else if _, ok := origins[key]; !ok {
			origins[key] = OriginDefault
		}
	})
	return err
}

// getViperKey generates a Viper key for a flag based on the command hierarchy.
// For inherited flags (persistent flags from parent commands), it uses the parent's path.
// For local flags, it uses the current command's path.
func getViperKey(cmd *cobra.Command, f *pflag.Flag) string {
	isInherited := cmd.Parent() != nil && cmd.Parent().PersistentFlags().Lookup(f.Name) != nil
	if isInherited {
		// Use parent path
		path := []string{}
		for c := cmd.Parent(); c.Parent() != nil; c = c.Parent() {
			path = append([]string{c.Name()}, path...)
		}
		if len(path) > 0 {
			return strings.Join(path, ".") + "." + f.Name
		}
		return f.Name
	} else {
		// Use current path
		path := []string{}
		for c := cmd; c.Parent() != nil; c = c.Parent() {
			path = append([]string{c.Name()}, path...)
		}
		if len(path) > 0 {
			return strings.Join(path, ".") + "." + f.Name
		}
		return f.Name
	}
}

// GetFlagOrigin returns the origin source of a flag's value.
// Possible origins are: "cli", "environment", "config file", or "default".
func (cm *ConfigurationManager) GetFlagOrigin(cmd *cobra.Command, flag *pflag.Flag) string {
	key := getViperKey(cmd, flag)
	if origin, ok := cm.origins[key]; ok {
		return origin
	}
	return OriginDefault
}

// GetConfigFile returns the name of the configuration file used, if any.
// Returns empty string if no config file was loaded.
func (cm *ConfigurationManager) GetConfigFile() string {
	return cm.v.ConfigFileUsed()
}

// Save writes the current configuration to the specified file.
// The file format is determined by the file extension (e.g., .toml, .yaml, .json).
func (cm *ConfigurationManager) Save(fileName string) error {
	return cm.v.WriteConfigAs(fileName)
}
