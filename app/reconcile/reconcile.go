package reconcile

import (
	"context"
	"errors"

	"github.com/simulot/immich-go/app"
	reconcilencm "github.com/simulot/immich-go/app/reconcile/nextcloudmemories"
	"github.com/spf13/cobra"
)

// NewCommand creates the top-level reconcile command.
func NewCommand(ctx context.Context, a *app.Application) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reconcile",
		Short: "Reconcile post-import migration state in Immich",
		Long: `Run post-import reconciliation workflows that converge destination-side
migration state already stored in Immich.`,
		Args: cobra.NoArgs,
	}

	cmd.AddCommand(reconcilencm.NewCommand(ctx, a))

	cmd.RunE = func(cmd *cobra.Command, args []string) error { //nolint:contextcheck
		return errors.New("you must specify a subcommand to the reconcile command")
	}

	return cmd
}
