package assets

import (
	"github.com/simulot/immich-go/internal/fshelper/cachereader"
	"github.com/simulot/immich-go/internal/fshelper/debugfiles"
	"github.com/simulot/immich-go/internal/fshelper/osfs"
)

// func (a *Asset) DeviceAssetID() string {
// 	return fmt.Sprintf("%s-%d", path.Base(a.File.Name()), a.FileSize)
// }

// OpenFile return an os.File whatever the type of source reader is.
// It can be called several times for the same asset.

func (a *Asset) OpenFile() (osfs.OSFS, error) {
	if a.cacheReader == nil {
		// get a FS.File from of the asset
		f, err := a.File.Open()
		if err != nil {
			return nil, err
		}
		debugfiles.TrackOpenFile(f, a.File.FullName())
		// Create a cache reader from the FS.File
		cr, sha1, err := cachereader.NewCacheReader(a.File.FullName(), f)
		if err != nil {
			return nil, err
		}
		a.cacheReader = cr
		if sha1 != "" {
			a.Checksum = sha1
		}
	}
	return a.cacheReader.OpenFile()
}

// Close close the temporary file  and close the source
func (a *Asset) Close() error {
	if a.cacheReader == nil {
		return nil
	}
	return a.cacheReader.Close()
}

/*

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
*/
