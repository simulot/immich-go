package fshelper

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/simulot/immich-go/internal/fshelper/debugfiles"
)

type FSCanWrite interface {
	OpenFile(name string, flag int, perm fs.FileMode) (WFile, error)
	Mkdir(name string, perm fs.FileMode) error
}

type FSCanMkdirAll interface {
	MkdirAll(path string, perm fs.FileMode) error
}
type FSCanRemove interface {
	Remove(name string) error
}

type FSCanStat interface {
	Stat(name string) (fs.FileInfo, error)
}

type FSCanLink interface {
	Lstat(name string) (fs.FileInfo, error)
	Readlink(name string) (string, error)
	MkSymlink(name, target string) error
}

type FileCanWrite interface {
	Write(b []byte) (ret int, err error)
}

type WFile interface {
	fs.File
	Write(b []byte) (ret int, err error)
}

func OpenFile(fsys fs.FS, name string, flag int, perm fs.FileMode) (WFile, error) {
	if fsys, ok := fsys.(FSCanWrite); ok {
		return fsys.OpenFile(name, flag, perm)
	}
	return nil, errors.New("openFile not supported")
}

func Mkdir(fsys fs.FS, name string, perm fs.FileMode) error {
	if fsys, ok := fsys.(FSCanWrite); ok {
		return fsys.Mkdir(name, perm)
	}
	return errors.New("mkdir not supported")
}

func MkdirAll(fsys fs.FS, path string, perm fs.FileMode) error {
	if fsys, ok := fsys.(FSCanMkdirAll); ok {
		return fsys.MkdirAll(path, perm)
	}
	if fsys, ok := fsys.(FSCanWrite); ok {
		parts := strings.Split(path, "/")

		// parts := strings.Split(path, string(filepath.Separator))
		path = ""
		for i := 0; i < len(parts); i++ {
			path = filepath.Join(path, parts[i])
			if err := fsys.Mkdir(path, perm); err != nil && !errors.Is(err, os.ErrExist) {
				return err
			}
		}
		return nil
	} else {
		return errors.New("mkdirAll not supported")
	}
}

func Remove(fsys fs.FS, name string) error {
	if fsys, ok := fsys.(FSCanRemove); ok {
		return fsys.Remove(name)
	}
	return errors.New("remove not supported")
}

func Stat(fsys fs.FS, name string) (fs.FileInfo, error) {
	if fsys, ok := fsys.(FSCanStat); ok {
		return fsys.Stat(name)
	}
	return nil, errors.New("stat not supported")
}

func Lstat(fsys fs.FS, name string) (fs.FileInfo, error) {
	if fsys, ok := fsys.(FSCanLink); ok {
		return fsys.Lstat(name)
	}
	return nil, errors.New("lstat not supported")
}

func Readlink(fsys fs.FS, name string) (string, error) {
	if fsys, ok := fsys.(FSCanLink); ok {
		return fsys.Readlink(name)
	}
	return "", errors.New("readlink not supported")
}

func WriteFile(fsys fs.FS, name string, r io.Reader) error {
	if fsys, ok := fsys.(FSCanWrite); ok {
		f, err := fsys.OpenFile(name, os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		debugfiles.TrackOpenFile(f, name)
		defer f.Close()
		defer debugfiles.TrackCloseFile(f)
		if f, ok := f.(FileCanWrite); ok {
			_, err = io.Copy(f, r)
			return err
		}
	}
	return errors.New("write not supported")
}
