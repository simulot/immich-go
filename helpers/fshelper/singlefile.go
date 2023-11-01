package fshelper

import (
	"io/fs"
	"os"
	"time"
)

type singleFileFS struct {
	name string
	info fs.FileInfo
}

func newSingleFileFS(name string) (*singleFileFS, error) {
	info, err := os.Stat(name)
	if err != nil {
		return nil, err
	}
	return &singleFileFS{
		name: name,
		info: info,
	}, nil
}

func (sfs singleFileFS) Open(name string) (fs.File, error) {
	if name != sfs.info.Name() {
		return nil, fs.ErrNotExist
	}

	return os.Open(sfs.name)
}

func (sfs singleFileFS) Stat(name string) (fs.FileInfo, error) {
	switch name {
	case ".":
		return fileinfo{
			_name:    ".",
			_size:    0,
			_mode:    sfs.info.Mode(),
			_modTime: sfs.info.ModTime(),
			_isDir:   true,
			_sys:     nil,
		}, nil
	case sfs.info.Name():
		return sfs.info, nil
	}
	return nil, fs.ErrNotExist
}

func (sfs singleFileFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return []fs.DirEntry{
		fileinfo{
			_name:    sfs.info.Name(),
			_size:    sfs.info.Size(),
			_mode:    sfs.info.Mode(),
			_modTime: sfs.info.ModTime(),
			_isDir:   false,
			_sys:     nil,
		},
	}, nil
}

type fileinfo struct {
	_name    string
	_size    int64
	_mode    fs.FileMode
	_modTime time.Time
	_isDir   bool
	_sys     any
}

func (fi fileinfo) Name() string               { return fi._name }
func (fi fileinfo) Size() int64                { return fi._size }
func (fi fileinfo) Mode() fs.FileMode          { return fi._mode }
func (fi fileinfo) ModTime() time.Time         { return fi._modTime }
func (fi fileinfo) IsDir() bool                { return fi._isDir }
func (fi fileinfo) Sys() any                   { return fi._sys }
func (fi fileinfo) Type() fs.FileMode          { return fi._mode }
func (fi fileinfo) Info() (fs.FileInfo, error) { return fi, nil }
