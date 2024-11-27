package exif

import (
	"fmt"
	"io/fs"
	"strings"
	"time"

	"github.com/rwcarlsen/goexif/exif"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/tzone"
)

// MetadataFromDirectRead read the file using GO package
func MetadataFromDirectRead(f fs.File, localTZ *time.Location) (*assets.Metadata, error) {
	var md assets.Metadata

	local, err := tzone.Local()
	if err != nil {
		return nil, fmt.Errorf("can't get local timezone: %w", err)
	}
	// Decode the EXIF data
	x, err := exif.Decode(f)
	if err != nil && exif.IsCriticalError(err) {
		return nil, nil
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

	return &md, err
}

func getTagSting(x *exif.Exif, tagName exif.FieldName) (string, error) {
	t, err := x.Get(tagName)
	if err != nil {
		return "", err
	}
	s := strings.TrimRight(strings.TrimLeft(t.String(), `"`), `"`)
	return s, nil
}
