package immichfs

import (
	"context"
	"io"
	"io/fs"
	"time"

	"github.com/simulot/immich-go/immich"
)

/*
Implement the immichfs package let read assets from an immich server

*/

var _ fs.FS = (*ImmichFS)(nil)

type ImmichFS struct {
	ctx    context.Context
	client immich.ImmichInterface
	url    string
}

// NewImmichFS creates a new ImmichFS using the client
func NewImmichFS(ctx context.Context, url string, client immich.ImmichInterface) *ImmichFS {
	return &ImmichFS{
		ctx:    ctx,
		client: client,
		url:    url,
	}
}

var _ fs.File = (*ImmichFile)(nil)

type ImmichFile struct {
	ctx    context.Context
	cancel func(err error)
	info   *fsFileInfo

	rc io.ReadCloser
}

// Open opens the named file for reading.
// name is the ID of the asset
func (ifs *ImmichFS) Open(name string) (fs.File, error) {
	ctx, cancel := context.WithCancelCause(ifs.ctx)

	fi, err := ifs.Stat(name)
	if err != nil {
		cancel(err)
		return nil, err
	}

	rc, err := ifs.client.DownloadAsset(ctx, name)
	if err != nil {
		cancel(err)
		return nil, err
	}
	file := &ImmichFile{
		ctx:    ctx,
		cancel: cancel,
		info:   fi,
		rc:     rc,
	}
	return file, nil
}

func (ifs *ImmichFS) Name() string {
	return ifs.url
}

// Read reads up to len(b) bytes from the file. It returns the number of bytes read and an error, if any.
func (file *ImmichFile) Read(b []byte) (n int, err error) {
	return file.rc.Read(b)
}

// Close closes the file, rendering it unusable for I/O.
func (file *ImmichFile) Close() error {
	if file.rc != nil {
		file.cancel(file.rc.Close())
		file.rc = nil
	}
	return nil
}

// Stat returns a FileInfo describing the file.
// name is the ID of the asset
func (file *ImmichFile) Stat() (fs.FileInfo, error) {
	return file.info, nil
}

// Stat returns a FileInfo describing the file.
// Name is the ID of the asset
func (ifs *ImmichFS) Stat(name string) (*fsFileInfo, error) {
	a, err := ifs.client.GetAssetInfo(ifs.ctx, name)
	if err != nil {
		return nil, err
	}
	return &fsFileInfo{
		name:    a.OriginalFileName,
		size:    a.ExifInfo.FileSizeInByte,
		mode:    fs.FileMode(0o444), // read-only mode
		modTime: a.ExifInfo.DateTimeOriginal.Unix(),
		isDir:   false,
	}, nil
}

var _ fs.FileInfo = (*fsFileInfo)(nil)

type fsFileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime int64
	isDir   bool
}

func (fi *fsFileInfo) Name() string       { return fi.name }
func (fi *fsFileInfo) Size() int64        { return fi.size }
func (fi *fsFileInfo) Mode() fs.FileMode  { return fi.mode }
func (fi *fsFileInfo) ModTime() time.Time { return time.Unix(fi.modTime, 0) }
func (fi *fsFileInfo) IsDir() bool        { return fi.isDir }
func (fi *fsFileInfo) Sys() interface{}   { return nil }
