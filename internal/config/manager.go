// Package config provides configuration management for the immich-go application.
// It integrates Viper for configuration file handling, environment variables, and Cobra for CLI flags.
// The ConfigurationManager handles flag registration, binding, and origin tracking.
package config

import (
	"errors"
	"fmt"
	"os"
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

// Init initializes the configuration manager with the specified config file.
// If cfgFile is empty, it defaults to looking for "immich-go.toml" in the current directory.
// It sets up environment variable prefix and automatic environment binding.
func (cm *ConfigurationManager) Init(cfgFile string) error {
	if cfgFile != "" {
		cm.v.SetConfigFile(cfgFile)
	} else {
		cm.v.AddConfigPath(".")
		cm.v.SetConfigName("immich-go")
	}

	cm.v.SetEnvPrefix("IMMICH_GO")
	cm.v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	cm.v.AutomaticEnv()

	if err := cm.v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
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
		if !f.Changed && cm.v.IsSet(key) {
			val := cm.v.Get(key)

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

// Save writes the current configuration to the specified file.
// The file format is determined by the file extension (e.g., .toml, .yaml, .json).
func (cm *ConfigurationManager) Save(fileName string) error {
	return cm.v.WriteConfigAs(fileName)
}
