package fshelper

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/simulot/immich-go/helpers/gen"
)

type pathFS struct {
	dir   string
	files []string
}

func newPathFS(dir string, files []string) (*pathFS, error) {
	_, err := os.Stat(dir)
	if err != nil {
		return nil, err
	}
	return &pathFS{
		dir:   dir,
		files: files,
	}, nil
}

func (fsys pathFS) listed(name string) bool {
	if len(fsys.files) > 0 {
		ext := path.Ext(name)
		if ext == strings.ToLower(".xmp") {
			name = strings.TrimSuffix(name, ext)
		}
		return slices.Contains(fsys.files, name)
	}
	return true
}

func (fsys pathFS) Open(name string) (fs.File, error) {
	if !fsys.listed(name) {
		return nil, fs.ErrNotExist
	}
	return os.Open(filepath.Join(fsys.dir, name))
}

func (fsys pathFS) Stat(name string) (fs.FileInfo, error) {
	if name == "." {
		return os.Stat(fsys.dir)
	}
	if !fsys.listed(name) {
		return nil, fs.ErrNotExist
	}
	return os.Stat(filepath.Join(fsys.dir, name))
}

func (fsys pathFS) ReadDir(name string) ([]fs.DirEntry, error) {
	d, err := os.ReadDir(filepath.Join(fsys.dir, name))

	if err == nil && len(fsys.files) > 0 {
		d = gen.Filter(d, func(i fs.DirEntry) bool {
			return fsys.listed(i.Name())
		})
	}
	return d, err
}
