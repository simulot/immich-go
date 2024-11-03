package archive

import (
	"context"
	"errors"
	"os"
	"strings"

	gp "github.com/simulot/immich-go/adapters/googlePhotos"

	"github.com/simulot/immich-go/adapters/folder"
	"github.com/simulot/immich-go/commands/application"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/simulot/immich-go/internal/fshelper/osfs"
	"github.com/spf13/cobra"
)

type ArchiveOptions struct {
	ArchivePath string
}

func NewArchiveCommand(ctx context.Context, app *application.Application) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "archive",
		Short: "Archive various sources of photos to a file system",
	}
	options := &ArchiveOptions{}

	cmd.PersistentFlags().StringVarP(&options.ArchivePath, "write-to-folder", "w", "", "Path where to write the archive")
	_ = cmd.MarkPersistentFlagRequired("write-to-folder")

	cmd.AddCommand(NewImportFromFolderCommand(ctx, app, options))
	cmd.AddCommand(NewFromGooglePhotosCommand(ctx, app, options))

	return cmd
}

func NewImportFromFolderCommand(ctx context.Context, app *application.Application, archOptions *ArchiveOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "from-folder",
		Short: "Archive photos from a folder",
	}

	options := &folder.ImportFolderOptions{}
	options.AddFromFolderFlags(cmd)

	cmd.RunE = func(cmd *cobra.Command, args []string) error { //nolint:contextcheck
		// ready to run
		ctx := cmd.Context()
		log := app.Log()
		if app.Jnl() == nil {
			app.SetJnl(fileevent.NewRecorder(app.Log().Logger))
			app.Jnl().SetLogger(app.Log().SetLogWriter(os.Stdout))
		}
		p, err := cmd.Flags().GetString("write-to-folder")
		if err != nil {
			return err
		}

		err = os.MkdirAll(p, 0o755)
		if err != nil {
			return err
		}

		destFS := osfs.DirFS(p)

		// parse arguments
		fsyss, err := fshelper.ParsePath(args)
		if err != nil {
			return err
		}
		if len(fsyss) == 0 {
			log.Message("No file found matching the pattern: %s", strings.Join(args, ","))
			return errors.New("No file found matching the pattern: " + strings.Join(args, ","))
		}
		options.InfoCollector = filenames.NewInfoCollector(app.GetTZ(), options.SupportedMedia)
		source, err := folder.NewLocalFiles(ctx, app.Jnl(), options, fsyss...)
		if err != nil {
			return err
		}

		dest, err := folder.NewLocalAssetWriter(destFS, ".")
		if err != nil {
			return err
		}
		return run(ctx, app.Jnl(), app, source, dest)
	}
	return cmd
}

func NewFromGooglePhotosCommand(ctx context.Context, app *application.Application, archOptions *ArchiveOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "from-google-photos [flags] <takeout-*.zip> | <takeout-folder>",
		Short: "Archive photos either from a zipped Google Photos takeout or decompressed archive",
		Args:  cobra.MinimumNArgs(1),
	}
	cmd.SetContext(ctx)
	options := &gp.ImportFlags{}
	options.AddFromGooglePhotosFlags(cmd)

	cmd.RunE = func(cmd *cobra.Command, args []string) error { //nolint:contextcheck
		ctx := cmd.Context()
		log := app.Log()
		if app.Jnl() == nil {
			app.SetJnl(fileevent.NewRecorder(app.Log().Logger))
			app.Jnl().SetLogger(app.Log().SetLogWriter(os.Stdout))
		}
		p, err := cmd.Flags().GetString("write-to-folder")
		if err != nil {
			return err
		}

		err = os.MkdirAll(p, 0o755)
		if err != nil {
			return err
		}

		destFS := osfs.DirFS(p)

		fsyss, err := fshelper.ParsePath(args)
		if err != nil {
			return err
		}
		if len(fsyss) == 0 {
			log.Message("No file found matching the pattern: %s", strings.Join(args, ","))
			return errors.New("No file found matching the pattern: " + strings.Join(args, ","))
		}
		source, err := gp.NewTakeout(ctx, app.Jnl(), options, fsyss...)
		if err != nil {
			return err
		}
		dest, err := folder.NewLocalAssetWriter(destFS, ".")
		if err != nil {
			return err
		}
		return run(ctx, app.Jnl(), app, source, dest)
	}

	return cmd
}
