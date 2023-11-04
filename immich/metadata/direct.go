package metadata

import (
	"fmt"
	"io"
	"io/fs"
	"path"
	"strings"
	"time"
)

type MetaData struct {
	DateTaken                     time.Time
	Latitude, Longitude, Altitude float64
}

func GetFileMetaData(fsys fs.FS, name string) (MetaData, error) {
	f, err := fsys.Open(name)
	if err != nil {
		return MetaData{}, err
	}
	defer f.Close()
	return GetFromReader(f, path.Ext(name))
}

// GetMetaData makes its best efforts to get the date of capture based on
// - if the name matches a at least 4 digits for the year, 2 for month, 2 for day, in this order.
//   It takes the hour, minute, second when present. Very fast
//
// - file content if the file includes some metadata, need read a part of the file
//
//

func GetFromReader(r io.Reader, ext string) (MetaData, error) {
	meta := MetaData{}
	var err error
	var dateTaken time.Time
	switch strings.ToLower(ext) {
	case ".heic", ".heif":
		dateTaken, err = readHEIFDateTaken(r)
	case ".jpg", ".jpeg", ".dng", ".cr2":
		dateTaken, err = readExifDateTaken(r)
	case ".mp4", ".mov":
		dateTaken, err = readMP4DateTaken(r)
	case ".cr3":
		dateTaken, err = readCR3DateTaken(r)
	default:
		err = fmt.Errorf("can't determine the taken date from metadata (%s)", ext)
	}
	meta.DateTaken = dateTaken
	return meta, err
}

// readExifDateTaken pase the file for Exif DateTaken
func readExifDateTaken(r io.Reader) (time.Time, error) {

	md, err := getExifFromReader(r)
	return md.DateTaken, err
}

// readHEIFDateTaken locate the Exif part and return the date of capture
func readHEIFDateTaken(r io.Reader) (time.Time, error) {

	r, err := seekReaderAtPattern(r, []byte{0x45, 0x78, 0x69, 0x66, 0, 0, 0x4d, 0x4d})
	if err != nil {
		return time.Time{}, err
	}

	filler := make([]byte, 6)
	r.Read(filler)

	md, err := getExifFromReader(r)
	return md.DateTaken, err
}

// readMP4DateTaken locate the mvhd atom and decode the date of capture
func readMP4DateTaken(r io.Reader) (time.Time, error) {

	b, err := searchPattern(r, []byte{'m', 'v', 'h', 'd'}, 60)
	if err != nil {
		return time.Time{}, err
	}
	atom, err := decodeMvhdAtom(b)
	if err != nil {
		return time.Time{}, err
	}
	return atom.CreationTime, nil
}

func readCR3DateTaken(r io.Reader) (time.Time, error) {
	r, err := seekReaderAtPattern(r, []byte("CMT1"))
	if err != nil {
		return time.Time{}, err
	}

	filler := make([]byte, 4)
	r.Read(filler)

	md, err := getExifFromReader(r)
	return md.DateTaken, err

}
