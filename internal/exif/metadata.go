package exif

import (
	"io"

	"github.com/simulot/immich-go/internal/assets"
)

// GetMetaData read metadata from the asset file to  enrich the metadata structure
func GetMetaData(r io.Reader, name string, options ExifToolFlags) (*assets.Metadata, error) {
	if options.UseExifTool && options.et != nil {
		return MetadataFromExiftool(r, name, options)
	}
	return MetadataFromDirectRead(r, name, options.Timezone.TZ)
}

// MetadataFromExiftool call exiftool to get exif data
func MetadataFromExiftool(f io.Reader, name string, options ExifToolFlags) (*assets.Metadata, error) {
	// be sure the file is completely extracted in the temp file
	_, err := io.Copy(io.Discard, f)
	if err != nil {
		return nil, err
	}

	return options.et.ReadMetaData(name)
}
