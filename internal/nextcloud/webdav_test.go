package nextcloud

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWebDAVFSRejectsEmptyUID(t *testing.T) {
	t.Parallel()

	client := &Client{}
	_, err := NewWebDAVFS(context.Background(), client, " ")
	require.Error(t, err)
	assert.ErrorContains(t, err, "user ID")
}

func TestWebDAVFSReadDirAndOpen(t *testing.T) {
	t.Parallel()

	rootInfo := fakeFileInfo{name: "alice", dir: true, modTime: time.Unix(1700000000, 0)}
	photosInfo := fakeFileInfo{name: "Photos", dir: true, modTime: time.Unix(1700000001, 0)}
	imageInfo := fakeFileInfo{name: "IMG_0001.JPG", size: 5, modTime: time.Unix(1700000002, 0)}

	dav := &fakeDAVClient{
		stats: map[string]fakeFileInfo{
			"/files/alice":                     rootInfo,
			"/files/alice/Photos":              photosInfo,
			"/files/alice/Photos/IMG_0001.JPG": imageInfo,
		},
		dirs: map[string][]fakeFileInfo{
			"/files/alice":        {photosInfo},
			"/files/alice/Photos": {imageInfo},
		},
		files: map[string]string{
			"/files/alice/Photos/IMG_0001.JPG": "hello",
		},
	}

	fsys := newWebDAVFS(context.Background(), dav, "/files/alice", "nextcloud:alice")

	entries, err := fs.ReadDir(fsys, "Photos")
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "IMG_0001.JPG", entries[0].Name())

	f, err := fsys.Open("Photos/IMG_0001.JPG")
	require.NoError(t, err)
	defer f.Close()

	b, err := io.ReadAll(f)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(b))
}

func TestWebDAVFSRejectsPathEscape(t *testing.T) {
	t.Parallel()

	fsys := newWebDAVFS(context.Background(), &fakeDAVClient{}, "/files/alice", "nextcloud:alice")
	_, err := fsys.Stat("../secrets.txt")
	require.Error(t, err)
	assert.ErrorContains(t, err, "invalid argument")
}

func TestWebDAVFSOpenCancelsActiveRead(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	blocker := newBlockingReadCloser()
	dav := &fakeDAVClient{
		stats: map[string]fakeFileInfo{
			"/files/alice":                  {name: "alice", dir: true},
			"/files/alice/Photos":           {name: "Photos", dir: true},
			"/files/alice/Photos/video.mp4": {name: "video.mp4", size: 42},
		},
		filesReader: map[string]io.ReadCloser{
			"/files/alice/Photos/video.mp4": blocker,
		},
	}

	fsys := newWebDAVFS(ctx, dav, "/files/alice", "nextcloud:alice")
	f, err := fsys.Open("Photos/video.mp4")
	require.NoError(t, err)

	readDone := make(chan error, 1)
	go func() {
		buf := make([]byte, 1)
		_, err := f.Read(buf)
		readDone <- err
	}()

	cancel()

	select {
	case err := <-readDone:
		require.Error(t, err)
		assert.True(t, errors.Is(err, context.Canceled) || errors.Is(err, io.ErrClosedPipe) || errors.Is(err, fs.ErrClosed))
	case <-time.After(2 * time.Second):
		t.Fatal("read did not unblock after context cancellation")
	}
}

type fakeDAVClient struct {
	stats       map[string]fakeFileInfo
	dirs        map[string][]fakeFileInfo
	files       map[string]string
	filesReader map[string]io.ReadCloser
}

func (f *fakeDAVClient) ReadDir(path string) ([]os.FileInfo, error) {
	entries, ok := f.dirs[path]
	if !ok {
		return nil, fs.ErrNotExist
	}
	result := make([]os.FileInfo, 0, len(entries))
	for _, entry := range entries {
		entry := entry
		result = append(result, entry)
	}
	return result, nil
}

func (f *fakeDAVClient) ReadStream(path string) (io.ReadCloser, error) {
	if reader, ok := f.filesReader[path]; ok {
		return reader, nil
	}
	b, ok := f.files[path]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return io.NopCloser(strings.NewReader(b)), nil
}

func (f *fakeDAVClient) Stat(path string) (os.FileInfo, error) {
	info, ok := f.stats[path]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return info, nil
}

type fakeFileInfo struct {
	name    string
	size    int64
	dir     bool
	modTime time.Time
}

func (f fakeFileInfo) Name() string { return f.name }
func (f fakeFileInfo) Size() int64  { return f.size }
func (f fakeFileInfo) Mode() fs.FileMode {
	if f.dir {
		return fs.ModeDir | 0o755
	}
	return 0o644
}
func (f fakeFileInfo) ModTime() time.Time { return f.modTime }
func (f fakeFileInfo) IsDir() bool        { return f.dir }
func (f fakeFileInfo) Sys() any           { return nil }

type blockingReadCloser struct {
	closed chan struct{}
	once   sync.Once
}

func newBlockingReadCloser() *blockingReadCloser {
	return &blockingReadCloser{closed: make(chan struct{})}
}

func (b *blockingReadCloser) Read(_ []byte) (int, error) {
	<-b.closed
	return 0, io.ErrClosedPipe
}

func (b *blockingReadCloser) Close() error {
	b.once.Do(func() {
		close(b.closed)
	})
	return nil
}
