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
func (a *Asset) Remove() error {
	if fsys, ok := a.File.FS().(fshelper.FSCanRemove); ok {
		return fsys.Remove(a.File.Name())
	}
	return nil
}

func (a *Asset) DeviceAssetID() string {
	return fmt.Sprintf("%s-%d", a.OriginalFileName, a.FileSize)
}

// PartialSourceReader open a reader on the current asset.
// each byte read from it is saved into a temporary file.
//
// It returns a TeeReader that writes each read byte from the sou²rce into the temporary file.
// The temporary file is discarded when the LocalAssetFile is closed
// TODO: possible optimization: when the file is a plain file, do not copy it into a temporary file
// TODO: use user temp folder

func (a *Asset) PartialSourceReader() (reader io.Reader, tmpName string, err error) {
	if a.sourceFile == nil {
		a.sourceFile, err = a.File.Open()
		if err != nil {
			return nil, "", err
		}
	}
	if a.tempFile == nil {
		a.tempFile, err = os.CreateTemp("", "immich-go_*"+a.NameInfo.Ext)
		if err != nil {
			return nil, "", err
		}
		if a.teeReader == nil {
			a.teeReader = io.TeeReader(a.sourceFile, a.tempFile)
		}
	}
	_, err = a.tempFile.Seek(0, 0)
	if err != nil {
		return nil, "", err
	}
	return io.MultiReader(a.tempFile, a.teeReader), a.tempFile.Name(), nil
}

// Open return fs.File that reads previously read bytes followed by the actual file content.
func (a *Asset) Open() (fs.File, error) {
	var err error
	if a.sourceFile == nil {
		a.sourceFile, err = a.File.Open()
		if err != nil {
			return nil, err
		}
	}
	if a.tempFile != nil {
		_, err = a.tempFile.Seek(0, 0)
		if err != nil {
			return nil, err
		}
		a.reader = io.MultiReader(a.tempFile, a.sourceFile)
	} else {
		a.reader = a.sourceFile
	}
	return a, nil
}

// Read
func (a *Asset) Read(b []byte) (int, error) {
	return a.reader.Read(b)
}

// Close close the temporary file  and close the source
func (a *Asset) Close() error {
	var err error
	if a.sourceFile != nil {
		err = errors.Join(err, a.sourceFile.Close())
		a.sourceFile = nil
	}
	if a.tempFile != nil {
		f := a.tempFile.Name()
		err = errors.Join(err, a.tempFile.Close())
		err = errors.Join(err, os.Remove(f))
		a.tempFile = nil
	}
	return err
}

// Stat implements the fs.FILE interface
func (a *Asset) Stat() (fs.FileInfo, error) {
	return a, nil
}
func (a *Asset) IsDir() bool { return false }

func (a *Asset) Name() string {
	return a.File.Name()
}

func (a *Asset) Size() int64 {
	return int64(a.FileSize)
}

// Mode Implements the fs.FILE interface
func (a *Asset) Mode() fs.FileMode { return 0 }

// ModTime implements the fs.FILE interface
func (a *Asset) ModTime() time.Time {
	s, err := a.File.Stat()
	if err != nil {
		return time.Time{}
	}
	return s.ModTime()
}

// Sys implements the fs.FILE interface
func (a *Asset) Sys() any { return nil }
