package folder

import (
	"context"
	"time"

	"github.com/simulot/immich-go/adapters"
	"github.com/simulot/immich-go/adapters/shared"
	"github.com/simulot/immich-go/app"
	cliflags "github.com/simulot/immich-go/internal/cliFlags"
	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/filetypes"
	"github.com/simulot/immich-go/internal/namematcher"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// ImportFolderOptions represents the flags used for importing assets from a file system.
type ImportFolderOptions struct {
	// CLI flags
	UsePathAsAlbumName     AlbumFolderMode
	AlbumNamePathSeparator string
	ImportIntoAlbum        string
	BannedFiles            namematcher.List
	Recursive              bool
	InclusionFlags         cliflags.InclusionFlags
	IgnoreSideCarFiles     bool
	FolderAsTags           bool
	TakeDateFromFilename   bool
	PicasaAlbum            bool
	ICloudTakeout          bool
	ICloudMemoriesAsAlbums bool

	Client         app.Client
	TZ             *time.Location
	SupportedMedia filetypes.SupportedMedia
	InfoCollector  *filenames.InfoCollector
}

func (o *ImportFolderOptions) RegisterFlags(flags *pflag.FlagSet, cmd *cobra.Command) {
	o.Recursive = true
	o.SupportedMedia = filetypes.DefaultSupportedMedia
	o.UsePathAsAlbumName = "none"
	o.BannedFiles, _ = namematcher.New(shared.DefaultBannedFiles...)

	flags.Var(&o.BannedFiles, "ban-file", "Exclude a file based on a pattern (case-insensitive). Can be specified multiple times.")
	flags.StringVar(&o.ImportIntoAlbum, "into-album", "", "Specify an album to import all files into")
	flags.Var(&o.UsePathAsAlbumName, "folder-as-album", "Import all files in albums defined by the folder structure. Can be set to 'FOLDER' to use the folder name as the album name, or 'PATH' to use the full path as the album name")
	flags.StringVar(&o.AlbumNamePathSeparator, "album-path-joiner", " / ", "Specify a string to use when joining multiple folder names to create an album name (e.g. ' ',' - ')")
	flags.BoolVar(&o.Recursive, "recursive", true, "Explore the folder and all its sub-folders")
	flags.BoolVar(&o.IgnoreSideCarFiles, "ignore-sidecar-files", false, "Don't upload sidecar with the photo.")
	flags.BoolVar(&o.FolderAsTags, "folder-as-tags", false, "Use the folder structure as tags, (ex: the file  holiday/summer 2024/file.jpg will have the tag holiday/summer 2024)")
	flags.BoolVar(&o.TakeDateFromFilename, "date-from-name", true, "Use the date from the filename if the date isn't available in the metadata (Only for jpg, mp4, heic, dng, cr2, cr3, arw, raf, nef, mov)")

	o.InclusionFlags.RegisterFlags(flags, "") // selection per extension
	o.ICloudTakeout = false
	o.PicasaAlbum = false
	switch cmd.Name() {
	case "from-picasa":
		flags.BoolVar(&o.PicasaAlbum, "album-picasa", true, "Use Picasa album name found in .picasa.ini file (default: false)")
	case "from-icloud":
		o.ICloudTakeout = true
		o.PicasaAlbum = false
		cmd.Flags().BoolVar(&o.ICloudMemoriesAsAlbums, "memories", false, "Import icloud memories as albums (default: false)")
	}
}

func NewFromFolderCommand(ctx context.Context, parent *cobra.Command, app *app.Application, runner adapters.Runner) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "from-folder [flags] <path>...",
		Short: "Upload photos from a folder",
		Args:  cobra.MinimumNArgs(1),
	}
	cmd.SetContext(ctx)
	flags := cmd.Flags()
	o := ImportFolderOptions{}
	o.RegisterFlags(flags, cmd)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return o.run(cmd, args, app, runner)
	}

	return cmd
}

func NewFromICloudCommand(ctx context.Context, parent *cobra.Command, app *app.Application, runner adapters.Runner) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "from-icloud [flags] <path>...",
		Short: "Upload photos from an iCloud takeout folder or zip file",
		Args:  cobra.MinimumNArgs(1),
	}
	cmd.SetContext(ctx)
	flags := cmd.Flags()
	o := ImportFolderOptions{}
	o.RegisterFlags(flags, cmd)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return o.run(cmd, args, app, runner)
	}
	return cmd
}

func NewFromPicasaCommand(ctx context.Context, parent *cobra.Command, app *app.Application, runner adapters.Runner) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "from-picasa [flags] <path>...",
		Short: "Upload photos from a Picasa folder or zip file",
		Args:  cobra.MinimumNArgs(1),
	}
	cmd.SetContext(ctx)
	flags := cmd.Flags()
	o := ImportFolderOptions{}
	o.RegisterFlags(flags, cmd)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return o.run(cmd, args, app, runner)
	}
	return cmd
}
