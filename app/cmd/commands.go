package cmd

import (
	"context"

	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/app/cmd/archive"
	"github.com/simulot/immich-go/app/cmd/stack"
	"github.com/simulot/immich-go/app/cmd/upload"
	"github.com/simulot/immich-go/internal/configuration"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Run immich-go
func RootImmichGoCommand(ctx context.Context) (*cobra.Command, *app.Application) {
	viper.SetEnvPrefix("IMMICHGO")

	// Add the root command
	c := &cobra.Command{
		Use:     "immich-go",
		Short:   "Immich-go is a command line application to interact with the Immich application using its API",
		Long:    `An alternative to the immich-CLI command that doesn't depend on nodejs installation. It tries its best for importing google photos takeout archives.`,
		Version: app.Version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Initialize configuration system
			return initializeConfig(cmd)
		},
	}
	cobra.EnableTraverseRunHooks = true // doc: cobra/site/content/user_guide.md

	// Create the application context
	a := app.New(ctx, c)

	// Add config generation command
	c.AddCommand(NewConfigCommand(ctx, a))

	// add immich-go commands
	c.AddCommand(
		app.NewVersionCommand(ctx, a),
		upload.NewUploadCommand(ctx, a),
		archive.NewArchiveCommand(ctx, a),
		stack.NewStackCommand(ctx, a),
	)

	return c, a
}

// initializeConfig sets up the configuration system
func initializeConfig(cmd *cobra.Command) error {
	configFile, _ := cmd.Flags().GetString("config")
	return configuration.InitializeConfig(configFile)
}

// NewConfigCommand creates a command for managing configuration files
func NewConfigCommand(ctx context.Context, a *app.Application) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration files",
		Long:  `Create and manage immich-go configuration files. Supports JSON, YAML, and TOML formats.`,
	}

	// Sub-command to generate a sample configuration file
	generateCmd := &cobra.Command{
		Use:   "generate [filename]",
		Short: "Generate a sample configuration file",
		Long: `Generate a sample configuration file with default values and documentation.
If no filename is provided, it will create a config file in the default location.
Supported formats: .json, .yaml, .yml, .toml`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var filename string
			if len(args) > 0 {
				filename = args[0]
			} else {
				filename = configuration.DefaultConfigFile()
				// Change default to YAML for better readability
				filename = filename[:len(filename)-5] + ".yaml"
			}

			err := configuration.WriteConfigFile(filename)
			if err != nil {
				return err
			}

			a.Log().Message("Configuration file created: %s", filename)
			a.Log().Message("Edit this file with your server details and API keys.")
			return nil
		},
	}

	// Sub-command to validate configuration
	validateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate the current configuration",
		Long:  `Validate the current configuration file and show resolved values from config file, environment variables, and defaults.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := configuration.GetConfiguration()
			if err != nil {
				return err
			}

			a.Log().Message("Configuration validation successful!")
			a.Log().Message("Server URL: %s", config.Server.URL)
			if config.Server.APIKey != "" {
				a.Log().Message("API Key: %s", maskAPIKey(config.Server.APIKey))
			}
			if config.Server.AdminAPIKey != "" {
				a.Log().Message("Admin API Key: %s", maskAPIKey(config.Server.AdminAPIKey))
			}
			a.Log().Message("Concurrent uploads: %d", config.Upload.ConcurrentUploads)
			a.Log().Message("Log level: %s", config.Logging.Level)

			if configFile := viper.ConfigFileUsed(); configFile != "" {
				a.Log().Message("Using config file: %s", configFile)
			}

			return nil
		},
	}

	cmd.AddCommand(generateCmd, validateCmd)
	return cmd
}

// maskAPIKey masks an API key for display purposes
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}
