package manage

import (
	"context"

	"github.com/simulot/immich-go/app"
	"github.com/spf13/cobra"
)

// NewManageCommand creates the parent cobra command for manage subcommands.
func NewManageCommand(ctx context.Context, a *app.Application) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "manage",
		Short: "Manage Immich resources",
		Long:  "Parent command for managing Immich resources. Use subcommands for specific operations.",
	}

	cmd.AddCommand(
		NewPeopleAlbumSyncCommand(ctx, a),
	)

	return cmd
}
