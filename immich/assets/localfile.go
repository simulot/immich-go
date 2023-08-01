package assets

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/rwcarlsen/goexif/exif"
)

/*
	localFile structure hold information on assets used for building immich assets.

	The asset is taken into a fs.FS system which doesn't implement anything else than a strait
	reader.
	fsys can be a zip file, a DirFS, or anything else.

	It implements a way to read a minimal quantity of data to be able to take a decision
	about chose a file or discard it.

	implements fs.File and fs.FileInfo, Stat

*/

type LocalAssetFile struct {
	// Common fields
	FileName string // The asset's path in the fsys
	Title    string // Google Photos may a have title longer than the filename
	Album    string // The asset's album, if any
	Err      error  // keep errors encountered

	// Google Photos flags
	Trashed     bool // The asset is trashed
	Archived    bool // The asset is archived
	FromPartner bool // the asset comes from a partner

	FSys fs.FS // Asset's file system

	// Unexported fields
	isResolved bool      // True when the FileName is resolved
	dateTaken  time.Time // Accessible via DateTaken()
	size       int       // Accessible via Stat()

	// buffer management
	sourceFile fs.File   // the opened source file
	tempFile   *os.File  // buffer that keep partial reads available for the full file reading
	teeReader  io.Reader // write each read from it into the tempWriter
	reader     io.Reader // the reader that combines the partial read and original file for full file reading
}

// partialSourceReader open a reader on the current asset.
// each byte read from it is saved into a temporary file.
//
// It returns a TeeReader that writes each read byte from the source into the temporary file.
// The temporary file is discarded when the LocalAssetFile is closed

func (l *LocalAssetFile) partialSourceReader() (reader io.Reader, err error) {
	if l.sourceFile == nil {
		l.sourceFile, err = l.FSys.Open(l.FileName)
		if err != nil {
			return nil, err
		}
	}
	if l.tempFile == nil {
		tempDir, err := os.UserCacheDir()
		if err != nil {
			return nil, err
		}
		tempDir = filepath.Join(tempDir, "immich-go")
		os.Mkdir(tempDir, 0700)
		l.tempFile, err = os.CreateTemp(tempDir, "")
		if err != nil {
			return nil, err
		}
		if l.teeReader == nil {
			l.teeReader = io.TeeReader(l.sourceFile, l.tempFile)
		}
	}
	l.tempFile.Seek(0, 0)
	return io.MultiReader(l.tempFile, l.teeReader), nil
}

// Open return fs.File that reads previously read bytes followed by the actual file content.
func (l *LocalAssetFile) Open() (fs.File, error) {
	var err error
	if l.sourceFile == nil {
		l.sourceFile, err = l.FSys.Open(l.FileName)
		if err != nil {
			return nil, err
		}
	}
	if l.tempFile != nil {
		l.tempFile.Seek(0, 0)
		l.reader = io.MultiReader(l.tempFile, l.sourceFile)
	} else {
		l.reader = l.sourceFile
	}
	return l, nil
}

// Read
func (l *LocalAssetFile) Read(b []byte) (int, error) {
	return l.reader.Read(b)
}

func (l *LocalAssetFile) Stat() (fs.FileInfo, error) {
	return l, nil
}
func (l *LocalAssetFile) IsDir() bool { return false }
func (l *LocalAssetFile) Name() string {
	filename := l.FileName
	var err error
	if fsys, ok := l.FSys.(NameResolver); ok && !l.isResolved {
		filename, err = fsys.ResolveName(l)
		if err != nil {
			return ""
		}
	}
	return filename
}
func (l *LocalAssetFile) Size() int64 {
	if l.size == 0 {
		filename := l.FileName
		var err error
		if fsys, ok := l.FSys.(NameResolver); ok && !l.isResolved {
			filename, err = fsys.ResolveName(l)
			if err != nil {
				return 0
			}
		}
		s, err := fs.Stat(l.FSys, filename)
		if err != nil {
			return 0
		}
		l.size = int(s.Size())
	}
	return int64(l.size)
}
func (l *LocalAssetFile) Mode() fs.FileMode  { return 0 }
func (l *LocalAssetFile) ModTime() time.Time { return l.dateTaken }
func (l *LocalAssetFile) Sys() any           { return nil }

// Close close the temporary file  and close the source
func (l *LocalAssetFile) Close() error {
	var err error
	if l.sourceFile != nil {
		err = errors.Join(err, l.sourceFile.Close())
		l.sourceFile = nil
	}
	if l.tempFile != nil {
		f := l.tempFile.Name()
		err = errors.Join(err, l.tempFile.Close())
		err = errors.Join(err, os.Remove(f))
		l.tempFile = nil
	}
	return err
}

// Remove the temporary file
func (l *LocalAssetFile) Remove() error {
	if fsys, ok := l.FSys.(Remover); ok {
		return fsys.Remove(l.FileName)
	}
	return nil
}

// DateTakenCached give the previously read date
func (l LocalAssetFile) DateTakenCached() time.Time {
	return l.dateTaken
}

// DateTaken make it best efforts to get the date of capture based on
// - if the name matches a at least 4 digits for the year, 2 for month, 2 for day, in this order.
// It takes the hour, minute, second when present. Very fast
//
// - file content if the file includes some metadata, need read a part of the file
//
//

func (l *LocalAssetFile) DateTaken() (time.Time, error) {
	if !l.dateTaken.IsZero() {
		return l.dateTaken, nil
	}

	l.dateTaken = TakeTimeFromName(l.FileName)
	if !l.dateTaken.IsZero() {
		return l.dateTaken, nil
	}

	ext := strings.ToLower(path.Ext(l.FileName))
	var err error
	switch ext {
	case ".heic", ".heif":
		l.dateTaken, err = l.readHEIFDateTaken()
	case ".jpg", ".jpeg":
		l.dateTaken, err = l.readExifDateTaken()
	case ".mp4", ".mov":
		l.dateTaken, err = l.readMP4DateTaken()
	default:
		err = fmt.Errorf("can't determine the taken date from this file: %q", l.FileName)
	}

	return l.dateTaken, err
}

// readExifDateTaken pase the file for Exif DateTaken
func (l *LocalAssetFile) readExifDateTaken() (time.Time, error) {

	// Open the file
	r, err := l.partialSourceReader()

	if err != nil {
		return time.Time{}, err
	}

	// Decode the EXIF data
	x, err := exif.Decode(r)
	if err != nil && exif.IsCriticalError(err) {
		return time.Time{}, fmt.Errorf("can't get DateTaken: %w", err)
	}

	// Get the date taken from the EXIF data
	tm, err := x.DateTime()
	if err != nil {
		return time.Time{}, fmt.Errorf("can't get DateTaken: %w", err)
	}
	t := time.Date(tm.Year(), tm.Month(), tm.Day(), tm.Hour(), tm.Minute(), tm.Second(), tm.Nanosecond(), time.Local)
	return t, nil
}

// readHEIFDateTaken locate the Exif part and return the date of capture
func (l *LocalAssetFile) readHEIFDateTaken() (time.Time, error) {
	// Open the file
	r, err := l.partialSourceReader()

	if err != nil {
		return time.Time{}, err
	}

	r2, err := seekReaderAtPattern(r, []byte{0x45, 0x78, 0x69, 0x66, 0, 0, 0x4d, 0x4d})
	if err != nil {
		return time.Time{}, err
	}

	filler := make([]byte, 6)
	r2.Read(filler)

	// Decode the EXIF data
	x, err := exif.Decode(r2)
	if err != nil && exif.IsCriticalError(err) {
		return time.Time{}, fmt.Errorf("can't get DateTaken: %w", err)
	}
	// Get the date taken from the EXIF data
	tm, err := x.DateTime()
	if err != nil {
		return time.Time{}, fmt.Errorf("can't get DateTaken: %w", err)
	}
	t := time.Date(tm.Year(), tm.Month(), tm.Day(), tm.Hour(), tm.Minute(), tm.Second(), tm.Nanosecond(), time.Local)
	return t, nil
}

// readMP4DateTaken locate the mvhd atom and decode the date of capture
func (l *LocalAssetFile) readMP4DateTaken() (time.Time, error) {
	// Open the file
	r, err := l.partialSourceReader()

	if err != nil {
		return time.Time{}, err
	}

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

/*
The mvhd atom contains metadata and information about the entire movie or presentation, such as its duration,
time scale, preferred playback rate, and more.

Here are some of the main attributes found in the mvhd atom:

- Timescale: This value indicates the time scale for the media presentation,
   which represents the number of time units per second. It allows for accurate timing of media content in the file.

- Duration: The duration is the total time the movie or presentation lasts,
	expressed in the time scale units defined in the file.

-  Preferred Rate: The preferred rate is the intended playback rate for the movie.
	It can be used to set the default playback speed when the media is played.

- Preferred Volume: The preferred volume specifies the default audio volume for the media playback.

- Matrix Structure: The mvhd atom may contain a matrix structure
		that defines transformations to be applied when rendering the video content, such as scaling or rotation.

-  Creation and Modification Time: The mvhd atom also stores the creation time and modification time
	of the movie or presentation.

In total, the minimum size of the mvhd atom is 108 bytes (version 0) or 112 bytes (version 1).
If any of the optional fields are present, the size of the atom would increase accordingly.
*/

type MvhdAtom struct {
	Marker           []byte //4 bytes
	Version          uint8
	Flags            []byte // 3 bytes
	CreationTime     time.Time
	ModificationTime time.Time
	// ignored fields:
	// Timescale        uint32
	// Duration         uint32
	// Rate             float32
	// Volume           float32
	// Matrix           [9]int32
	// NextTrackID      uint32
}

func decodeMvhdAtom(b []byte) (*MvhdAtom, error) {
	r := &sliceReader{Reader: bytes.NewReader(b)}

	a := &MvhdAtom{}

	// Read the mvhd marker (4 bytes)
	a.Marker, _ = r.ReadSlice(4)

	// Read the mvhd version (1 byte)
	a.Version, _ = r.ReadByte()

	// Read the mvhd flags (3 bytes)
	a.Flags, _ = r.ReadSlice(3)

	if a.Version == 0 {
		// Read the creation time (4 bytes)
		b, _ := r.ReadSlice(4)
		a.ModificationTime = convertTime32(binary.BigEndian.Uint32(b))
		b, _ = r.ReadSlice(4)
		a.CreationTime = convertTime32(binary.BigEndian.Uint32(b))

	} else {
		// Read the creation time (4 bytes)
		b, _ := r.ReadSlice(8)
		a.ModificationTime = convertTime64(binary.BigEndian.Uint64(b))

		b, _ = r.ReadSlice(4)
		a.CreationTime = convertTime64(binary.BigEndian.Uint64(b))
	}

	return a, nil
}

func convertTime32(timestamp uint32) time.Time {
	return time.Unix(int64(timestamp)-int64(2082844800), 0).Local()
}
func convertTime64(timestamp uint64) time.Time {
	// Unix epoch starts on January 1, 1970, subtracting the number of seconds from January 1, 1904 to January 1, 1970.
	epochOffset := int64(2082844800)

	// Convert the creation time to Unix timestamp
	unixTimestamp := int64(timestamp>>32) - epochOffset

	// Convert the Unix timestamp to time.Time
	return time.Unix(unixTimestamp, 0).Local()
}

type sliceReader struct {
	*bytes.Reader
}

func (r *sliceReader) ReadSlice(l int) ([]byte, error) {
	b := make([]byte, l)
	_, err := r.Read(b)
	return b, err
}
