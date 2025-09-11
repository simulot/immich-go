package tgzname

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/simulot/immich-go/internal/fshelper/debugfiles"
)

type TgzReadCloser struct {
	f    *os.File
	name string
	zr   *gzip.Reader
	tr   *tar.Reader
}

type fileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
}

func (fi fileInfo) Name() string {
	return fi.name
}
func (fi fileInfo) Size() int64 {
	return fi.size
}
func (fi fileInfo) Mode() fs.FileMode {
	return fi.mode
}
func (fi fileInfo) ModTime() time.Time {
	return fi.modTime
}
func (fi fileInfo) IsDir() bool {
	return fi.mode.IsDir()
}
func (fi fileInfo) Sys() any {
	return nil
}

var (
	_ fs.File     = (*file)(nil)
	_ fs.FileInfo = (*fileInfo)(nil)
)

type file struct {
	io.Reader
	io.Closer
	fi os.FileInfo
}

func (f file) Stat() (fs.FileInfo, error) {
	return f.fi, nil
}

func OpenReader(name string) (*TgzReadCloser, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}

	s, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if s.IsDir() {
		return nil, fs.ErrInvalid
	}
	zr, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}

	tr := tar.NewReader(zr)

	debugfiles.TrackOpenFile(f, name)
	baseName := filepath.Base(name)
	baseName = strings.TrimSuffix(baseName, filepath.Ext(baseName))
	if strings.HasSuffix(strings.ToLower(baseName), ".tar") {
		baseName = strings.TrimSuffix(baseName, ".tar")
	}

	return &TgzReadCloser{
		f:      f,
		name:   baseName,
		zr:     zr,
		tr:     tr,
	}, nil
}

func (z TgzReadCloser) Open(name string) (fs.File, error) {
	err := z.rewind()
	if err != nil {
		return nil, err
	}
	for {
		h, err := z.tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if h.Name == name {
			return file{
				Reader: z.tr,
				Closer: io.NopCloser(nil),
				fi:     h.FileInfo(),
			}, nil
		}
	}
	return nil, fs.ErrNotExist
}

func (z *TgzReadCloser) rewind() error {
	_, err := z.f.Seek(0, 0)
	if err != nil {
		return err
	}
	err = z.zr.Reset(z.f)
	if err != nil {
		return err
	}
	z.tr = tar.NewReader(z.zr)
	return nil
}

func (z TgzReadCloser) Close() error {
	debugfiles.TrackCloseFile(z.f)
	err := z.zr.Close()
	err2 := z.f.Close()
	if err != nil {
		return err
	}
	return err2
}

func (z TgzReadCloser) Name() string {
	return z.name
}
