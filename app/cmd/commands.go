// Package cmd provides the command-line interface for immich-go using Cobra.
// It defines the root command and integrates subcommands for various operations.
package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/app/cmd/archive"
	"github.com/simulot/immich-go/app/cmd/stack"
	"github.com/simulot/immich-go/app/cmd/upload"
	"github.com/simulot/immich-go/internal/config"
	"github.com/spf13/cobra"
)

// RootImmichGoCommand creates and returns the root Cobra command for immich-go.
// It sets up the CLI structure, configuration handling, and adds all subcommands.
// Returns the root command and the application instance.
func RootImmichGoCommand(ctx context.Context) (*cobra.Command, *app.Application) {
	var cfgFile string
	cm := config.New()

	// Create the application context

	// Initialize the root Cobra command with basic metadata
	c := &cobra.Command{
		Use:     "immich-go",
		Short:   "Immich-go is a command line application to interact with the Immich application using its API",
		Long:    `An alternative to the immich-CLI command that doesn't depend on nodejs installation. It tries its best for importing google photos takeout archives.`,
		Version: app.Version,
		// PersistentPreRunE is executed before any command runs, used for initialization
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Initialize configuration from the specified config file
			err := cm.Init(cfgFile)
			if err != nil {
				return err
			}
			// Process command-specific configuration
			err = cm.ProcessCommand(cmd)
			if err != nil {
				return err
			}
			// Save configuration if the --save-config flag is set
			if save, _ := cmd.Flags().GetBool("save-config"); save {
				if err := cm.Save("immich-go.yaml"); err != nil {
					fmt.Fprintln(os.Stderr, "Can't save the configuration: ", err.Error())
				}
			}
			return nil
		},
	}

	// Define persistent flags available to all commands
	c.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./immich-go.yaml)")
	c.PersistentFlags().Bool("save-config", false, "Save the configuration to immich-go.yaml")

	// Enable traverse run hooks to ensure PersistentPreRunE runs for all commands
	cobra.EnableTraverseRunHooks = true // doc: cobra/site/content/user_guide.md

	// Create the application instance with context, command, and config
	a := app.New(ctx, c, cm)

	// Add all subcommands to the root command
	c.AddCommand(
		app.NewVersionCommand(ctx, a),     // Version command to display app version
		upload.NewUploadCommand(ctx, a),   // Upload command for uploading assets
		archive.NewArchiveCommand(ctx, a), // Archive command for archiving assets
		stack.NewStackCommand(ctx, a),     // Stack command for managing stacks
	)

	return c, a
}
