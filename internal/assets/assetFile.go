package assets

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"time"

	"github.com/simulot/immich-go/internal/fshelper"
)

// Remove the temporary file
func (l *Asset) Remove() error {
	if fsys, ok := l.File.FS().(fshelper.FSCanRemove); ok {
		return fsys.Remove(l.File.Name())
	}
	return nil
}

func (l *Asset) DeviceAssetID() string {
	return fmt.Sprintf("%s-%d", l.OriginalFileName, l.FileSize)
}

// PartialSourceReader open a reader on the current asset.
// each byte read from it is saved into a temporary file.
//
// It returns a TeeReader that writes each read byte from the sou²rce into the temporary file.
// The temporary file is discarded when the LocalAssetFile is closed
// TODO: possible optimization: when the file is a plain file, do not copy it into a temporary file
// TODO: use user temp folder

func (l *Asset) PartialSourceReader() (reader io.Reader, err error) {
	if l.sourceFile == nil {
		l.sourceFile, err = l.File.Open()
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
		l.sourceFile, err = l.File.Open()
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
	return l.File.Name()
}

func (l *Asset) Size() int64 {
	return int64(l.FileSize)
}

// Mode Implements the fs.FILE interface
func (l *Asset) Mode() fs.FileMode { return 0 }

// ModTime implements the fs.FILE interface
func (l *Asset) ModTime() time.Time {
	s, err := l.File.Stat()
	if err != nil {
		return time.Time{}
	}
	return s.ModTime()
}

// Sys implements the fs.FILE interface
func (l *Asset) Sys() any { return nil }
