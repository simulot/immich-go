package metadata

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/rwcarlsen/goexif/exif"
)

func getExifFromReader(r io.Reader, localTZ *time.Location) (Metadata, error) {
	var md Metadata
	// Decode the EXIF data
	x, err := exif.Decode(r)
	if err != nil && exif.IsCriticalError(err) {
		if errors.Is(err, io.EOF) {
			return md, nil
		}
		return md, fmt.Errorf("can't get DateTaken: %w", err)
	}

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
