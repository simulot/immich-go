package folder

import (
	"fmt"
	"strings"

	cliflags "github.com/simulot/immich-go/internal/cliFlags"
	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/filters"
	"github.com/simulot/immich-go/internal/metadata"
	"github.com/simulot/immich-go/internal/namematcher"
)

// ImportFolderOptions represents the flags used for importing assets from a file system.
type ImportFolderOptions struct {
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

	// Stack jpg/raw
	StackJpgWithRaw bool

	// Stack burst
	StackBurstPhotos bool

	// SupportedMedia is the server's actual list of supported media types.
	SupportedMedia metadata.SupportedMedia

	// InfoCollector is used to extract information from the file name.
	InfoCollector *filenames.InfoCollector

	// ManageHEICJPG determines whether to manage HEIC to JPG conversion options.
	ManageHEICJPG filters.HeicJpgFlag

	// ManageRawJPG determines how to manage raw and JPEG files.
	ManageRawJPG filters.RawJPGFlag

	// BurstFlag determines how to manage burst photos.
	ManageBurst filters.BurstFlag

	// ManageEpsonFastFoto enables the management of Epson FastFoto files.
	ManageEpsonFastFoto bool
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
