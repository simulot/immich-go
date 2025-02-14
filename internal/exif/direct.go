package exif

/*
	Read metadata from a file not using exiftool.

	TODO: Use sync.Pool for buffers
*/
import (
	"bytes"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/rwcarlsen/goexif/exif"
	"github.com/simulot/immich-go/internal/assets"
)

// MetadataFromDirectRead read the file using GO package
func MetadataFromDirectRead(f io.Reader, name string, localTZ *time.Location) (*assets.Metadata, error) {
	var md *assets.Metadata
	var err error
	ext := strings.ToLower(path.Ext(name))

	switch strings.ToLower(ext) {
	case ".heic", ".heif":
		md, err = readHEIFMetadata(f, localTZ)
	case ".jpg", ".jpeg", ".dng", ".cr2", ".arw", ".raf", ".nef":
		md, err = readExifMetadata(f, localTZ)
	case ".mp4", ".mov":
		md, err = readMP4Metadata(f)
	case ".cr3":
		md, err = readCR3Metadata(f, localTZ)
	default:
		return nil, fmt.Errorf("can't read metadata for this format '%s'", ext)
	}
	if err != nil {
		return nil, fmt.Errorf("can't read metadata: %w", err)
	}

	return md, nil
}

// readExifMetadata locate the Exif part and return the date of capture
func readExifMetadata(r io.Reader, localTZ *time.Location) (*assets.Metadata, error) {
	// try to read the Exif data directly
	readBuffer := bytes.NewBuffer(make([]byte, searchBufferSize))
	r2 := io.TeeReader(r, readBuffer)
	x, err := exif.Decode(r2)
	if err == nil || !exif.IsCriticalError(err) {
		return getExifMetadata(x, localTZ)
	}
	b := make([]byte, searchBufferSize)

	// search for the Exif header
	r, err = searchPattern(io.MultiReader(readBuffer, r), []byte("Exif\x00\x00"), b)
	if err == nil {
		x, err = exif.Decode(r)
		if err == nil || !exif.IsCriticalError(err) {
			return getExifMetadata(x, localTZ)
		}
	}
	return nil, err
}

const searchBufferSize = 32 * 1024

// readHEIFMetadata locate the Exif part and return the date of capture
func readHEIFMetadata(r io.Reader, localTZ *time.Location) (*assets.Metadata, error) {
	b := make([]byte, searchBufferSize)
	r, err := searchPattern(r, []byte{0x45, 0x78, 0x69, 0x66, 0, 0, 0x4d, 0x4d}, b)
	if err != nil {
		return nil, err
	}

	filler := make([]byte, 6)
	_, err = r.Read(filler)
	if err != nil {
		return nil, err
	}
	x, err := exif.Decode(r)
	if err == nil || !exif.IsCriticalError(err) {
		return getExifMetadata(x, localTZ)
	}
	return nil, err
}

// readMP4Metadata locate the mvhd atom and decode the date of capture
func readMP4Metadata(r io.Reader) (*assets.Metadata, error) {
	b := make([]byte, searchBufferSize)

	r, err := searchPattern(r, []byte{'m', 'v', 'h', 'd'}, b)
	if err != nil {
		return nil, err
	}
	atom, err := decodeMvhdAtom(r)
	if err != nil {
		return nil, err
	}
	t := atom.CreationTime
	if t.Year() < 2000 {
		t = atom.ModificationTime
	}
	return &assets.Metadata{DateTaken: t}, nil
}

// readCR3Metadata locate the CMT1 atom and decode the date of capture
func readCR3Metadata(r io.Reader, localTZ *time.Location) (*assets.Metadata, error) {
	b := make([]byte, searchBufferSize)

	r, err := searchPattern(r, []byte("CMT1"), b)
	if err != nil {
		return nil, err
	}

	filler := make([]byte, 4)
	_, err = r.Read(filler)
	if err != nil {
		return nil, err
	}
	x, err := exif.Decode(r)
	if err == nil || !exif.IsCriticalError(err) {
		return getExifMetadata(x, localTZ)
	}
	return nil, err
}

// type exifDumper struct{}

// func (exifDumper) Walk(name exif.FieldName, tag *tiff.Tag) error {
// 	fmt.Printf("%s: %s\n", name, tag)
// 	return nil
// }

// getExifMetadata extract the date and location from the Exif data

func getExifMetadata(x *exif.Exif, local *time.Location) (*assets.Metadata, error) {
	var err error

	// _ = x.Walk(exifDumper{})

	md := &assets.Metadata{}
	// md.DateTaken, err = readGPSTimeStamp(x, local)
	// if err != nil || md.DateTaken.IsZero() {
	// GPS Time Stamp is not reliable

	md.DateTaken, err = readDateTime(x, exif.DateTimeOriginal, exif.SubSecTimeOriginal, local)
	if err != nil {
		md.DateTaken, err = readDateTime(x, exif.DateTime, exif.SubSecTime, local)
	}
	if err == nil {
		lat, lon, err := x.LatLong()
		if err == nil {
			md.Latitude = lat
			md.Longitude = lon
		}
	}
	return md, err
}

// readDateTime with subsecond when possible
func readDateTime(x *exif.Exif, dateTag exif.FieldName, subSecTag exif.FieldName, local *time.Location) (time.Time, error) {
	date, err := getTagSting(x, dateTag)
	if err != nil {
		return time.Time{}, err
	}
	subSec, err := getTagSting(x, subSecTag)
	if err == nil {
		subSec += "000"
		date = date + "." + subSec[:3]
		return time.ParseInLocation("2006:01:02 15:04:05.000", date, local)
	}
	return time.ParseInLocation("2006:01:02 15:04:05", date, local)
}

/*
// readGPSTimeStamp extract the date from the GPS data

	func readGPSTimeStamp(x *exif.Exif, _ *time.Location) (time.Time, error) {
		tag, err := getTagSting(x, exif.GPSDateStamp)
		if err == nil {
			var tags *tiff.Tag
			tags, err = x.Get(exif.GPSTimeStamp)
			if err == nil {
				tag = tag + " " + fmt.Sprintf("%02d:%02d:%02dZ", ratToInt(tags, 0), ratToInt(tags, 1), ratToInt(tags, 2))
				t, err := time.ParseInLocation("2006:01:02 15:04:05Z", tag, time.UTC)
				if err == nil {
					return t, nil
				}
			}
		}
		return time.Time{}, err
	}

	func ratToInt(t *tiff.Tag, i int) int {
		n, d, err := t.Rat2(i)
		if err != nil {
			return 0
		}
		if d == 1 {
			return int(n)
		}
		return int(float64(n) / float64(d))
	}
*/

func getTagSting(x *exif.Exif, tagName exif.FieldName) (string, error) {
	t, err := x.Get(tagName)
	if err != nil {
		return "", err
	}
	s := strings.TrimRight(strings.TrimLeft(t.String(), `"`), `"`)
	return s, nil
}
