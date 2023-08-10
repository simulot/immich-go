package fsys

import (
	"archive/zip"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

/*
	Some helps with file systems

*/

// OpenFSs open all paths and return a merged file system
// allowing
func OpenFSs(names ...string) (fs.FS, error) {
	fss := []fs.FS{}

	for _, p := range names {
		s, err := os.Stat(p)
		if err != nil {
			return nil, err
		}

		switch {
		case !s.IsDir() && strings.ToLower(filepath.Ext(s.Name())) == ".zip":
			fsys, err := zip.OpenReader(p)
			if err != nil {
				return nil, err
			}
			fss = append(fss, fsys)
		default:
			fsys := DirRemoveFS(p)
			fss = append(fss, fsys)
		}
	}
	return NewMergedFS(fss), nil
}

type MergedFS struct {
	fss []fs.FS
}

func NewMergedFS(fss []fs.FS) *MergedFS {
	return &MergedFS{fss: fss}
}
func (mfs *MergedFS) Open(name string) (file fs.File, err error) {
	for i := range mfs.fss {
		file, err = mfs.fss[i].Open(name)
		if err == nil {
			return
		}
	}
	return
}

func (mfs *MergedFS) Stat(name string) (info fs.FileInfo, err error) {
	for i := range mfs.fss {
		info, err = fs.Stat(mfs.fss[i], name)
		if err == nil {
			return
		}
	}
	return
}

func (mfs *MergedFS) Remove(name string) (err error) {
	for i := range mfs.fss {
		_, err = fs.Stat(mfs.fss[i], name)
		if err == nil {
			return Remove(mfs.fss[i], name)
		}
	}
	return

}

type Remover interface {
	Remove(name string) error
}

func Remove(fsys fs.FS, name string) error {
	if fsys, ok := fsys.(Remover); ok {
		return fsys.Remove(name)
	}
	return nil
}

type dirRemoveFS struct {
	dir string
	fs.FS
}

func DirRemoveFS(name string) fs.FS {

	fsys := &dirRemoveFS{
		FS:  os.DirFS(name),
		dir: name,
	}

	return fsys
}
func (fsys dirRemoveFS) Remove(name string) error {
	return os.Remove(filepath.Join(fsys.dir, name))
}
