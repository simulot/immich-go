package upload

import (
	"context"
	"errors"
	"strings"

	"github.com/simulot/immich-go/adapters/folder"
	"github.com/simulot/immich-go/commands/application"
	cliflags "github.com/simulot/immich-go/internal/cliFlags"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/simulot/immich-go/internal/metadata"
	"github.com/simulot/immich-go/internal/namematcher"
	"github.com/spf13/cobra"
)

func NewFromFolderCommand(ctx context.Context, app *application.Application, upOptions *UploadOptions) *cobra.Command {
	options := &folder.ImportFolderOptions{}
	options.BannedFiles, _ = namematcher.New(
		`@eaDir/`,
		`@__thumb/`,          // QNAP
		`SYNOFILE_THUMB_*.*`, // SYNOLOGY
		`Lightroom Catalog/`, // LR
		`thumbnails/`,        // Android photo
		`.DS_Store/`,         // Mac OS custom attributes
	)
	options.SupportedMedia = metadata.DefaultSupportedMedia
	options.UsePathAsAlbumName = folder.FolderModeNone

	cmd := &cobra.Command{
		Use:   "from-folder [flags] <path>...",
		Short: "Upload photos from a folder",
	}
	cmd.SetContext(ctx)

	cmd.Flags().StringVar(&options.ImportIntoAlbum, "into-album", "", "Specify an album to import all files into")
	cmd.Flags().Var(&options.UsePathAsAlbumName, "folder-as-album", "Import all files in albums defined by the folder structure. Can be set to 'FOLDER' to use the folder name as the album name, or 'PATH' to use the full path as the album name")
	cmd.Flags().StringVar(&options.AlbumNamePathSeparator, "album-path-joiner", " / ", "Specify a string to use when joining multiple folder names to create an album name (e.g. ' ',' - ')")
	cmd.Flags().BoolVar(&options.Recursive, "recursive", true, "Explore the folder and all its sub-folders")
	cmd.Flags().Var(&options.BannedFiles, "ban-file", "Exclude a file based on a pattern (case-insensitive). Can be specified multiple times.")
	cmd.Flags().BoolVar(&options.IgnoreSideCarFiles, "ignore-sidecar-files", false, "Don't upload sidecar with the photo.")
	// cmd.Flags().BoolVar(&options.StackJpgWithRaw, "stack-jpg-with-raw", false, "Stack JPG images with their corresponding raw images in Immich")
	// cmd.Flags().BoolVar(&options.StackBurstPhotos, "stack-burst-photos", false, "Stack bursts of photos in Immich")

	cliflags.AddInclusionFlags(cmd, &options.InclusionFlags)
	cliflags.AddDateHandlingFlags(cmd, &options.DateHandlingFlags)
	metadata.AddExifToolFlags(cmd, &options.ExifToolFlags)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// ready to run
		ctx := cmd.Context()
		log := app.Log()
		client := app.Client()

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
		adapter, err := folder.NewLocalFiles(ctx, app.Jnl(), options, fsyss...)
		if err != nil {
			return err
		}

		return newUpload(app, upOptions).run(ctx, adapter, app)
	}

	return cmd
}
