package osfs

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/simulot/immich-go/internal/fshelper"
)

/*
  Define a file system that can write, remove, stats,etc...
*/

func DirFS(name string) fs.FS {
	return dirFS(name)
}

// check that dirFS implements the interfaces
var (
	_ fshelper.FSCanWrite = dirFS("")
	// _ fshelper.FSCanMkdirAll = dirFS("")
	_ fshelper.FSCanRemove = dirFS("")
	_ fshelper.FSCanStat   = dirFS("")
	_ fshelper.FSCanLink   = dirFS("")
)

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
	return os.Symlink(filepath.Join(string(dir), name), filepath.Join(string(dir), target))
}

func (dir dirFS) Remove(name string) error {
	return os.Remove(filepath.Join(string(dir), name))
}
