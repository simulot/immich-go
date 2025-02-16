package zipname

import (
	stdZip "archive/zip"
	"os"
	"path/filepath"
	"strings"

	"github.com/simulot/immich-go/internal/fshelper/debugfiles"
)

type ZipReadCloser struct {
	*stdZip.Reader
	f    *os.File
	name string
}

func OpenReader(name string) (*ZipReadCloser, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}

	s, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if s.IsDir() {
		return nil, stdZip.ErrFormat
	}
	z, err := stdZip.NewReader(f, s.Size())
	if err != nil {
		f.Close()
		return nil, err
	}
	debugfiles.TrackOpenFile(f, name)
	name = filepath.Base(name)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	return &ZipReadCloser{
		Reader: z,
		name:   name,
		f:      f,
	}, nil
}

func (z ZipReadCloser) Close() error {
	debugfiles.TrackCloseFile(z.f)
	return z.f.Close()
}

func (z ZipReadCloser) Name() string {
	return z.name
}
