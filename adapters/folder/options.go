package folder

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/simulot/immich-go/adapters/folder"
	"github.com/simulot/immich-go/adapters/shared"
	"github.com/simulot/immich-go/app"
	cliflags "github.com/simulot/immich-go/internal/cliFlags"
	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/filetypes"
	"github.com/simulot/immich-go/internal/filters"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/simulot/immich-go/internal/namematcher"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// ImportFolderOptions represents the flags used for importing assets from a file system.
type ImportFolderOptions struct {
	// UsePathAsAlbumName determines whether to create albums based on the full path to the asset.
	UsePathAsAlbumName AlbumFolderMode `mapstructure:"folder-as-album" yaml:"folder-as-album" json:"folder-as-album" toml:"folder-as-album"`

	// AlbumNamePathSeparator specifies how multiple (sub) folders are joined when creating album names.
	AlbumNamePathSeparator string `mapstructure:"album-path-joiner" yaml:"album-path-joiner" json:"album-path-joiner" toml:"album-path-joiner"`

	// ImportIntoAlbum is the name of the album where all assets will be added.
	ImportIntoAlbum string `mapstructure:"into-album" yaml:"into-album" json:"into-album" toml:"into-album"`

	// BannedFiles is a list of file name patterns to be excluded from the import process.
	BannedFiles namematcher.List `mapstructure:"ban-file" yaml:"ban-file" json:"ban-file" toml:"ban-file"`

	// Recursive indicates whether to explore the folder and all its sub-folders.
	Recursive bool `mapstructure:"recursive" yaml:"recursive" json:"recursive" toml:"recursive"`

	// InclusionFlags controls the file extensions to be included in the import process.
	InclusionFlags cliflags.InclusionFlags `mapstructure:",squash" yaml:",inline" json:",inline" toml:",inline"`

	// // ExifToolFlags specifies options for the exif.
	// ExifToolFlags exif.ExifToolFlags

	// IgnoreSideCarFiles indicates whether to ignore XMP files during the import process.
	IgnoreSideCarFiles bool `mapstructure:"ignore-sidecar-files" yaml:"ignore-sidecar-files" json:"ignore-sidecar-files" toml:"ignore-sidecar-files"`

	shared.StackOptions

	// Tags is a list of tags to be added to the imported assets.
	Tags []string `mapstructure:"tag" yaml:"tag" json:"tag" toml:"tag"`

	// Folder as tags
	FolderAsTags bool `mapstructure:"folder-as-tags" yaml:"folder-as-tags" json:"folder-as-tags" toml:"folder-as-tags"`

	// SessionTag indicates whether to add a session tag to the imported assets.
	SessionTag bool   `mapstructure:"session-tag" yaml:"session-tag" json:"session-tag" toml:"session-tag"`
	session    string // Session tag value

	// TakeDateFromFilename indicates whether to take the date from the filename if the date isn't available in the image.
	TakeDateFromFilename bool `mapstructure:"date-from-name" yaml:"date-from-name" json:"date-from-name" toml:"date-from-name"`

	// Use picasa albums
	PicasaAlbum bool `mapstructure:"album-picasa" yaml:"album-picasa" json:"album-picasa" toml:"album-picasa"`

	// Use icloud takeout metadata (albums & creation date)
	ICloudTakeout          bool `mapstructure:"icloud-takeout" yaml:"icloud-takeout" json:"icloud-takeout" toml:"icloud-takeout"`
	ICloudMemoriesAsAlbums bool `mapstructure:"memories" yaml:"memories" json:"memories" toml:"memories"`

	Client         app.Client
	TZ             *time.Location
	SupportedMedia filetypes.SupportedMedia
	InfoCollector  *filenames.InfoCollector
}

func (o *ImportFolderOptions) RegisterFlags(flags *pflag.FlagSet, cmd *cobra.Command) {
	o.ManageHEICJPG = filters.HeicJpgNothing
	o.ManageRawJPG = filters.RawJPGNothing
	o.ManageBurst = filters.BurstNothing
	o.Recursive = true
	o.SupportedMedia = filetypes.DefaultSupportedMedia
	o.UsePathAsAlbumName = "none"
	o.BannedFiles, _ = namematcher.New(shared.DefaultBannedFiles...)

	flags.Var(&o.BannedFiles, "ban-file", "Exclude a file based on a pattern (case-insensitive). Can be specified multiple times.")
	flags.StringVar(&o.ImportIntoAlbum, "into-album", "", "Specify an album to import all files into")
	flags.Var(&o.UsePathAsAlbumName, "folder-as-album", "Import all files in albums defined by the folder structure. Can be set to 'FOLDER' to use the folder name as the album name, or 'PATH' to use the full path as the album name")
	flags.StringVar(&o.AlbumNamePathSeparator, "album-path-joiner", " / ", "Specify a string to use when joining multiple folder names to create an album name (e.g. ' ',' - ')")
	flags.BoolVar(&o.Recursive, "recursive", true, "Explore the folder and all its sub-folders")
	flags.Var(&o.BannedFiles, "ban-file", "Exclude a file based on a pattern (case-insensitive). Can be specified multiple times.")
	flags.BoolVar(&o.IgnoreSideCarFiles, "ignore-sidecar-files", false, "Don't upload sidecar with the photo.")
	flags.StringSliceVar(&o.Tags, "tag", nil, "Add tags to the imported assets. Can be specified multiple times. Hierarchy is supported using a / separator (e.g. 'tag1/subtag1')")
	flags.BoolVar(&o.FolderAsTags, "folder-as-tags", false, "Use the folder structure as tags, (ex: the file  holiday/summer 2024/file.jpg will have the tag holiday/summer 2024)")
	flags.BoolVar(&o.SessionTag, "session-tag", false, "Tag uploaded photos with a tag \"{immich-go}/YYYY-MM-DD HH-MM-SS\"")
	flags.BoolVar(&o.TakeDateFromFilename, "date-from-name", true, "Use the date from the filename if the date isn't available in the metadata (Only for jpg, mp4, heic, dng, cr2, cr3, arw, raf, nef, mov)")

	o.InclusionFlags.RegisterFlags(flags, "") // selection per extension

	// Stacking is available only for upload
	if cmd.Parent() != nil && cmd.Parent().Name() == "upload" {
		o.StackOptions.RegisterFlags(flags) // stack options
	}

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

func (options *ImportFolderOptions) Run(cmd *cobra.Command, args []string, app *app.Application) error {
	ctx := cmd.Context()
	log := app.Log()
	err := options.Client.Open(ctx, app)
	if err != nil {
		return nil
	}
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
	options.SupportedMedia = options.Client.Immich.SupportedMedia()
	options.StackOptions.Filters = append(options.StackOptions.Filters, options.ManageBurst.GroupFilter(), options.ManageRawJPG.GroupFilter(), options.ManageHEICJPG.GroupFilter())

	options.InfoCollector = filenames.NewInfoCollector(app.GetTZ(), options.SupportedMedia)
	adapter, err := folder.NewLocalFiles(ctx, app.Jnl(), options, fsyss...)
	if err != nil {
		return err
	}

	return newUpload(UpModeFolder, app, upOptions).run(ctx, adapter, app, fsyss)
}

/*
func (o *ImportFolderOptions) AddFromFolderFlags(cmd *cobra.Command, parent *cobra.Command) {
	o.ManageHEICJPG = filters.HeicJpgNothing
	o.ManageRawJPG = filters.RawJPGNothing
	o.ManageBurst = filters.BurstNothing
	o.Recursive = true
	o.SupportedMedia = filetypes.DefaultSupportedMedia
	o.UsePathAsAlbumName = FolderModeNone
	o.BannedFiles, _ = namematcher.New()
	o.ICloudTakeout = false
	o.PicasaAlbum = false
	o.RegisterFlags(cmd.Flags())
	if parent != nil && parent.Name() == UploadCmdName {
		o.UploadFlags.RegisterFlags(cmd.Flags())
	}
}


func (o *ImportFolderOptions) AddFromICloudFlags(cmd *cobra.Command, parent *cobra.Command) {
	o.ManageHEICJPG = filters.HeicJpgNothing
	o.ManageRawJPG = filters.RawJPGNothing
	o.ManageBurst = filters.BurstNothing
	o.Recursive = true
	o.SupportedMedia = filetypes.DefaultSupportedMedia
	o.UsePathAsAlbumName = FolderModeNone
	o.BannedFiles, _ = namematcher.New(DefaultBannedFiles...)

	o.ICloudTakeout = true
	cmd.Flags().BoolVar(&o.ICloudMemoriesAsAlbums, "memories", false, "Import icloud memories as albums (default: false)")
	o.PicasaAlbum = false
	if parent != nil && parent.Name() == UploadCmdName {
		o.UploadFlags.RegisterFlags(cmd.Flags())
	}
}

func (o *ImportFolderOptions) AddFromPicasaFlags(cmd *cobra.Command, parent *cobra.Command) {
	o.ManageHEICJPG = filters.HeicJpgNothing
	o.ManageRawJPG = filters.RawJPGNothing
	o.ManageBurst = filters.BurstNothing
	o.Recursive = true
	o.SupportedMedia = filetypes.DefaultSupportedMedia
	o.UsePathAsAlbumName = FolderModeNone
	o.BannedFiles, _ = namematcher.New(DefaultBannedFiles...)

	o.ICloudTakeout = false
	o.PicasaAlbum = true
	if parent != nil && parent.Name() == UploadCmdName {
		o.UploadFlags.RegisterFlags(cmd.Flags())
	}
}

*/

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

// MarshalJSON implements json.Marshaler
func (m AlbumFolderMode) MarshalJSON() ([]byte, error) {
	return []byte(`"` + m.String() + `"`), nil
}

// UnmarshalJSON implements json.Unmarshaler
func (m *AlbumFolderMode) UnmarshalJSON(data []byte) error {
	if len(data) < 2 || data[0] != '"' || data[len(data)-1] != '"' {
		return fmt.Errorf("invalid JSON string for AlbumFolderMode")
	}
	s := string(data[1 : len(data)-1])
	return m.Set(s)
}

// MarshalYAML implements yaml.Marshaler
func (m AlbumFolderMode) MarshalYAML() (interface{}, error) {
	return m.String(), nil
}

// UnmarshalYAML implements yaml.Unmarshaler
func (m *AlbumFolderMode) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	return m.Set(s)
}

// MarshalText implements encoding.TextMarshaler
func (m AlbumFolderMode) MarshalText() ([]byte, error) {
	return []byte(m.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler
func (m *AlbumFolderMode) UnmarshalText(data []byte) error {
	return m.Set(string(data))
}
