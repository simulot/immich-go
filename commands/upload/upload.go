package upload

import (
	"context"
	"errors"

	"github.com/simulot/immich-go/commands/application"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/spf13/cobra"
)

// UploadOptions represents a set of common flags used for filtering assets.
type UploadOptions struct {
	// TODO place this option at the top
	NoUI bool // Disable UI
}

// NewUploadCommand adds the Upload command
func NewUploadCommand(ctx context.Context, app *application.Application) *cobra.Command {
	options := &UploadOptions{}
	cmd := &cobra.Command{
		Use:   "upload ...",
		Short: "Upload photos on an Immich server from various sources",
	}
	application.AddClientFlags(ctx, cmd, app)
	cmd.TraverseChildren = true
	cmd.PersistentFlags().BoolVar(&options.NoUI, "no-ui", false, "Disable the user interface")
	cmd.PersistentPreRunE = application.ChainRunEFunctions(cmd.PersistentPreRunE, options.Open, ctx, cmd, app)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return errors.New("missing from... sub-command")
	}
	cmd.AddCommand(NewFromFolderCommand(ctx, app, options))
	return cmd
}

func (options *UploadOptions) Open(ctx context.Context, cmd *cobra.Command, app *application.Application) error {
	// Initialize the Journal
	if app.Jnl() == nil {
		app.SetJnl(fileevent.NewRecorder(app.Log().Logger))
	}
	return nil
}
