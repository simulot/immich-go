package upload

import (
	"context"
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
)

func (m UpLoadMode) String() string {
	switch m {
	case UpModeGoogleTakeout:
		return "Google Takeout"
	case UpModeFolder:
		return "Folder"
	default:
		return "Unknown"
	}
}

// UploadOptions represents a set of common flags used for filtering assets.
type UploadOptions struct {
	// TODO place this option at the top
	NoUI bool // Disable UI

	Filters []filters.Filter
}

// NewUploadCommand adds the Upload command
func NewUploadCommand(ctx context.Context, a *app.Application) *cobra.Command {
	options := &UploadOptions{}
	cmd := &cobra.Command{
		Use:   "upload",
		Short: "Upload photos to an Immich server from various sources",
	}
	app.AddClientFlags(ctx, cmd, a, false)
	cmd.TraverseChildren = true
	cmd.PersistentFlags().BoolVar(&options.NoUI, "no-ui", false, "Disable the user interface")
	cmd.PersistentPreRunE = app.ChainRunEFunctions(cmd.PersistentPreRunE, options.Open, ctx, cmd, a)

	cmd.AddCommand(NewFromFolderCommand(ctx, cmd, a, options))
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
	return nil
}
