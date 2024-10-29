package upload

import (
	"context"
	"time"

	"github.com/simulot/immich-go/commands/application"
	"github.com/simulot/immich-go/internal/fileevent"
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
}

// NewUploadCommand adds the Upload command
func NewUploadCommand(ctx context.Context, app *application.Application) *cobra.Command {
	options := &UploadOptions{}
	cmd := &cobra.Command{
		Use:   "upload",
		Short: "Upload photos to an Immich server from various sources",
	}
	application.AddClientFlags(ctx, cmd, app)
	cmd.TraverseChildren = true
	cmd.PersistentFlags().BoolVar(&options.NoUI, "no-ui", false, "Disable the user interface")
	cmd.PersistentPreRunE = application.ChainRunEFunctions(cmd.PersistentPreRunE, options.Open, ctx, cmd, app)

	cmd.AddCommand(NewFromFolderCommand(ctx, app, options))
	cmd.AddCommand(NewFromGooglePhotosCommand(ctx, app, options))
	return cmd
}

func (options *UploadOptions) Open(ctx context.Context, cmd *cobra.Command, app *application.Application) error {
	// Initialize the Journal
	if app.Jnl() == nil {
		app.SetJnl(fileevent.NewRecorder(app.Log().Logger))
	}
	app.SetTZ(time.Local)
	if tz, err := cmd.Flags().GetString("time-zone"); err == nil {
		if loc, err := time.LoadLocation(tz); err == nil {
			app.SetTZ(loc)
		}
	}
	return nil
}
