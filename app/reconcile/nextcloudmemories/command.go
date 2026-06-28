package nextcloudmemories

import (
	"context"
	"strings"

	"github.com/simulot/immich-go/app"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type commandRunner struct {
	app                  *app.Application
	client               app.Client
	cleanupMigrationTags bool
	run                  func(context.Context, *app.Application, *app.Client, bool) error
}

func (cr *commandRunner) registerFlags(flags *pflag.FlagSet) {
	cr.client.RegisterFlags(flags, "")
	flags.BoolVar(&cr.cleanupMigrationTags, "cleanup-migration-tags", false, "Remove successfully reconciled synthetic migration tags after the run")
}

func (cr *commandRunner) runCommand(ctx context.Context) error {
	return cr.run(ctx, cr.app, &cr.client, cr.cleanupMigrationTags)
}

// NewCommand creates the reconcile subcommand for Nextcloud Memories.
func NewCommand(ctx context.Context, app *app.Application) *cobra.Command {
	return newCommand(ctx, app, runNextcloudMemoriesReconcile)
}

func newCommand(ctx context.Context, app *app.Application, run func(context.Context, *app.Application, *app.Client, bool) error) *cobra.Command {
	runner := &commandRunner{app: app, run: run}

	cmd := &cobra.Command{
		Use:   "nextcloud-memories [flags]",
		Short: "Reconcile Nextcloud Memories shared-album migration state in Immich",
		Long: strings.TrimSpace(`Reconcile destination-side Nextcloud Memories migration state already stored in Immich.

This command is intended for post-import convergence workflows such as shared-album
reconstruction after one or more users have already imported their own libraries.`),
		Args: cobra.NoArgs,
	}
	cmd.SetContext(ctx)
	runner.registerFlags(cmd.Flags())

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runner.runCommand(ctx)
	}

	return cmd
}
