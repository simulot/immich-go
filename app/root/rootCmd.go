package root

import (
	"context"
	"fmt"
	"os"

	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/app/archive"
	"github.com/simulot/immich-go/app/stack"
	"github.com/simulot/immich-go/app/upload"
	"github.com/simulot/immich-go/app/version"
	"github.com/spf13/cobra"
)

// RootImmichGoCommand creates and returns the root Cobra command for immich-go.
// It sets up the CLI structure, configuration handling, and adds all subcommands.
// Returns the root command and the application instance.
func RootImmichGoCommand(ctx context.Context) (*cobra.Command, *app.Application) {
	// Enable traverse run hooks to ensure PersistentPreRunE runs for all commands
	cobra.EnableTraverseRunHooks = true // doc: cobra/site/content/user_guide.md

	// Initialize the root Cobra command with basic metadata
	cmd := &cobra.Command{
		Use:     "immich-go",
		Short:   "Immich-go is a command line application to interact with the Immich application using its API",
		Long:    `An alternative to the immich-CLI command that doesn't depend on nodejs installation. It tries its best for importing google photos takeout archives.`,
		Version: app.Version,
	}

	// Create the application context
	a := app.New(ctx, cmd)

	flags := cmd.PersistentFlags()
	_ = a.OnErrors.Set("stop")
	a.RegisterFlags(flags)
	a.Log().RegisterFlags(flags)

	// Add all subcommands to the root command
	cmd.AddCommand(
		version.NewVersionCommand(ctx, a), // Version command to display app version
		upload.NewUploadCommand(ctx, a),   // Upload command for uploading assets
		archive.NewArchiveCommand(ctx, a), // Archive command for archiving assets
		stack.NewStackCommand(ctx, a),     // Stack command for managing stacks
	)

	// PersistentPreRunE is executed before any command runs, used for initialization
	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// Initialize configuration from the specified config file
		err := a.Config.Init(a.CfgFile)
		if err != nil {
			return err
		}

		// Process command-specific configuration
		err = a.Config.ProcessCommand(cmd)
		if err != nil {
			return err
		}

		// clip the number of concurrent tasks
		a.ConcurrentJobs = min(max(a.ConcurrentJobs, 1), 20)

		// Save configuration if the --save-config flag is set
		if save, _ := cmd.Flags().GetBool("save-config"); save {
			if err := a.Config.Save("immich-go.yaml"); err != nil {
				fmt.Fprintln(os.Stderr, "Can't save the configuration: ", err.Error())
				return err
			}
		}

		// Start the log
		err = a.Log().Open(cmd.Context(), cmd, a)

		return err
	}

	return cmd, a
}
