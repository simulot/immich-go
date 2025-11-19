package archive

import (
	"context"
	"errors"

	"github.com/simulot/immich-go/adapters/folder"
	"github.com/simulot/immich-go/adapters/fromimmich"
	gp "github.com/simulot/immich-go/adapters/googlePhotos"
	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/internal/assettracker"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/fileprocessor"
	"github.com/spf13/cobra"
)

type ArchiveCmd struct {
	ArchivePath string

	app  *app.Application
	dest *folder.LocalAssetWriter
}

func NewArchiveCommand(ctx context.Context, app *app.Application) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "archive",
		Short: "Archive various sources of photos to a file system",
	}
	ac := &ArchiveCmd{
		app: app,
	}

	cmd.PersistentFlags().StringVarP(&ac.ArchivePath, "write-to-folder", "w", "", "Path where to write the archive")
	_ = cmd.MarkPersistentFlagRequired("write-to-folder")

	cmd.AddCommand(folder.NewFromFolderCommand(ctx, cmd, app, ac))
	cmd.AddCommand(folder.NewFromICloudCommand(ctx, cmd, app, ac))
	cmd.AddCommand(folder.NewFromPicasaCommand(ctx, cmd, app, ac))
	cmd.AddCommand(fromimmich.NewFromImmichCommand(ctx, cmd, app, ac))
	cmd.AddCommand(gp.NewFromGooglePhotosCommand(ctx, cmd, app, ac))

	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// Initialize the FileProcessor (tracker + logger)
		if app.FileProcessor() == nil {
			logger := fileevent.NewRecorder(app.Log().Logger)
			tracker := assettracker.NewWithLogger(app.Log().Logger, app.DryRun) // Enable debug mode in dry-run
			processor := fileprocessor.New(tracker, logger)
			app.SetFileProcessor(processor)
		}

		// app.SetTZ(time.Local)
		// if tz, err := cmd.Flags().GetString("time-zone"); err == nil && tz != "" {
		// 	if loc, err := time.LoadLocation(tz); err == nil {
		// 		app.SetTZ(loc)
		// 	}
		// } else {
		// 	return err
		// }
		return nil
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error { //nolint:contextcheck
		return errors.New("you must specify a subcommand to the archive command")
	}
	return cmd
}
