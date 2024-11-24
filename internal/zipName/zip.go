package zipname

import (
	stdZip "archive/zip"
	"path/filepath"
	"strings"
)

type ZipReadCloser struct {
	*stdZip.ReadCloser
	name string
}

func OpenReader(f string) (*ZipReadCloser, error) {
	z, err := stdZip.OpenReader(f)
	if err != nil {
		return nil, err
	}
	name := filepath.Base(f)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	return &ZipReadCloser{
		ReadCloser: z,
		name:       name,
	}, nil
}

func (z ZipReadCloser) Name() string {
	return z.name
}
