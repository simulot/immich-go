package exif

import (
	"io"
	"strings"

	"github.com/simulot/immich-go/internal/assets"
	cliflags "github.com/simulot/immich-go/internal/cliFlags"
)

// GetMetaData read metadata from the asset file to  enrich the metadata structure
func GetMetaData(a *assets.Asset, md *assets.Metadata, options ExifToolFlags) error {
	ms := strings.Split(string(options.DateMethod), "-")
	for _, m := range ms {
		switch cliflags.DateMethod(m) {
		case cliflags.DateMethodNone:
			return nil
		case cliflags.DateMethodEXIF:
			if options.et != nil {
				err := MetadataFromExiftool(a, md, options)
				if err != nil {
					return err
				}
				if !md.DateTaken.IsZero() {
					return nil
				}
			}
			continue // no exiftool... try next method
		case cliflags.DateMethodName:
			md.DateTaken = a.NameInfo.Taken
			return nil
		}
	}
	return nil
}

// MetadataFromExiftool call exiftool to get exif data
func MetadataFromExiftool(a *assets.Asset, md *assets.Metadata, options ExifToolFlags) error {
	f, tmp, err := a.PartialSourceReader()
	if err != nil {
		return nil
	}
	// be sure the file is completely extracted in the temp file
	_, err = io.Copy(io.Discard, f)
	if err != nil {
		return nil
	}

	return options.et.ReadMetaData(tmp, md)
}
