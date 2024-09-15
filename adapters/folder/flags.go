package folder

import (
	"fmt"
	"strings"

	cliflags "github.com/simulot/immich-go/internal/cliFlags"
	"github.com/simulot/immich-go/internal/metadata"
	"github.com/simulot/immich-go/internal/namematcher"
	"github.com/spf13/cobra"
)

// ImportFlags represents the flags used for importing assets from a file system.
type ImportFlags struct {
	// UsePathAsAlbumName determines whether to create albums based on the full path to the asset.
	UsePathAsAlbumName AlbumFolderMode

	// AlbumNamePathSeparator specifies how multiple (sub) folders are joined when creating album names.
	AlbumNamePathSeparator string

	// ImportIntoAlbum is the name of the album where all assets will be added.
	ImportIntoAlbum string

	// BannedFiles is a list of file name patterns to be excluded from the import process.
	BannedFiles namematcher.List

	// Recursive indicates whether to explore the folder and all its sub-folders.
	Recursive bool

	// InclusionFlags controls the file extensions to be included in the import process.
	InclusionFlags cliflags.InclusionFlags

	// DateHandlingFlags provides options for handling the capture date of the assets.
	DateHandlingFlags cliflags.DateHandlingFlags

	// ExifToolFlags specifies options for the exif.
	ExifToolFlags metadata.ExifToolFlags

	// IgnoreSideCarFiles indicates whether to ignore XMP files during the import process.
	IgnoreSideCarFiles bool

	// SupportedMedia is the server's actual list of supported media types.
	SupportedMedia metadata.SupportedMedia
}

func AddUploadFolderFlags(cmd *cobra.Command, flags *ImportFlags) {
	flags.BannedFiles, _ = namematcher.New(
		`@eaDir/`,
		`@__thumb/`,          // QNAP
		`SYNOFILE_THUMB_*.*`, // SYNOLOGY
		`Lightroom Catalog/`, // LR
		`thumbnails/`,        // Android photo
		`.DS_Store/`,         // Mac OS custom attributes
	)
	flags.UsePathAsAlbumName = FolderModeNone

	cmd.Flags().StringVar(&flags.ImportIntoAlbum, "into-album", "", "Specify an album to import all files into")
	cmd.Flags().Var(&flags.UsePathAsAlbumName, "folder-as-album", "Import all files in albums defined by the folder structure. Can be set to 'FOLDER' to use the folder name as the album name, or 'PATH' to use the full path as the album name")
	cmd.Flags().StringVar(&flags.AlbumNamePathSeparator, "album-path-joiner", " / ", "Specify a string to use when joining multiple folder names to create an album name (e.g. ' ',' - ')")
	cmd.Flags().BoolVar(&flags.Recursive, "recursive", true, "Explore the folder and all its sub-folders")
	cmd.Flags().Var(&flags.BannedFiles, "ban-file", "Exclude a file based on a pattern (case-insensitive). Can be specified multiple times.")
	cmd.Flags().BoolVar(&flags.IgnoreSideCarFiles, "ignore-sidecar-files", false, "Don't upload sidecar with the photo.")
	cliflags.AddInclusionFlags(cmd, &flags.InclusionFlags)
	cliflags.AddDateHandlingFlags(cmd, &flags.DateHandlingFlags)
	metadata.AddExifToolFlags(cmd, &flags.ExifToolFlags)
	flags.SupportedMedia = metadata.DefaultSupportedMedia
}

// AlbumFolderMode represents the mode in which album folders are organized.
// Implement the interface pflag.Value

type AlbumFolderMode string

const (
	FolderModeNone   AlbumFolderMode = "NONE"
	FolderModeFolder AlbumFolderMode = "FOLDER"
	FolderModePath   AlbumFolderMode = "PATH"
)

func (m AlbumFolderMode) String() string {
	return string(m)
}

func (m *AlbumFolderMode) Set(v string) error {
	v = strings.TrimSpace(strings.ToUpper(v))
	switch v {
	case string(FolderModeFolder), string(FolderModePath):
		*m = AlbumFolderMode(v)
	default:
		return fmt.Errorf("invalid value for folder mode, expected %s, %s or %s", FolderModeFolder, FolderModePath, FolderModeNone)
	}
	return nil
}

func (m AlbumFolderMode) Type() string {
	return "folderMode"
}
