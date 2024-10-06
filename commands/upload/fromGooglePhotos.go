package upload

import (
	"context"
	"errors"
	"strings"

	gp "github.com/simulot/immich-go/adapters/googlePhotos"
	"github.com/simulot/immich-go/commands/application"
	cliflags "github.com/simulot/immich-go/internal/cliFlags"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/simulot/immich-go/internal/metadata"
	"github.com/spf13/cobra"
)

func NewFromGooglePhotosCommand(ctx context.Context, app *application.Application, upOptions *UploadOptions) *cobra.Command {
	options := &gp.ImportFlags{}

	cmd := &cobra.Command{
		Use:   "from-google-photos [flags] <takeout-*.zip> | <takeout-folder>",
		Short: "Upload photos either from a zipped Google Photos takeout or decompressed archive",
		Args:  cobra.MinimumNArgs(1),
	}
	cmd.SetContext(ctx)

	// AddGoogleTakeoutFlags adds the command-line flags for the Google Photos takeout import command
	cmd.Flags().BoolVarP(&options.UseJSONMetadata, "use-json-metadata", "j", true, "Use JSON metadata for date and GPS information")
	cmd.Flags().BoolVar(&options.CreateAlbums, "sync-albums", true, "Automatically create albums in Immich that match the albums in your Google Photos takeout")
	cmd.Flags().StringVar(&options.ImportFromAlbum, "import-from-album-name", "", "Only import photos from the specified Google Photos album")
	cmd.Flags().BoolVar(&options.KeepUntitled, "include-untitled-albums", false, "Include photos from albums without a title in the import process")
	cmd.Flags().BoolVarP(&options.KeepTrashed, "include-trashed", "t", false, "Import photos that are marked as trashed in Google Photos")
	cmd.Flags().BoolVarP(&options.KeepPartner, "include-partner", "p", true, "Import photos from your partner's Google Photos account")
	cmd.Flags().StringVar(&options.PartnerSharedAlbum, "partner-shared-album", "", "Add partner's photo to the specified album name")
	cmd.Flags().BoolVarP(&options.KeepArchived, "include-archived", "a", true, "Import archived Google Photos")
	cmd.Flags().BoolVarP(&options.KeepJSONLess, "include-unmatched", "u", false, "Import photos that do not have a matching JSON file in the takeout")
	cmd.Flags().Var(&options.BannedFiles, "ban-file", "Exclude a file based on a pattern (case-insensitive). Can be specified multiple times.")

	// TODO
	// cmd.Flags().BoolVar(&options.StackJpgWithRaw, "stack-jpg-with-raw", false, "Stack JPG images with their corresponding raw images in Immich")
	// cmd.Flags().BoolVar(&options.StackBurstPhotos, "stack-burst-photos", false, "Stack bursts of photos in Immich")

	cliflags.AddInclusionFlags(cmd, &options.InclusionFlags)
	cliflags.AddDateHandlingFlags(cmd, &options.DateHandlingFlags)
	metadata.AddExifToolFlags(cmd, &options.ExifToolFlags)
	options.SupportedMedia = metadata.DefaultSupportedMedia

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		log := app.Log()
		client := app.Client()

		fsyss, err := fshelper.ParsePath(args)
		if err != nil {
			return err
		}
		if len(fsyss) == 0 {
			log.Message("No file found matching the pattern: %s", strings.Join(args, ","))
			return errors.New("No file found matching the pattern: " + strings.Join(args, ","))
		}

		options.SupportedMedia = client.Immich.SupportedMedia()
		adapter, err := gp.NewTakeout(ctx, app.Jnl(), options, fsyss...)
		if err != nil {
			return err
		}
		return newUpload(app, upOptions).run(ctx, adapter, app)
	}

	return cmd
}
