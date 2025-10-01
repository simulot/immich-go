// Package gp provides functionality for importing Google Photos takeout into Immich.

package gp

import (
	"time"

	"github.com/simulot/immich-go/adapters/folder"
	cliflags "github.com/simulot/immich-go/internal/cliFlags"
	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/filetypes"
	"github.com/simulot/immich-go/internal/filters"
	"github.com/simulot/immich-go/internal/namematcher"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type UploadFlags struct {
	// ManageHEICJPG determines whether to manage HEIC to JPG conversion options.
	ManageHEICJPG filters.HeicJpgFlag

	// ManageRawJPG determines how to manage raw and JPEG files.
	ManageRawJPG filters.RawJPGFlag

	// BurstFlag determines how to manage burst photos.
	ManageBurst filters.BurstFlag

	// ManageEpsonFastFoto enables the management of Epson FastFoto files.
	ManageEpsonFastFoto bool
}

func (o *UploadFlags) RegisterFlags(flags *pflag.FlagSet) {
	flags.Var(&o.ManageHEICJPG, "manage-heic-jpeg", "Manage coupled HEIC and JPEG files. Possible values: NoStack, KeepHeic, KeepJPG, StackCoverHeic, StackCoverJPG")
	flags.Var(&o.ManageRawJPG, "manage-raw-jpeg", "Manage coupled RAW and JPEG files. Possible values: NoStack, KeepRaw, KeepJPG, StackCoverRaw, StackCoverJPG")
	flags.Var(&o.ManageBurst, "manage-burst", "Manage burst photos. Possible values: NoStack, Stack, StackKeepRaw, StackKeepJPEG")
	flags.BoolVar(&o.ManageEpsonFastFoto, "manage-epson-fastfoto", false, "Manage Epson FastFoto file (default: false)")
}

// ImportFlags represents the command-line flags for the Google Photos takeout import command.
type ImportFlags struct {
	UploadFlags
	// CreateAlbums determines whether to create albums in Immich that match the albums in the Google Photos takeout.
	CreateAlbums bool `mapstructure:"sync-albums" yaml:"sync-albums" json:"sync-albums" toml:"sync-albums"`

	// ImportFromAlbum specifies the name of the Google Photos album to import from. If empty, all albums will be imported.
	ImportFromAlbum string `mapstructure:"from-album-name" yaml:"from-album-name" json:"from-album-name" toml:"from-album-name"`

	// ImportIntoAlbum specifies the name of the album to import assets into.
	ImportIntoAlbum string `mapstructure:"into-album" yaml:"into-album" json:"into-album" toml:"into-album"`

	// PartnerSharedAlbum specifies the name of the album to add partner's photos to.
	PartnerSharedAlbum string `mapstructure:"partner-shared-album" yaml:"partner-shared-album" json:"partner-shared-album" toml:"partner-shared-album"`

	// KeepTrashed determines whether to import photos that are marked as trashed in Google Photos.
	KeepTrashed bool `mapstructure:"include-trashed" yaml:"include-trashed" json:"include-trashed" toml:"include-trashed"`

	// KeepPartner determines whether to import photos from the partner's Google Photos account.
	KeepPartner bool `mapstructure:"include-partner" yaml:"include-partner" json:"include-partner" toml:"include-partner"`

	// KeepUntitled determines whether to include photos from albums without a title in the import process.
	KeepUntitled bool `mapstructure:"include-untitled-albums" yaml:"include-untitled-albums" json:"include-untitled-albums" toml:"include-untitled-albums"`

	// KeepArchived determines whether to import archived Google Photos.
	KeepArchived bool `mapstructure:"include-archived" yaml:"include-archived" json:"include-archived" toml:"include-archived"`

	// KeepJSONLess determines whether to import photos that do not have a matching JSON file in the takeout.
	KeepJSONLess bool `mapstructure:"include-unmatched" yaml:"include-unmatched" json:"include-unmatched" toml:"include-unmatched"`

	// Flags  for controlling the extensions of the files to be uploaded
	InclusionFlags cliflags.InclusionFlags `mapstructure:",squash" yaml:",inline" json:",inline" toml:",inline"`

	// List of banned files
	BannedFiles namematcher.List `mapstructure:"ban-file" yaml:"ban-file" json:"ban-file" toml:"ban-file"` // List of banned file name patterns

	// SupportedMedia represents the server's actual list of supported media. This is not a flag.
	SupportedMedia filetypes.SupportedMedia

	// InfoCollector collects information about filenames.
	InfoCollector *filenames.InfoCollector

	// Tags is a list of tags to be added to the imported assets.
	Tags []string `mapstructure:"tag" yaml:"tag" json:"tag" toml:"tag"`

	// SessionTag indicates whether to add a session tag to the imported assets.
	SessionTag bool   `mapstructure:"session-tag" yaml:"session-tag" json:"session-tag" toml:"session-tag"`
	session    string // Session tag value

	// Add the takeout file name as tag
	TakeoutTag  bool `mapstructure:"takeout-tag" yaml:"takeout-tag" json:"takeout-tag" toml:"takeout-tag"`
	TakeoutName string

	// PeopleTag indicates whether to add a people tag to the imported assets.
	PeopleTag bool `mapstructure:"people-tag" yaml:"people-tag" json:"people-tag" toml:"people-tag"`
	// Timezone
	TZ *time.Location
}

func (o *ImportFlags) AddFromGooglePhotosFlags(cmd *cobra.Command, parent *cobra.Command) {
	o.BannedFiles, _ = namematcher.New(folder.DefaultBannedFiles...)

	// exif.AddExifToolFlags(cmd, &o.ExifToolFlags)
	o.SupportedMedia = filetypes.DefaultSupportedMedia
	o.RegisterFlags(cmd.Flags())

	if parent != nil && parent.Name() == "upload" {
		o.UploadFlags.RegisterFlags(cmd.Flags())
	}
}

func (o *ImportFlags) RegisterFlags(flags *pflag.FlagSet) {
	flags.BoolVar(&o.CreateAlbums, "sync-albums", true, "Automatically create albums in Immich that match the albums in your Google Photos takeout")
	flags.StringVar(&o.ImportFromAlbum, "from-album-name", "", "Only import photos from the specified Google Photos album")
	flags.BoolVar(&o.KeepUntitled, "include-untitled-albums", false, "Include photos from albums without a title in the import process")
	flags.BoolVarP(&o.KeepTrashed, "include-trashed", "t", false, "Import photos that are marked as trashed in Google Photos")
	flags.BoolVarP(&o.KeepPartner, "include-partner", "p", true, "Import photos from your partner's Google Photos account")
	flags.StringVar(&o.PartnerSharedAlbum, "partner-shared-album", "", "Add partner's photo to the specified album name")
	flags.BoolVarP(&o.KeepArchived, "include-archived", "a", true, "Import archived Google Photos")
	flags.BoolVarP(&o.KeepJSONLess, "include-unmatched", "u", false, "Import photos that do not have a matching JSON file in the takeout")
	flags.Var(&o.BannedFiles, "ban-file", "Exclude a file based on a pattern (case-insensitive). Can be specified multiple times.")
	flags.StringSliceVar(&o.Tags, "tag", nil, "Add tags to the imported assets. Can be specified multiple times. Hierarchy is supported using a / separator (e.g. 'tag1/subtag1')")
	flags.BoolVar(&o.SessionTag, "session-tag", false, "Tag uploaded photos with a tag \"{immich-go}/YYYY-MM-DD HH-MM-SS\"")
	flags.BoolVar(&o.TakeoutTag, "takeout-tag", true, "Tag uploaded photos with a tag \"{takeout}/takeout-YYYYMMDDTHHMMSSZ\"")
	flags.BoolVar(&o.PeopleTag, "people-tag", true, "Tag uploaded photos with tags \"people/name\" found in the JSON file")
	o.InclusionFlags.RegisterFlags(flags, "")
}
