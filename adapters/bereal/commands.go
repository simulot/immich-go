package bereal

import (
	"context"
	"io/fs"
	"os"

	"github.com/simulot/immich-go/adapters"
	"github.com/simulot/immich-go/app"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// ImportBeRealCmd holds options for importing BeReal memories
type ImportBeRealCmd struct {
	app         *app.Application
	BerealAlbum bool
}

func (ibc *ImportBeRealCmd) RegisterFlags(flags *pflag.FlagSet, cmd *cobra.Command) {
	// No additional flags for BeReal at this time
	flags.BoolVar(&ibc.BerealAlbum, "bereal-album", false, "Group BeReal assets into per-memory albums named 'BeReal/YYYY-MM-DD'")
}

// NewFromBeRealCommand creates the "from-bereal" subcommand for uploading BeReal memories
func NewFromBeRealCommand(ctx context.Context, parent *cobra.Command, app *app.Application, runner adapters.Runner) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "from-bereal [flags] <path>",
		Short: "Upload BeReal memories from an export folder",
		Long: `Upload BeReal memories from a BeReal export folder.
The folder should contain a 'memories.json' file and a 'Photos/bereal/' directory with the memory images.`,
		Args: cobra.ExactArgs(1),
	}
	cmd.SetContext(ctx)
	flags := cmd.Flags()
	o := ImportBeRealCmd{
		app: app,
	}
	o.RegisterFlags(flags, cmd)
	cmd.RunE = func(cmd *cobra.Command, args []string) error { //nolint:contextcheck
		return o.run(cmd.Context(), cmd, args, runner)
	}

	return cmd
}

func (ibc *ImportBeRealCmd) run(ctx context.Context, cmd *cobra.Command, args []string, runner adapters.Runner) error {
	sourcePath := args[0]

	// Open the source as a filesystem
	// For now, we only support local directories
	// The path could also be a BeReal export directory with structure:
	// <path>/memories.json
	// <path>/Photos/bereal/...

	// Try to find the memories.json at the top level or within a subfolder
	dirFS := os.DirFS(sourcePath)

	// Check if memories.json exists
	if _, err := fs.Stat(dirFS, "memories.json"); err != nil {
		// Try looking in subdirectories (in case user points to user ID folder)
		// For now, just report the error
		return err
	}

	// Create the adapter
	adapter, err := NewBeRealAdapter(dirFS, ibc.app.Log())
	if err != nil {
		return err
	}

	// Apply CLI options to adapter
	adapter.SetPerMemoryAlbum(ibc.BerealAlbum)

	// Report discovery results
	if err := adapter.Report(ctx); err != nil {
		return err
	}

	// Run the upload process
	return runner.Run(cmd, adapter)
}
