package nextcloud

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	pathpkg "path"
	"strings"
	"sync"
)

type davReadClient interface {
	ReadDir(path string) ([]os.FileInfo, error)
	ReadStream(path string) (io.ReadCloser, error)
	Stat(path string) (os.FileInfo, error)
}

// WebDAVFS exposes a read-only fs.FS view over the authenticated Nextcloud DAV
// files tree for a single user.
type WebDAVFS struct {
	ctx      context.Context
	client   davReadClient
	rootPath string
	name     string
}

const davOpenOp = "open"

var (
	_ fs.FS        = (*WebDAVFS)(nil)
	_ fs.ReadDirFS = (*WebDAVFS)(nil)
	_ fs.StatFS    = (*WebDAVFS)(nil)
)

// NewWebDAVFS creates a read-only filesystem rooted at /files/{uid}.
func NewWebDAVFS(ctx context.Context, client *Client, uid string) (*WebDAVFS, error) {
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return nil, errors.New("missing Nextcloud user ID for DAV browsing")
	}
	if ctx == nil {
		return nil, errors.New("nil context passed to NewWebDAVFS")
	}
	if client == nil {
		return nil, errors.New("nil Nextcloud client passed to NewWebDAVFS")
	}
	return newWebDAVFS(ctx, client.DAV(), pathpkg.Join("/files", uid), "nextcloud:"+uid), nil
}

func newWebDAVFS(ctx context.Context, client davReadClient, rootPath string, name string) *WebDAVFS {
	return &WebDAVFS{
		ctx:      ctx,
		client:   client,
		rootPath: pathpkg.Clean(rootPath),
		name:     name,
	}
}

func (wfs *WebDAVFS) Name() string {
	return wfs.name
}

func (wfs *WebDAVFS) Open(name string) (fs.File, error) {
	davPath, cleanedName, err := wfs.resolveDAVPath(name)
	if err != nil {
		return nil, &fs.PathError{Op: davOpenOp, Path: name, Err: err}
	}

	info, err := wfs.client.Stat(davPath)
	if err != nil {
		return nil, &fs.PathError{Op: davOpenOp, Path: cleanedName, Err: err}
	}
	if info.IsDir() {
		return &webdavDir{fsys: wfs, name: cleanedName, info: info}, nil
	}

	stream, err := wfs.client.ReadStream(davPath)
	if err != nil {
		return nil, &fs.PathError{Op: davOpenOp, Path: cleanedName, Err: err}
	}
	return &webdavFile{ReadCloser: newCancelableReadCloser(wfs.ctx, stream), info: info}, nil
}

func (wfs *WebDAVFS) Stat(name string) (fs.FileInfo, error) {
	davPath, cleanedName, err := wfs.resolveDAVPath(name)
	if err != nil {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: err}
	}

	info, err := wfs.client.Stat(davPath)
	if err != nil {
		return nil, &fs.PathError{Op: "stat", Path: cleanedName, Err: err}
	}
	return info, nil
}

func (wfs *WebDAVFS) ReadDir(name string) ([]fs.DirEntry, error) {
	davPath, cleanedName, err := wfs.resolveDAVPath(name)
	if err != nil {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: err}
	}

	entries, err := wfs.client.ReadDir(davPath)
	if err != nil {
		return nil, &fs.PathError{Op: "readdir", Path: cleanedName, Err: err}
	}

	dirEntries := make([]fs.DirEntry, 0, len(entries))
	for _, entry := range entries {
		dirEntries = append(dirEntries, fs.FileInfoToDirEntry(entry))
	}
	return dirEntries, nil
}

func (wfs *WebDAVFS) resolveDAVPath(name string) (string, string, error) {
	cleanedName, err := cleanRelativePath(name)
	if err != nil {
		return "", "", err
	}
	if cleanedName == "." {
		return wfs.rootPath, cleanedName, nil
	}
	return pathpkg.Join(wfs.rootPath, cleanedName), cleanedName, nil
}

func cleanRelativePath(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || name == "." {
		return ".", nil
	}
	name = strings.TrimPrefix(name, "/")
	cleaned := pathpkg.Clean(name)
	if cleaned == "." {
		return ".", nil
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fs.ErrInvalid
	}
	return cleaned, nil
}

type webdavFile struct {
	io.ReadCloser
	info fs.FileInfo
}

func (wf *webdavFile) Stat() (fs.FileInfo, error) {
	return wf.info, nil
}

type webdavDir struct {
	fsys    *WebDAVFS
	name    string
	info    fs.FileInfo
	entries []fs.DirEntry
	offset  int
}

func (wd *webdavDir) Stat() (fs.FileInfo, error) {
	return wd.info, nil
}

func (wd *webdavDir) Read(_ []byte) (int, error) {
	return 0, io.EOF
}

func (wd *webdavDir) Close() error {
	return nil
}

func (wd *webdavDir) ReadDir(n int) ([]fs.DirEntry, error) {
	if wd.entries == nil {
		entries, err := wd.fsys.ReadDir(wd.name)
		if err != nil {
			return nil, err
		}
		wd.entries = entries
	}

	if wd.offset >= len(wd.entries) && n > 0 {
		return nil, io.EOF
	}
	if n <= 0 || wd.offset+n > len(wd.entries) {
		n = len(wd.entries) - wd.offset
	}
	chunk := wd.entries[wd.offset : wd.offset+n]
	wd.offset += n
	return chunk, nil
}

type cancelableReadCloser struct {
	ctx   context.Context
	rc    io.ReadCloser
	done  chan struct{}
	close sync.Once
}

func newCancelableReadCloser(ctx context.Context, rc io.ReadCloser) io.ReadCloser {
	if ctx == nil {
		return rc
	}
	c := &cancelableReadCloser{
		ctx:  ctx,
		rc:   rc,
		done: make(chan struct{}),
	}
	go func() {
		select {
		case <-ctx.Done():
			_ = c.Close()
		case <-c.done:
		}
	}()
	return c
}

func (c *cancelableReadCloser) Read(p []byte) (int, error) {
	select {
	case <-c.ctx.Done():
		return 0, c.ctx.Err()
	default:
	}
	return c.rc.Read(p)
}

func (c *cancelableReadCloser) Close() error {
	var err error
	c.close.Do(func() {
		close(c.done)
		err = c.rc.Close()
	})
	return err
}
