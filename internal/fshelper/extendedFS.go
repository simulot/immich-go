package fshelper

import (
	"errors"
	"io/fs"
)

type CanWrite interface {
	OpenFile(name string, flag int, perm fs.FileMode) (fs.File, error)
	Mkdir(name string, perm fs.FileMode) error
	Remove(name string) error
}

type CanStat interface {
	Stat(name string) (fs.FileInfo, error)
}

type CanLink interface {
	Lstat(name string) (fs.FileInfo, error)
	Readlink(name string) (string, error)
	MkSymlink(name, target string) error
}

func OpenFile(fsys fs.FS, name string, flag int, perm fs.FileMode) (fs.File, error) {
	if fsys, ok := fsys.(CanWrite); ok {
		return fsys.OpenFile(name, flag, perm)
	}
	return nil, errors.New("OpenFile not supported")
}

func Mkdir(fsys fs.FS, name string, perm fs.FileMode) error {
	if fsys, ok := fsys.(CanWrite); ok {
		return fsys.Mkdir(name, perm)
	}
	return errors.New("Mkdir not supported")
}

func Remove(fsys fs.FS, name string) error {
	if fsys, ok := fsys.(CanWrite); ok {
		return fsys.Remove(name)
	}
	return errors.New("Remove not supported")
}

func Stat(fsys fs.FS, name string) (fs.FileInfo, error) {
	if fsys, ok := fsys.(CanStat); ok {
		return fsys.Stat(name)
	}
	return nil, errors.New("Stat not supported")
}

func Lstat(fsys fs.FS, name string) (fs.FileInfo, error) {
	if fsys, ok := fsys.(CanLink); ok {
		return fsys.Lstat(name)
	}
	return nil, errors.New("Lstat not supported")
}

func Readlink(fsys fs.FS, name string) (string, error) {
	if fsys, ok := fsys.(CanLink); ok {
		return fsys.Readlink(name)
	}
	return "", errors.New("Readlink not supported")
}
