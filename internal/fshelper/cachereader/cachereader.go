package cachereader

import (
	"io"
	"os"
	"path/filepath"
)

// CacheReader is a reader that caches the data in a temporary file to allow multiple reads
type CacheReader struct {
	tmpFile      *os.File // tmpFile is the temporary file or the original file
	shouldRemove bool
}

// NewCacheReader creates a new CacheReader from an io.ReadCloser
// When the reader is an os.File, it will be used directly
// Otherwise, the content will be copied into a temporary file, and the original reader will be closed
func NewCacheReader(rc io.ReadCloser) (*CacheReader, error) {
	var err error
	c := &CacheReader{}
	if f, ok := rc.(*os.File); ok {
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
		// be sure to copy the reader content into the temporary file
		_, err = io.Copy(c.tmpFile, rc)
		if err != nil {
			name := c.tmpFile.Name()
			c.tmpFile.Close()
			_ = os.Remove(name)
			return nil, err
		}
		rc.Close()
		c.shouldRemove = true
	}
	return c, err
}

// OpenFile creates a new file handler based on the temporary file
func (cr *CacheReader) OpenFile() (*os.File, error) {
	f, err := os.Open(cr.tmpFile.Name())
	if err != nil {
		return nil, err
	}
	return f, nil
}

// Close closes the temporary file only if it was created by NewCacheReader
func (cr *CacheReader) Close() error {
	if cr.shouldRemove {
		// the source is already closed
		return os.Remove(cr.tmpFile.Name())
	} else {
		return cr.tmpFile.Close()
	}
}
