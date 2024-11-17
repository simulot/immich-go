package exif

import (
	"io"
	"io/fs"
	"time"

	"github.com/simulot/immich-go/internal/assets"
)

type ReadMetadataOptions struct {
	ExifTool     *ExifTool
	ExifTimezone *time.Location
}

// GetMetaData returns the metadata of the file according the method
func GetMetaData(f fs.File, options ReadMetadataOptions) (*assets.Metadata, error) {
	if options.ExifTool != nil {
		return MetadataFromExiftool(f, options)
	}
	return nil, nil // direct read disabled
	// return MetadataFromDirectRead(f, options.ExifTimezone)
}

// MetadataFromExiftool call exiftool to get exif data
func MetadataFromExiftool(f fs.File, options ReadMetadataOptions) (*assets.Metadata, error) {
	// Get information about the file
	i, err := f.Stat()
	if err != nil {
		return nil, err
	}

	// be sure the file is completely extracted in the temp file
	_, err = io.Copy(io.Discard, f)
	if err != nil {
		return nil, err
	}

	md, err := options.ExifTool.ReadMetaData(i.Name())
	return md, err
}
