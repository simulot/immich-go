package fshelper

import (
	"errors"
	"io"
	"io/fs"
)

type FSCanWrite interface {
	OpenFile(name string, flag int, perm fs.FileMode) (fs.File, error)
	Mkdir(name string, perm fs.FileMode) error
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

func OpenFile(fsys fs.FS, name string, flag int, perm fs.FileMode) (fs.File, error) {
	if fsys, ok := fsys.(FSCanWrite); ok {
		return fsys.OpenFile(name, flag, perm)
	}
	return nil, errors.New("OpenFile not supported")
}

func Mkdir(fsys fs.FS, name string, perm fs.FileMode) error {
	if fsys, ok := fsys.(FSCanWrite); ok {
		return fsys.Mkdir(name, perm)
	}
	return errors.New("Mkdir not supported")
}

func Remove(fsys fs.FS, name string) error {
	if fsys, ok := fsys.(FSCanRemove); ok {
		return fsys.Remove(name)
	}
	return errors.New("Remove not supported")
}

func Stat(fsys fs.FS, name string) (fs.FileInfo, error) {
	if fsys, ok := fsys.(FSCanStat); ok {
		return fsys.Stat(name)
	}
	return nil, errors.New("Stat not supported")
}

func Lstat(fsys fs.FS, name string) (fs.FileInfo, error) {
	if fsys, ok := fsys.(FSCanLink); ok {
		return fsys.Lstat(name)
	}
	return nil, errors.New("Lstat not supported")
}

func Readlink(fsys fs.FS, name string) (string, error) {
	if fsys, ok := fsys.(FSCanLink); ok {
		return fsys.Readlink(name)
	}
	return "", errors.New("Readlink not supported")
}

func WriteFile(fsys fs.FS, name string, r io.Reader) error {
	if fsys, ok := fsys.(FSCanWrite); ok {
		f, err := fsys.OpenFile(name, 0, 0)
		if err != nil {
			return err
		}
		defer f.Close()
		if f, ok := f.(FileCanWrite); ok {
			_, err = io.Copy(f, r)
			return err
		}
	}
	return errors.New("Write not supported")
}
