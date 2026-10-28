package archive

import (
	"context"
	"errors"
	"os"

	"github.com/simulot/immich-go/adapters"
	"github.com/simulot/immich-go/adapters/folder"
	"github.com/simulot/immich-go/internal/assettracker"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/fileprocessor"
	"github.com/simulot/immich-go/internal/fshelper/osfs"
	"github.com/spf13/cobra"
)

func (ac *ArchiveCmd) Run(cmd *cobra.Command, adapter adapters.Reader) error {
	// The run's context. An error that stops the run cancels it, which stops the adapter too, and
	// the run ends with that error, its cause, as upload does.
	ctx, cancel := context.WithCancelCause(cmd.Context())
	defer cancel(nil)
	log := ac.app.Log()
	log.Info("in ArchiveCmd.Run", "archivePath", ac.ArchivePath)

	// Initialize the Journal and FileProcessor
	if ac.app.FileProcessor() == nil {
		recorder := fileevent.NewRecorder(ac.app.Log().Logger)
		tracker := assettracker.NewWithLogger(ac.app.Log().Logger, ac.app.DryRun)
		processor := fileprocessor.New(tracker, recorder)
		ac.app.SetFileProcessor(processor)
	}

	p := ac.ArchivePath
	err := os.MkdirAll(p, 0o755)
	if err != nil {
		return err
	}

	destFS := osfs.DirFS(p)
	ac.dest, err = folder.NewLocalAssetWriter(destFS)
	if err != nil {
		return err
	}
	ac.dest.SetLogger(log.Logger)

	gChan := adapter.Browse(ctx)
	for {
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case g, ok := <-gChan:
			if !ok {
				// The adapter closes its channel at the end, and also when the run is cancelled
				// or when it fails.
				if ctx.Err() != nil {
					return context.Cause(ctx)
				}
				return ac.runErrors()
			}
			for _, a := range g.Assets {
				// Closed even when it is refused: closing removes the cache reader's copy of the download.
				err := errors.Join(ac.dest.WriteAsset(ctx, a), a.Close())
				switch {
				case err == nil:
					// Asset successfully archived
					ac.app.FileProcessor().RecordAssetProcessed(ctx, a.File, int64(a.FileSize), fileevent.ProcessedFileArchived)
				case ctx.Err() != nil && errors.Is(err, ctx.Err()):
					// The run is cancelled, and the asset with it: that is not an error of the
					// asset, and it does not count against --on-errors. The run ends with the
					// cancellation.
					return context.Cause(ctx)
				default:
					ac.app.FileProcessor().RecordAssetError(ctx, a.File, int64(a.FileSize), fileevent.ErrorFileAccess, err)
					// --on-errors tells whether the run goes on. When it stops, the run ends with
					// this error, or with the cancellation when the run was cancelled first (a
					// download cut short by it): a cancelled context keeps its first cause.
					if err := ac.app.ProcessError(err); err != nil {
						cancel(err)
						return context.Cause(ctx)
					}
				}
			}
		}
	}
}

// runErrors returns an error when the run had one, even when --on-errors let it go on: an asset
// not archived, or an error the adapter counted against --on-errors or recorded as an event.
func (ac *ArchiveCmd) runErrors() error {
	counts := ac.app.FileProcessor().Logger().GetCounts()
	if counts[fileevent.ErrorUploadFailed]+counts[fileevent.ErrorServerError]+counts[fileevent.ErrorFileAccess]+counts[fileevent.ErrorIncomplete] > 0 ||
		ac.app.NumErrors() > 0 {
		return errors.New("some errors have occurred, look at the log for details")
	}
	return nil
}
