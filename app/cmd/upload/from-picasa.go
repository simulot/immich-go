package upload

import (
	"context"
	"errors"
	"strings"

	"github.com/simulot/immich-go/adapters/folder"
	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/internal/filetypes"
	"github.com/simulot/immich-go/internal/filters"
	"github.com/simulot/immich-go/internal/namematcher"

	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/spf13/cobra"
)

func NewFromPicasaCommand(ctx context.Context, parent *cobra.Command, app *app.Application, upOptions *UploadOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "from-picasa [flags] <path>...",
		Short: "Upload photos from a Picasa folder or zip file",
		Args:  cobra.MinimumNArgs(1),
	}
	cmd.SetContext(ctx)
	options := &folder.ImportFolderOptions{}
	app.Config.Register(cmd, options, &options.UploadFlags, &options.InclusionFlags)

	options.ManageHEICJPG = filters.HeicJpgNothing
	options.ManageRawJPG = filters.RawJPGNothing
	options.ManageBurst = filters.BurstNothing
	options.Recursive = true
	options.SupportedMedia = filetypes.DefaultSupportedMedia
	options.UsePathAsAlbumName = folder.FolderModeNone
	options.BannedFiles, _ = namematcher.New(
		`@eaDir/`,
		`@__thumb/`,          // QNAP
		`SYNOFILE_THUMB_*.*`, // SYNOLOGY
		`Lightroom Catalog/`, // LR
		`thumbnails/`,        // Android photo
		`.DS_Store/`,         // Mac OS custom attributes
		`/._*`,               // MacOS resource files
		`.photostructure/`,   // PhotoStructure
	)

	options.ICloudTakeout = false
	options.PicasaAlbum = true

	cmd.RunE = func(cmd *cobra.Command, args []string) error { //nolint:contextcheck
		// ready to run
		ctx := cmd.Context()
		log := app.Log()
		client := app.Client()
		options.TZ = app.GetTZ()
		options.InclusionFlags.SetIncludeTypeExtensions()

		// parse arguments
		fsyss, err := fshelper.ParsePath(args)
		if err != nil {
			return err
		}
		if len(fsyss) == 0 {
			log.Message("No file found matching the pattern: %s", strings.Join(args, ","))
			return errors.New("No file found matching the pattern: " + strings.Join(args, ","))
		}

		// create the adapter for folders
		options.SupportedMedia = client.Immich.SupportedMedia()
		upOptions.Filters = append(upOptions.Filters, options.ManageBurst.GroupFilter(), options.ManageRawJPG.GroupFilter(), options.ManageHEICJPG.GroupFilter())

		options.InfoCollector = filenames.NewInfoCollector(app.GetTZ(), options.SupportedMedia)
		adapter, err := folder.NewLocalFiles(ctx, app.Jnl(), options, fsyss...)
		if err != nil {
			return err
		}

		return newUpload(UpModeFolder, app, upOptions).run(ctx, adapter, app, fsyss)
	}

	return cmd
}
