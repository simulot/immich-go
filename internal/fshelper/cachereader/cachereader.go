package cachereader

import (
	"io"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/simulot/immich-go/internal/fshelper/osfs"
	"github.com/simulot/immich-go/internal/loghelper"
)

// CacheReader is a reader that caches the data in a temporary file to allow multiple reads
type CacheReader struct {
	tmpFile      osfs.OSFS //*os.File // tmpFile is the temporary file or the original file
	name         string
	shouldRemove bool
	references   int64
}

// NewCacheReader creates a new CacheReader from an io.ReadCloser
// When the reader is an os.File, it will be used directly
// Otherwise, the content will be copied into a temporary file, and the original reader will be closed
func NewCacheReader(name string, rc io.ReadCloser) (*CacheReader, error) {
	var err error
	c := &CacheReader{}
	if f, ok := rc.(osfs.OSFS); ok {
		c.name = f.Name()
		c.tmpFile = f
	} else {
		d := os.Getenv("IMMICHGO_TEMPDIR")
		if d == "" {
			d, err = os.UserCacheDir()
			if err != nil {
				d = os.TempDir()
			}
		}
		d = filepath.Join(d, "immich-go", "temp")

		err = os.MkdirAll(d, 0o700)
		if err != nil {
			d = os.TempDir()
		}
		c.tmpFile, err = os.CreateTemp(d, "immich-go_*")
		if err != nil {
			return nil, err
		}
		c.name = c.tmpFile.Name()
		// be sure to copy the reader content into the temporary file
		_, err = io.Copy(c.tmpFile, rc)
		if err != nil {
			c.tmpFile.Close()
			_ = os.Remove(c.name)
			return nil, err
		}
		rc.Close()
		loghelper.Debug("CacheReader: create temporary file", "Source file", name, "temp file", c.name)
		c.shouldRemove = true
	}
	return c, err
}

// OpenFile creates a new file handler based on the temporary file
func (cr *CacheReader) OpenFile() (*tempFile, error) {
	refs := atomic.AddInt64(&cr.references, 1)
	loghelper.Debug("tempFile:", "Open file", cr.name, "references", refs)
	f, err := os.Open(cr.name)
	if err != nil {
		return nil, err
	}
	return &tempFile{File: f, cr: cr}, nil
}

// Close closes the temporary file only if it was created by NewCacheReader
func (cr *CacheReader) Close() error {
	refs := atomic.LoadInt64(&cr.references)
	loghelper.Debug("CacheReader: close", "name", cr.name, "references", refs)
	if cr.shouldRemove {
		// the source is already closed
		loghelper.Debug("CacheReader: remove temporary file", "name", cr.name)
		return os.Remove(cr.name)
	} else {
		return cr.tmpFile.Close()
	}
}

type tempFile struct {
	*os.File
	cr *CacheReader
}

func (t *tempFile) Close() error {
	refs := atomic.AddInt64(&t.cr.references, -1)
	loghelper.Debug("tempFile:", "assetName", t.cr.name, "close file", t.File.Name(), "references", refs+1)
	if refs < 0 {
		panic("tempFile: Close() called on a closed file")
	}
	err := t.File.Close()
	return err
}
