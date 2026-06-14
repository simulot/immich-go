package nextcloudmemories

import (
	"context"
	"errors"
	"strings"

	"github.com/simulot/immich-go/app"
	"github.com/spf13/cobra"
)

// NewCommand creates the reconcile subcommand scaffold for Nextcloud Memories.
func NewCommand(ctx context.Context, app *app.Application) *cobra.Command {
	_ = app

	cmd := &cobra.Command{
		Use:   "nextcloud-memories [flags]",
		Short: "Reconcile Nextcloud Memories shared-album migration state in Immich",
		Long: strings.TrimSpace(`Reconcile destination-side Nextcloud Memories migration state already stored in Immich.

This command is intended for post-import convergence workflows such as shared-album
reconstruction after one or more users have already imported their own libraries.`),
		Args: cobra.NoArgs,
	}
	cmd.SetContext(ctx)

	cmd.Flags().Bool("cleanup-migration-tags", false, "Remove successfully reconciled synthetic migration tags after the run")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return errors.New("nextcloud Memories reconciliation is not implemented yet")
	}

	return cmd
}
