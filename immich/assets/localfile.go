package assets

import (
	"errors"
	"immich-go/fshelper"
	"immich-go/immich/metadata"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"
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
	FileName string   // The asset's path in the fsys
	Title    string   // Google Photos may a have title longer than the filename
	Album    []string // The asset's album, if any
	Err      error    // keep errors encountered
	SideCar  *metadata.SideCar

	// Common metadata
	DateTaken time.Time // the date of capture
	Latitude  float64   // GPS Latitude
	Longitude float64   // GPS Longitude
	Altitude  float64   // GPS Altitude

	// Google Photos flags
	Trashed     bool // The asset is trashed
	Archived    bool // The asset is archived
	FromPartner bool // the asset comes from a partner

	FSys fs.FS // Asset's file system

	// Unexported fields
	isResolved bool // True when the FileName is resolved, for google photos
	fInfo      fs.FileInfo
	size       int // Accessible via Stat()

	// dateTaken       time.Time // Accessible via DateTaken()
	// dateAlreadyRead bool      // true when the date has been read

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

func (l *LocalAssetFile) resolve() error {
	var err error
	if l.isResolved {
		return nil
	}
	if fsys, ok := l.FSys.(NameResolver); ok && !l.isResolved {
		l.FileName, err = fsys.ResolveName(l)
		if err != nil {
			return err
		}
	}
	l.fInfo, err = fs.Stat(l.FSys, l.FileName)
	l.isResolved = true
	return nil
}

func (l *LocalAssetFile) Name() string {
	l.resolve()
	return l.FileName
}
func (l *LocalAssetFile) Size() int64 {
	l.resolve()
	return l.fInfo.Size()
}
func (l *LocalAssetFile) Mode() fs.FileMode { return 0 }
func (l *LocalAssetFile) ModTime() time.Time {
	l.resolve()
	return l.fInfo.ModTime()
}
func (l *LocalAssetFile) Sys() any { return nil }

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

func (l *LocalAssetFile) AddAlbum(album string) {
	for _, al := range l.Album {
		if al == album {
			return
		}
	}
	l.Album = append(l.Album, album)
}

func (l *LocalAssetFile) IsInAlbum(album string) bool {
	for _, al := range l.Album {
		if al == album {
			return true
		}
	}
	return false
}

// Remove the temporary file
func (l *LocalAssetFile) Remove() error {
	if fsys, ok := l.FSys.(fshelper.Remover); ok {
		return fsys.Remove(l.FileName)
	}
	return nil
}
