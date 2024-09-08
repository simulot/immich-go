// Package gp provides functionality for importing Google Photos takeout into Immich.
package gp

import (
	cliflags "github.com/simulot/immich-go/internal/cliFlags"
	"github.com/simulot/immich-go/internal/metadata"
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

	// SupportedMedia represents the server's actual list of supported media. This is not a flag.
	SupportedMedia metadata.SupportedMedia

	// Flags  for controlling the extensions of the files to be uploaded
	*cliflags.InclusionFlags

	// List of banned files
	BannedFiles namematcher.List // List of banned file name patterns
}

// AddGoogleTakeoutFlags adds the command-line flags for the Google Photos takeout import command to the provided Cobra command.
func AddGoogleTakeoutFlags(cmd *cobra.Command) *ImportFlags {
	flags := ImportFlags{}
	cmd.Flags().BoolVar(&flags.CreateAlbums, "sync-albums", true, "Automatically create albums in Immich that match the albums in your Google Photos takeout")
	cmd.Flags().StringVar(&flags.ImportFromAlbum, "import-from-album-name", "", "Only import photos from the specified Google Photos album")
	cmd.Flags().BoolVar(&flags.KeepUntitled, "include-untitled-albums", false, "Include photos from albums without a title in the import process")
	cmd.Flags().BoolVarP(&flags.KeepTrashed, "include-trashed", "t", false, "Import photos that are marked as trashed in Google Photos")
	cmd.Flags().BoolVarP(&flags.KeepPartner, "include-partner", "p", true, "Import photos from your partner's Google Photos account")
	cmd.Flags().StringVar(&flags.PartnerSharedAlbum, "partner-shared-album", "", "Add partner's photo to the specified album name")
	cmd.Flags().BoolVarP(&flags.KeepArchived, "include-archived", "a", true, "Import archived Google Photos")
	cmd.Flags().BoolVarP(&flags.KeepJSONLess, "include-unmatched", "u", false, "Import photos that do not have a matching JSON file in the takeout")
	cmd.Flags().Var(&flags.BannedFiles, "ban-file", "Exclude a file based on a pattern (case-insensitive). Can be specified multiple times.")
	flags.InclusionFlags = cliflags.AddInclusionFlags(cmd)
	return &flags
}
