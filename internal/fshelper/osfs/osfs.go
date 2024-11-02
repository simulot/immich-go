package osfs

import (
	"io/fs"
	"os"
	"path/filepath"
)

/*
  Define a file system that can write, remove, stats,etc...

*/

func DirFS(name string) fs.FS {
	return dirFS(name)
}

type dirFS string

func (dir dirFS) Open(name string) (fs.File, error) {
	return os.Open(filepath.Join(string(dir), name))
}

func (dir dirFS) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(filepath.Join(string(dir), name))
}

func (dir dirFS) OpenFile(name string, flag int, perm fs.FileMode) (fs.File, error) {
	return os.OpenFile(filepath.Join(string(dir), name), flag, perm)
}

func (dir dirFS) Mkdir(name string, perm fs.FileMode) error {
	return os.Mkdir(filepath.Join(string(dir), name), perm)
}

func (dir dirFS) Readlink(name string) (string, error) {
	return os.Readlink(filepath.Join(string(dir), name))
}

func (dir dirFS) Lstat(name string) (fs.FileInfo, error) {
	return os.Lstat(filepath.Join(string(dir), name))
}

func (dir dirFS) MkSymlink(name, target string) error {
	return os.Symlink(filepath.Join(string(dir), name))
}

func (dir dirFS) Remove(name, target string) error {
	return os.Remove(filepath.Join(string(dir), name))
}
