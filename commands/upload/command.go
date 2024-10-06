package upload

import (
	"context"
	"errors"
	"fmt"

	"github.com/simulot/immich-go/adapters/folder"
	"github.com/simulot/immich-go/commands/application"
	cliflags "github.com/simulot/immich-go/internal/cliFlags"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/metadata"
	"github.com/simulot/immich-go/internal/namematcher"
	"github.com/spf13/cobra"
)

// UploadOption represents a set of common flags used for filtering assets.
type UploadOption struct {
	NoUI             bool // Disable UI
	StackJpgWithRaw  bool // Stack jpg/raw (Default: TRUE)
	StackBurstPhotos bool // Stack burst (Default: TRUE)

	Jnl *fileevent.Recorder // Log file events
}

// NewUploadCommand adds the Upload command
func NewUploadCommand(ctx context.Context, app *application.Application) *cobra.Command {
	options := &UploadOption{}
	cmd := &cobra.Command{
		Use:   "upload sub-command",
		Short: "Upload photos on an Immich server",
	}
	application.AddClientFlags(ctx, cmd, app)
	cmd.TraverseChildren = true
	cmd.PersistentFlags().BoolVar(&options.NoUI, "no-ui", false, "Disable the user interface")
	cmd.PersistentFlags().BoolVar(&options.StackJpgWithRaw, "stack-jpg-with-raw", false, "Stack JPG images with their corresponding raw images in Immich")
	cmd.PersistentFlags().BoolVar(&options.StackBurstPhotos, "stack-burst-photos", false, "Stack bursts of photos in Immich")
	cmd.PersistentPreRunE = application.ChainRunEFunctions(cmd.PersistentPreRunE, options.Open, ctx, cmd, app)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return errors.New("misisng sub-command")
	}
	cmd.AddCommand(NewUploadFolderCommand(ctx, app))
	return cmd
}

func (options *UploadOption) Open(ctx context.Context, cmd *cobra.Command, app *application.Application) error {
	// Initialize the Journal
	if options.Jnl == nil {
		options.Jnl = fileevent.NewRecorder(app.Log().Logger)
	}
	return nil
}

func NewUploadFolderCommand(ctx context.Context, app *application.Application) *cobra.Command {
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
		Use:   "from-folder [OPTIONS] <path> <path>...",
		Short: "Upload photos from a folder",
	}
	// cmd.Flags().SortFlags = false

	cmd.Flags().StringVar(&options.ImportIntoAlbum, "into-album", "", "Specify an album to import all files into")
	cmd.Flags().Var(&options.UsePathAsAlbumName, "folder-as-album", "Import all files in albums defined by the folder structure. Can be set to 'FOLDER' to use the folder name as the album name, or 'PATH' to use the full path as the album name")
	cmd.Flags().StringVar(&options.AlbumNamePathSeparator, "album-path-joiner", " / ", "Specify a string to use when joining multiple folder names to create an album name (e.g. ' ',' - ')")
	cmd.Flags().BoolVar(&options.Recursive, "recursive", true, "Explore the folder and all its sub-folders")
	cmd.Flags().Var(&options.BannedFiles, "ban-file", "Exclude a file based on a pattern (case-insensitive). Can be specified multiple times.")
	cmd.Flags().BoolVar(&options.IgnoreSideCarFiles, "ignore-sidecar-files", false, "Don't upload sidecar with the photo.")
	cliflags.AddInclusionFlags(cmd, &options.InclusionFlags)
	cliflags.AddDateHandlingFlags(cmd, &options.DateHandlingFlags)
	metadata.AddExifToolFlags(cmd, &options.ExifToolFlags)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		fmt.Println("Upload folder!", args)
		return nil
	}
	return cmd
}
