package exif

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"
	"time"

	"github.com/rwcarlsen/goexif/exif"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/tzone"
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
	return MetadataFromDirectRead(f, options.ExifTimezone)
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

// MetadataFromDirectRead read the file using GO pakcage
func MetadataFromDirectRead(f fs.File, localTZ *time.Location) (*assets.Metadata, error) {
	var md *assets.Metadata

	local, err := tzone.Local()
	if err != nil {
		return md, fmt.Errorf("can't get local timezone: %w", err)
	}
	// Decode the EXIF data
	x, err := exif.Decode(f)
	if err != nil && exif.IsCriticalError(err) {
		if errors.Is(err, io.EOF) {
			return md, nil
		}
		return md, fmt.Errorf("can't get DateTaken: %w", err)
	}

	// TODO : Add more tags, as when using exiftool
	tag, err := getTagSting(x, exif.GPSDateStamp)
	if err == nil {
		md.DateTaken, err = time.ParseInLocation("2006:01:02 15:04:05Z", tag, local)
	}
	if err != nil {
		tag, err = getTagSting(x, exif.DateTimeOriginal)
		if err == nil {
			md.DateTaken, err = time.ParseInLocation("2006:01:02 15:04:05", tag, localTZ)
		}
	}
	if err != nil {
		tag, err = getTagSting(x, exif.DateTime)
		if err == nil {
			md.DateTaken, err = time.ParseInLocation("2006:01:02 15:04:05", tag, localTZ)
		}
	}

	return md, err
}

func getTagSting(x *exif.Exif, tagName exif.FieldName) (string, error) {
	t, err := x.Get(tagName)
	if err != nil {
		return "", err
	}
	s := strings.TrimRight(strings.TrimLeft(t.String(), `"`), `"`)
	return s, nil
}
