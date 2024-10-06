// Package gp provides functionality for importing Google Photos takeout into Immich.

package gp

import (
	cliflags "github.com/simulot/immich-go/internal/cliFlags"
	"github.com/simulot/immich-go/internal/metadata"
	"github.com/simulot/immich-go/internal/namematcher"
)

// ImportFlags represents the command-line flags for the Google Photos takeout import command.
type ImportFlags struct {
	// UseJSONMetadata  indicates whether to use JSON metadata. A virtual XMP sidecar is created to convey the GPS location and the date of capture
	UseJSONMetadata bool

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

	// DateHandlingFlags provides options for handling the capture date of the assets.
	DateHandlingFlags cliflags.DateHandlingFlags

	// ExifToolFlags specifies options for the exif.
	ExifToolFlags metadata.ExifToolFlags

	// List of banned files
	BannedFiles namematcher.List // List of banned file name patterns

	// SupportedMedia represents the server's actual list of supported media. This is not a flag.
	SupportedMedia metadata.SupportedMedia
}
