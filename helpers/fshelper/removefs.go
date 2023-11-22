package fshelper

import (
	"io/fs"
	"os"
	"path/filepath"
)

/*
	Some helps with file systems

*/

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

func (fsys dirRemoveFS) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(filepath.Join(fsys.dir, name))
}
