package osfs

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/simulot/immich-go/internal/fshelper/debugfiles"
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
	f, err := os.Open(filepath.Join(string(dir), name))
	if err != nil {
		debugfiles.TrackOpenFile(f, name)
	}
	return f, err
}

func (dir dirFS) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(filepath.Join(string(dir), name))
}

func (dir dirFS) OpenFile(name string, flag int, perm fs.FileMode) (fshelper.WFile, error) {
	f, err := os.OpenFile(filepath.Join(string(dir), name), flag, perm)
	if err != nil {
		debugfiles.TrackOpenFile(f, name)
	}
	return f, err
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

type OSFS interface {
	fs.File
	Name() string
	ReadAt(b []byte, off int64) (n int, err error)
	Seek(offset int64, whence int) (ret int64, err error)
	Stat() (fs.FileInfo, error)
	Write(b []byte) (n int, err error)
	WriteAt(b []byte, off int64) (n int, err error)
}
