package fshelper

import (
	"io/fs"
	"log/slog"
)

type FSAndName struct {
	fsys fs.FS
	name string
}

func FSName(fsys fs.FS, name string) FSAndName {
	return FSAndName{fsys: fsys, name: name}
}

func (fn FSAndName) LogValue() slog.Value {
	return slog.StringValue(fn.FullName())
}

func (fn FSAndName) FS() fs.FS {
	return fn.fsys
}

func (fn FSAndName) Name() string {
	return fn.name
}

func (fn FSAndName) FullName() string {
	fsys := fn.fsys
	if fsys, ok := fsys.(NameFS); ok {
		return fsys.Name() + ":" + fn.name
	}
	return fn.name
}

func (fn FSAndName) Open() (fs.File, error) {
	return fn.fsys.Open(fn.name)
}

func (fn FSAndName) Stat() (fs.FileInfo, error) {
	return fs.Stat(fn.fsys, fn.name)
}
