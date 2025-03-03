// Package gp provides functionality for importing Google Photos takeout into Immich.

package gp

import (
	"time"

	cliflags "github.com/simulot/immich-go/internal/cliFlags"
	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/filetypes"
	"github.com/simulot/immich-go/internal/filters"
	"github.com/simulot/immich-go/internal/namematcher"
	"github.com/spf13/cobra"
)

// ImportFlags represents the command-line flags for the Google Photos takeout import command.
type ImportFlags struct {
	// CreateAlbums determines whether to create albums in Immich that match the albums in the Google Photos takeout.
	CreateAlbums bool

	// ImportFromAlbum specifies the name of the Google Photos album to import from. If empty, all albums will be imported.
	ImportFromAlbum string

	// ImportIntoAlbum specifies the name of the album to import assets into.
	ImportIntoAlbum string

	// PartnerSharedAlbum specifies the name of the album to add partner's photos to.
	PartnerSharedAlbum string

	// KeepTrashed determines whether to import photos that are marked as trashed in Google Photos.
	KeepTrashed bool

	// KeepPartner determines whether to import photos from the partner's Google Photos account.
	KeepPartner bool

	// KeepUntitled determines whether to include photos from albums without a title in the import process.
	KeepUntitled bool

	// KeepArchived determines whether to import archived Google Photos.
	KeepArchived bool

	// KeepJSONLess determines whether to import photos that do not have a matching JSON file in the takeout.
	KeepJSONLess bool

	// Flags  for controlling the extensions of the files to be uploaded
	InclusionFlags cliflags.InclusionFlags

	// List of banned files
	BannedFiles namematcher.List // List of banned file name patterns

	// SupportedMedia represents the server's actual list of supported media. This is not a flag.
	SupportedMedia filetypes.SupportedMedia

	// InfoCollector collects information about filenames.
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

	// SessionTag indicates whether to add a session tag to the imported assets.
	SessionTag bool
	session    string // Session tag value

	// Add the takeout file name as tag
	TakeoutTag  bool
	TakeoutName string

	// PeopleTag indicates whether to add a people tag to the imported assets.
	PeopleTag bool
	// Timezone
	TZ *time.Location
}

func (o *ImportFlags) AddFromGooglePhotosFlags(cmd *cobra.Command, parent *cobra.Command) {
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
	cmd.Flags().BoolVar(&o.CreateAlbums, "sync-albums", true, "Automatically create albums in Immich that match the albums in your Google Photos takeout")
	cmd.Flags().StringVar(&o.ImportFromAlbum, "from-album-name", "", "Only import photos from the specified Google Photos album")
	cmd.Flags().BoolVar(&o.KeepUntitled, "include-untitled-albums", false, "Include photos from albums without a title in the import process")
	cmd.Flags().BoolVarP(&o.KeepTrashed, "include-trashed", "t", false, "Import photos that are marked as trashed in Google Photos")
	cmd.Flags().BoolVarP(&o.KeepPartner, "include-partner", "p", true, "Import photos from your partner's Google Photos account")
	cmd.Flags().StringVar(&o.PartnerSharedAlbum, "partner-shared-album", "", "Add partner's photo to the specified album name")
	cmd.Flags().BoolVarP(&o.KeepArchived, "include-archived", "a", true, "Import archived Google Photos")
	cmd.Flags().BoolVarP(&o.KeepJSONLess, "include-unmatched", "u", false, "Import photos that do not have a matching JSON file in the takeout")
	cmd.Flags().Var(&o.BannedFiles, "ban-file", "Exclude a file based on a pattern (case-insensitive). Can be specified multiple times.")
	cmd.Flags().StringSliceVar(&o.Tags, "tag", nil, "Add tags to the imported assets. Can be specified multiple times. Hierarchy is supported using a / separator (e.g. 'tag1/subtag1')")
	cmd.Flags().BoolVar(&o.SessionTag, "session-tag", false, "Tag uploaded photos with a tag \"{immich-go}/YYYY-MM-DD HH-MM-SS\"")
	cmd.Flags().BoolVar(&o.TakeoutTag, "takeout-tag", true, "Tag uploaded photos with a tag \"{takeout}/takeout-YYYYMMDDTHHMMSSZ\"")
	cmd.Flags().BoolVar(&o.PeopleTag, "people-tag", true, "Tag uploaded photos with tags \"people/name\" found in the JSON file")
	cliflags.AddInclusionFlags(cmd, &o.InclusionFlags)

	// exif.AddExifToolFlags(cmd, &o.ExifToolFlags)
	o.SupportedMedia = filetypes.DefaultSupportedMedia

	if parent != nil && parent.Name() == "upload" {
		cmd.Flags().Var(&o.ManageHEICJPG, "manage-heic-jpeg", "Manage coupled HEIC and JPEG files. Possible values: NoStack, KeepHeic, KeepJPG, StackCoverHeic, StackCoverJPG")
		cmd.Flags().Var(&o.ManageRawJPG, "manage-raw-jpeg", "Manage coupled RAW and JPEG files. Possible values: NoStack, KeepRaw, KeepJPG, StackCoverRaw, StackCoverJPG")
		cmd.Flags().Var(&o.ManageBurst, "manage-burst", "Manage burst photos. Possible values: NoStack, Stack, StackKeepRaw, StackKeepJPEG")
		cmd.Flags().BoolVar(&o.ManageEpsonFastFoto, "manage-epson-fastfoto", false, "Manage Epson FastFoto file (default: false)")
	}
}
