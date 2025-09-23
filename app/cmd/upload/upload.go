package upload

import (
	"context"
	"runtime"
	"time"

	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/filters"
	"github.com/spf13/cobra"
)

type UpLoadMode int

const (
	UpModeGoogleTakeout UpLoadMode = iota
	UpModeFolder
	UpModeICloud
	UpModePicasa
)

func (m UpLoadMode) String() string {
	switch m {
	case UpModeGoogleTakeout:
		return "Google Takeout"
	case UpModeFolder:
		return "Folder"
	case UpModeICloud:
		return "iCloud"
	case UpModePicasa:
		return "Picasa"
	default:
		return "Unknown"
	}
}

// UploadOptions represents a set of common flags used for filtering assets.
type UploadOptions struct {
	// TODO place this option at the top
	NoUI bool // Disable UI

	// Add Overwrite flag to UploadOptions
	Overwrite bool // Always overwrite files on the server with local versions

	// Concurrent upload configuration
	ConcurrentUploads int // Number of concurrent upload workers

	Filters []filters.Filter
}

// NewUploadCommand adds the Upload command
func NewUploadCommand(ctx context.Context, a *app.Application) *cobra.Command {
	options := &UploadOptions{}
	cmd := &cobra.Command{
		Use:   "upload",
		Short: "Upload photos to an Immich server from various sources",
		Args:  cobra.NoArgs, // This command does not accept any arguments, only subcommands

	}
	app.AddClientFlags(ctx, cmd, a, false)
	cmd.TraverseChildren = true
	cmd.PersistentFlags().BoolVar(&options.NoUI, "no-ui", false, "Disable the user interface")
	cmd.PersistentFlags().BoolVar(&options.Overwrite, "overwrite", false, "Always overwrite files on the server with local versions")
	cmd.PersistentFlags().IntVar(&options.ConcurrentUploads, "concurrent-uploads", runtime.NumCPU(), "Number of concurrent upload workers (1-20)")
	cmd.PersistentPreRunE = app.ChainRunEFunctions(cmd.PersistentPreRunE, options.Open, ctx, cmd, a)

	cmd.AddCommand(NewFromFolderCommand(ctx, cmd, a, options))
	cmd.AddCommand(NewFromICloudCommand(ctx, cmd, a, options))
	cmd.AddCommand(NewFromPicasaCommand(ctx, cmd, a, options))
	cmd.AddCommand(NewFromGooglePhotosCommand(ctx, cmd, a, options))
	cmd.AddCommand(NewFromImmichCommand(ctx, cmd, a, options))
	return cmd
}

func (options *UploadOptions) Open(ctx context.Context, cmd *cobra.Command, app *app.Application) error {
	// Initialize the Journal
	if app.Jnl() == nil {
		app.SetJnl(fileevent.NewRecorder(app.Log().Logger))
	}
	app.SetTZ(time.Local)
	if tz, err := cmd.Flags().GetString("time-zone"); err == nil && tz != "" {
		if loc, err := time.LoadLocation(tz); err == nil {
			app.SetTZ(loc)
		}
	}

	// Validate concurrent uploads range
	options.ConcurrentUploads = min(max(options.ConcurrentUploads, 1), 20)
	return nil
}
