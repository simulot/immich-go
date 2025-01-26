package folder

import (
	"fmt"
	"strings"
	"time"

	cliflags "github.com/simulot/immich-go/internal/cliFlags"
	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/filetypes"
	"github.com/simulot/immich-go/internal/filters"
	"github.com/simulot/immich-go/internal/namematcher"
	"github.com/spf13/cobra"
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

	// // ExifToolFlags specifies options for the exif.
	// ExifToolFlags exif.ExifToolFlags

	// IgnoreSideCarFiles indicates whether to ignore XMP files during the import process.
	IgnoreSideCarFiles bool

	// Stack jpg/raw
	StackJpgWithRaw bool

	// Stack burst
	StackBurstPhotos bool

	// SupportedMedia is the server's actual list of supported media types.
	SupportedMedia filetypes.SupportedMedia

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

	// Tags is a list of tags to be added to the imported assets.
	Tags []string

	// Folder as tags
	FolderAsTags bool

	// SessionTag indicates whether to add a session tag to the imported assets.
	SessionTag bool
	session    string // Session tag value

	// TakeDateFromFilename indicates whether to take the date from the filename if the date isn't available in the image.
	TakeDateFromFilename bool

	// Use picasa albums
	PicasaAlbum bool

	// local time zone
	TZ *time.Location
}

func (o *ImportFolderOptions) AddFromFolderFlags(cmd *cobra.Command, parent *cobra.Command) {
	o.ManageHEICJPG = filters.HeicJpgNothing
	o.ManageRawJPG = filters.RawJPGNothing
	o.ManageBurst = filters.BurstNothing
	o.Recursive = true
	o.SupportedMedia = filetypes.DefaultSupportedMedia
	o.UsePathAsAlbumName = FolderModeNone
	o.BannedFiles, _ = namematcher.New(
		`@eaDir/`,
		`@__thumb/`,          // QNAP
		`SYNOFILE_THUMB_*.*`, // SYNOLOGY
		`Lightroom Catalog/`, // LR
		`thumbnails/`,        // Android photo
		`.DS_Store/`,         // Mac OS custom attributes
		`/._*`,               // MacOS resource files
		`.photostructure/`,   // PhotoStructure
	)
	cmd.Flags().StringVar(&o.ImportIntoAlbum, "into-album", "", "Specify an album to import all files into")
	cmd.Flags().Var(&o.UsePathAsAlbumName, "folder-as-album", "Import all files in albums defined by the folder structure. Can be set to 'FOLDER' to use the folder name as the album name, or 'PATH' to use the full path as the album name")
	cmd.Flags().StringVar(&o.AlbumNamePathSeparator, "album-path-joiner", " / ", "Specify a string to use when joining multiple folder names to create an album name (e.g. ' ',' - ')")
	cmd.Flags().BoolVar(&o.Recursive, "recursive", true, "Explore the folder and all its sub-folders")
	cmd.Flags().Var(&o.BannedFiles, "ban-file", "Exclude a file based on a pattern (case-insensitive). Can be specified multiple times.")
	cmd.Flags().BoolVar(&o.IgnoreSideCarFiles, "ignore-sidecar-files", false, "Don't upload sidecar with the photo.")

	cmd.Flags().StringSliceVar(&o.Tags, "tag", nil, "Add tags to the imported assets. Can be specified multiple times. Hierarchy is supported using a / separator (e.g. 'tag1/subtag1')")
	cmd.Flags().BoolVar(&o.FolderAsTags, "folder-as-tags", false, "Use the folder structure as tags, (ex: the file  holiday/summer 2024/file.jpg will have the tag holiday/summer 2024)")
	cmd.Flags().BoolVar(&o.SessionTag, "session-tag", false, "Tag uploaded photos with a tag \"{immich-go}/YYYY-MM-DD HH-MM-SS\"")

	cliflags.AddInclusionFlags(cmd, &o.InclusionFlags)
	cmd.Flags().BoolVar(&o.TakeDateFromFilename, "date-from-name", true, "Use the date from the filename if the date isn't available in the metadata (Only for jpg, mp4, heic, dng, cr2, cr3, arw, raf, nef, mov)")

	// exif.AddExifToolFlags(cmd, &o.ExifToolFlags) // disabled for now

	// upload specific flags, not for archive to folder
	if parent != nil && parent.Name() == "upload" {
		cmd.Flags().Var(&o.ManageHEICJPG, "manage-heic-jpeg", "Manage coupled HEIC and JPEG files. Possible values: KeepHeic, KeepJPG, StackCoverHeic, StackCoverJPG")
		cmd.Flags().Var(&o.ManageRawJPG, "manage-raw-jpeg", "Manage coupled RAW and JPEG files. Possible values: KeepRaw, KeepJPG, StackCoverRaw, StackCoverJPG")
		cmd.Flags().Var(&o.ManageBurst, "manage-burst", "Manage burst photos. Possible values: Stack, StackKeepRaw, StackKeepJPEG")
		cmd.Flags().BoolVar(&o.ManageEpsonFastFoto, "manage-epson-fastfoto", false, "Manage Epson FastFoto file (default: false)")
		cmd.Flags().BoolVar(&o.PicasaAlbum, "album-picasa", false, "Use Picasa album name found in .picasa.ini file (default: false)")
	}
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
