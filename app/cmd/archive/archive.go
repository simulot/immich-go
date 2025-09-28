package archive

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/simulot/immich-go/adapters/folder"
	"github.com/simulot/immich-go/adapters/fromimmich"
	gp "github.com/simulot/immich-go/adapters/googlePhotos"
	"github.com/simulot/immich-go/app"
	cliflags "github.com/simulot/immich-go/internal/cliFlags"
	"github.com/simulot/immich-go/internal/configuration"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/simulot/immich-go/internal/fshelper/osfs"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type ArchiveOptions struct {
	ArchivePath string
	DateRange   cliflags.DateRange // Set date range for archiving
	DryRun      bool               // Enable dry-run mode
}

func NewArchiveCommand(ctx context.Context, a *app.Application) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "archive",
		Short: "Archive various sources of photos to a file system",
	}
	options := &ArchiveOptions{}

	app.AddClientFlags(ctx, cmd, a, false)
	cmd.TraverseChildren = true

	// Add archive-specific flags
	cmd.PersistentFlags().StringVarP(&options.ArchivePath, "write-to-folder", "w", "", "Path where to write the archive")
	cmd.PersistentFlags().Var(&options.DateRange, "date-range", "Archive photos taken in the date range")
	_ = cmd.MarkPersistentFlagRequired("write-to-folder")

	// Bind archive flags to Viper
	_ = viper.BindPFlag("archive.date_range", cmd.PersistentFlags().Lookup("date-range"))
	// Note: dry-run is handled by AddClientFlags and bound to upload.dry_run

	cmd.PersistentPreRunE = app.ChainRunEFunctions(cmd.PersistentPreRunE, options.LoadConfiguration, ctx, cmd, a)

	cmd.AddCommand(NewImportFromFolderCommand(ctx, cmd, a, options))
	cmd.AddCommand(NewFromGooglePhotosCommand(ctx, cmd, a, options))
	cmd.AddCommand(NewFromImmichCommand(ctx, cmd, a, options))

	cmd.RunE = func(cmd *cobra.Command, args []string) error { //nolint:contextcheck
		return errors.New("you must specify a subcommand to the archive command")
	}
	return cmd
}

func (options *ArchiveOptions) LoadConfiguration(ctx context.Context, cmd *cobra.Command, app *app.Application) error {
	// Load configuration values from Viper into archive options
	config, err := configuration.GetConfiguration()
	if err != nil {
		return err
	}

	var archiveConfigSources []string

	// Apply configuration values (only if not set by flags)
	if !cmd.PersistentFlags().Changed("date-range") && config.Archive.DateRange != "" {
		_ = options.DateRange.Set(config.Archive.DateRange)
		archiveConfigSources = append(archiveConfigSources, fmt.Sprintf("date-range=%s from config", config.Archive.DateRange))
	} else if cmd.PersistentFlags().Changed("date-range") {
		archiveConfigSources = append(archiveConfigSources, "date-range from CLI flag")
	}

	if !cmd.PersistentFlags().Changed("dry-run") {
		options.DryRun = config.Upload.DryRun
		if config.Upload.DryRun {
			archiveConfigSources = append(archiveConfigSources, "dry-run=true from config")
		}
	} else {
		archiveConfigSources = append(archiveConfigSources, "dry-run from CLI flag")
	}

	// Log archive-specific configuration
	if len(archiveConfigSources) > 0 {
		app.Log().Message("Archive configuration sources:")
		for _, source := range archiveConfigSources {
			app.Log().Message("  - %s", source)
		}
	}

	return nil
}

func NewImportFromFolderCommand(ctx context.Context, parent *cobra.Command, app *app.Application, archOptions *ArchiveOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "from-folder",
		Short: "Archive photos from a folder",
	}

	options := &folder.ImportFolderOptions{}
	options.AddFromFolderFlags(cmd, parent)

	cmd.RunE = func(cmd *cobra.Command, args []string) error { //nolint:contextcheck
		// ready to run
		ctx := cmd.Context()
		log := app.Log()
		if app.Jnl() == nil {
			app.SetJnl(fileevent.NewRecorder(app.Log().Logger))
			app.Jnl().SetLogger(app.Log().SetLogWriter(os.Stdout))
		}

		options.TZ = app.GetTZ()

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
		options.InfoCollector = filenames.NewInfoCollector(options.TZ, options.SupportedMedia)
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

func NewFromGooglePhotosCommand(ctx context.Context, parent *cobra.Command, app *app.Application, archOptions *ArchiveOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "from-google-photos [flags] <takeout-*.zip> | <takeout-folder>",
		Short: "Archive photos either from a zipped Google Photos takeout or decompressed archive",
		Args:  cobra.MinimumNArgs(1),
	}
	cmd.SetContext(ctx)
	options := &gp.ImportFlags{}
	options.AddFromGooglePhotosFlags(cmd, parent)

	cmd.RunE = func(cmd *cobra.Command, args []string) error { //nolint:contextcheck
		ctx := cmd.Context()
		log := app.Log()
		if app.Jnl() == nil {
			app.SetJnl(fileevent.NewRecorder(app.Log().Logger))
			app.Jnl().SetLogger(app.Log().SetLogWriter(os.Stdout))
		}
		options.TZ = app.GetTZ()
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

func NewFromImmichCommand(ctx context.Context, parent *cobra.Command, app *app.Application, archOptions *ArchiveOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "from-immich [from-flags]",
		Short: "Archive photos from Immich",
	}
	cmd.SetContext(ctx)
	options := &fromimmich.FromImmichFlags{}
	options.AddFromImmichFlags(cmd, parent)

	cmd.RunE = func(cmd *cobra.Command, args []string) error { //nolint:contextcheck
		ctx := cmd.Context()
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

		dest, err := folder.NewLocalAssetWriter(destFS, ".")
		if err != nil {
			return err
		}

		source, err := fromimmich.NewFromImmich(ctx, app, app.Jnl(), options)
		if err != nil {
			return err
		}
		return run(ctx, app.Jnl(), app, source, dest)
	}

	return cmd
}
