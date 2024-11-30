package exif

import (
	"io"
	"time"

	"github.com/simulot/immich-go/internal/assets"
)

// GetMetaData read metadata from the asset file to  enrich the metadata structure
func GetMetaData(r io.Reader, name string, local *time.Location) (*assets.Metadata, error) {
	return MetadataFromDirectRead(r, name, local)
}

// MetadataFromExiftool call exiftool to get exif data
// func MetadataFromExiftool(f io.Reader, name string, options ExifToolFlags) (*assets.Metadata, error) {
// 	// be sure the file is completely extracted in the temp file
// 	_, err := io.Copy(io.Discard, f)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return options.et.ReadMetaData(name)
// }
