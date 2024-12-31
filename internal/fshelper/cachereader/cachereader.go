package cachereader

import (
	"io"
	"os"
)

// CacheReader is a reader that caches the data in a temporary file to allow multiple reads
// If the reader passed to NewCacheReader is an osFile, it will be used directly
type CacheReader struct {
	tmpFile *os.File // tmpFile is the temporary file or the original file
}

// NewCacheReader creates a new CacheReader from an io.Reader
func NewCacheReader(r io.Reader) (*CacheReader, error) {
	var err error
	c := &CacheReader{}
	if f, ok := r.(*os.File); ok {
		c.tmpFile = f
	} else {
		c.tmpFile, err = os.CreateTemp("", "immich-go_*")
		if err != nil {
			return nil, err
		}
		// be sure to copy the reader content into the temporary file
		_, err = io.Copy(c.tmpFile, r)
	}
	return c, err
}

// NewReaderAt creates a new readerAt from the temporary file
func (cr *CacheReader) NewReaderAt() (*os.File, error) {
	f, err := os.Open(cr.tmpFile.Name())
	if err != nil {
		return nil, err
	}
	return f, nil
}

// Close closes the temporary file
func (cr *CacheReader) Close() error {
	os.Remove(cr.tmpFile.Name())
	return cr.tmpFile.Close()
}
