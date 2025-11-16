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
				return nil
			}
			for _, a := range g.Assets {
				err := ac.dest.WriteAsset(ctx, a)
				if err == nil {
					err = a.Close()
				}
				if err != nil {
					ac.app.FileProcessor().RecordAssetError(ctx, a.File, fileevent.ErrorFileAccess, err)
					errCount++
					if errCount > 5 {
						err := errors.New("too many errors, aborting")
						log.Error(err.Error())
						return err
					}
				} else {
					ac.app.FileProcessor().RecordAssetProcessed(ctx, a.File, fileevent.Written)
				}
			}
		}
	}
}
