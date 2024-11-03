package assets

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"time"

	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/simulot/immich-go/internal/metadata"
)

/*
	Asset structure hold information on assets used for building immich assets.

	The asset is taken into a fs.FS system which doesn't implement anything else than a strait
	reader.
	fsys can be a zip file, a DirFS, or anything else.

	It implements a way to read a minimal quantity of data to be able to take a decision
	about chose a file or discard it.

	implements fs.File and fs.FileInfo, Stat

*/

type Asset struct {
	// File system and file name
	FSys     fs.FS     // Asset's file system
	FileName string    // The asset's path in the fsys
	FileDate time.Time // File creation date

	// Common fields
	Title    string // Google Photos may a have title longer than the filename
	FileSize int    // File size in bytes
	ID       string // Immich ID after upload

	// Flags that are provided to Immich Upload API call
	CaptureDate time.Time // Date of the capture
	Trashed     bool      // The asset is trashed
	Archived    bool      // The asset is archived
	FromPartner bool      // the asset comes from a partner
	Favorite    bool      // the asset is marked as favorite
	Stars       int       // the asset is marked with stars

	// GPS location
	Latitude  float64 // GPS latitude
	Longitude float64 // GPS longitude

	// When a sidecar is found beside the asset
	SideCar metadata.SideCarFile // sidecar file if found
	Albums  []Album              // List of albums the asset is in

	// Internal fields

	nameInfo filenames.NameInfo

	// buffer management
	sourceFile fs.File   // the opened source file
	tempFile   *os.File  // buffer that keep partial reads available for the full file reading
	teeReader  io.Reader // write each read from it into the tempWriter
	reader     io.Reader // the reader that combines the partial read and original file for full file reading
}

func (l *Asset) SetNameInfo(ni filenames.NameInfo) {
	l.nameInfo = ni
	if l.CaptureDate.IsZero() {
		l.CaptureDate = ni.Taken
	}
}

func (l *Asset) NameInfo() filenames.NameInfo {
	return l.nameInfo
}

func (l *Asset) DateTaken() time.Time {
	return l.CaptureDate
}

// Remove the temporary file
func (l *Asset) Remove() error {
	if fsys, ok := l.FSys.(fshelper.FSCanRemove); ok {
		return fsys.Remove(l.FileName)
	}
	return nil
}

func (l *Asset) DeviceAssetID() string {
	return fmt.Sprintf("%s-%d", l.Title, l.FileSize)
}

// PartialSourceReader open a reader on the current asset.
// each byte read from it is saved into a temporary file.
//
// It returns a TeeReader that writes each read byte from the source into the temporary file.
// The temporary file is discarded when the LocalAssetFile is closed
// TODO: possible optimization: when the file is a plain file, do not copy it into a temporary file
// TODO: use user temp folder

func (l *Asset) PartialSourceReader() (reader io.Reader, err error) {
	if l.sourceFile == nil {
		l.sourceFile, err = l.FSys.Open(l.FileName)
		if err != nil {
			return nil, err
		}
	}
	if l.tempFile == nil {
		l.tempFile, err = os.CreateTemp("", "immich-go_*.tmp")
		if err != nil {
			return nil, err
		}
		if l.teeReader == nil {
			l.teeReader = io.TeeReader(l.sourceFile, l.tempFile)
		}
	}
	_, err = l.tempFile.Seek(0, 0)
	if err != nil {
		return nil, err
	}
	return io.MultiReader(l.tempFile, l.teeReader), nil
}

// Open return fs.File that reads previously read bytes followed by the actual file content.
func (l *Asset) Open() (fs.File, error) {
	var err error
	if l.sourceFile == nil {
		l.sourceFile, err = l.FSys.Open(l.FileName)
		if err != nil {
			return nil, err
		}
	}
	if l.tempFile != nil {
		_, err = l.tempFile.Seek(0, 0)
		if err != nil {
			return nil, err
		}
		l.reader = io.MultiReader(l.tempFile, l.sourceFile)
	} else {
		l.reader = l.sourceFile
	}
	return l, nil
}

// Read
func (l *Asset) Read(b []byte) (int, error) {
	return l.reader.Read(b)
}

// Close close the temporary file  and close the source
func (l *Asset) Close() error {
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

// Stat implements the fs.FILE interface
func (l *Asset) Stat() (fs.FileInfo, error) {
	return l, nil
}
func (l *Asset) IsDir() bool { return false }

func (l *Asset) Name() string {
	return l.FileName
}

func (l *Asset) Size() int64 {
	return int64(l.FileSize)
}

// Mode Implements the fs.FILE interface
func (l *Asset) Mode() fs.FileMode { return 0 }

// ModTime implements the fs.FILE interface
func (l *Asset) ModTime() time.Time {
	s, err := fs.Stat(l.FSys, l.FileName)
	if err != nil {
		return time.Time{}
	}
	return s.ModTime()
}

// Sys implements the fs.FILE interface
func (l *Asset) Sys() any { return nil }

// LogValue returns a slog.Value representing the LocalAssetFile's properties.
func (l Asset) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("FileName", fileevent.AsFileAndName(l.FSys, l.FileName).Name()),
		slog.Time("FileDate", l.FileDate),
		slog.String("Title", l.Title),
		slog.Int("FileSize", l.FileSize),
		slog.String("ID", l.ID),
		slog.Time("CaptureDate", l.CaptureDate),
		slog.Bool("Trashed", l.Trashed),
		slog.Bool("Archived", l.Archived),
		slog.Bool("FromPartner", l.FromPartner),
		slog.Bool("Favorite", l.Favorite),
		slog.Int("Stars", l.Stars),
		slog.Float64("Latitude", l.Latitude),
		slog.Float64("Longitude", l.Longitude),
	)
}
