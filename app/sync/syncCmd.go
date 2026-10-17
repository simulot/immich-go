package sync

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/internal/syncstate"
	"github.com/spf13/cobra"
)

// syncOptions holds shared flags for sync subcommands.
type syncOptions struct {
	Directory   string
	Delete      bool
	NoRecursive bool
	Force       bool

	app *app.Application
}

// NewSyncCommand creates the `sync` parent command with `down` and `up` subcommands.
func NewSyncCommand(ctx context.Context, a *app.Application) *cobra.Command {
	opts := &syncOptions{
		app: a,
	}

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Synchronize assets between local directory and Immich server",
		Long:  `Bidirectional sync: use 'sync down' to download from server, 'sync up' to upload to server.`,
	}

	flags := cmd.PersistentFlags()
	flags.StringVarP(&opts.Directory, "directory", "d", "", "Local directory for sync operations")
	flags.BoolVar(&opts.Delete, "delete", false, "Delete assets that no longer exist on the source side")
	flags.BoolVar(&opts.NoRecursive, "no-recursive", false, "Do not recurse into subdirectories")
	flags.BoolVar(&opts.Force, "force", false, "Force operation even on state mismatch or large deletions")

	cmd.AddCommand(
		newDownCommand(ctx, opts),
		newUpCommand(ctx, opts),
	)

	return cmd
}

// withGracefulShutdown wraps the context with signal handling (SIGINT, SIGTERM).
// On first signal: cancels the context so the sync loop stops gracefully.
// On second signal: exits immediately.
// Returns the wrapped context and a cleanup function that saves state and unlocks.
// The cleanup function MUST be called (typically via defer) regardless of how the sync ends.
func withGracefulShutdown(ctx context.Context, sm *syncstate.Manager, log *app.Log) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)

	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case <-sigCh:
			log.Message("Interrupt received, finishing current operation...")
			cancel()
			// Wait for second signal → force exit
			<-sigCh
			log.Message("Second interrupt, saving state and exiting...")
			_ = sm.Save()
			_ = sm.Unlock()
			os.Exit(1)
		case <-ctx.Done():
		}
		signal.Stop(sigCh)
	}()

	return ctx, func() {
		signal.Stop(sigCh)
		cancel()
	}
}
