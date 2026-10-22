package archive

import (
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
	// ready to run
	ctx := cmd.Context()
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
	ac.dest, err = folder.NewLocalAssetWriter(destFS, ".")
	if err != nil {
		return err
	}

	gChan := adapter.Browse(ctx)
	errCount := 0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case g, ok := <-gChan:
			if !ok {
				// The adapter closes its channel at the end, and also when the run is cancelled:
				// then the run did not finish.
				return ctx.Err()
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
					// asset, and it does not count toward the error limit. The run ends with the
					// context's error.
					return ctx.Err()
				default:
					ac.app.FileProcessor().RecordAssetError(ctx, a.File, int64(a.FileSize), fileevent.ErrorFileAccess, err)
					errCount++
					if errCount > 5 {
						err := errors.New("too many errors, aborting")
						log.Error(err.Error())
						return err
					}
				}
			}
		}
	}
}
