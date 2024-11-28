package exif

import (
	"io"

	"github.com/simulot/immich-go/internal/assets"
)

// GetMetaData read metadata from the asset file to  enrich the metadata structure
func GetMetaData(a *assets.Asset, md *assets.Metadata, options ExifToolFlags) (*assets.Metadata, error) {
	if options.UseExifTool && options.et != nil {
		return MetadataFromExiftool(a, options)
	}
	return MetadataFromDirectRead(a, options.Timezone.TZ)
}

// MetadataFromExiftool call exiftool to get exif data
func MetadataFromExiftool(a *assets.Asset, options ExifToolFlags) (*assets.Metadata, error) {
	f, tmp, err := a.PartialSourceReader()
	if err != nil {
		return nil, err
	}
	// be sure the file is completely extracted in the temp file
	_, err = io.Copy(io.Discard, f)
	if err != nil {
		return nil, err
	}

	return options.et.ReadMetaData(tmp)
}
