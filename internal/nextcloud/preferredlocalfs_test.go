package nextcloud

import (
	"io"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreferredLocalFSOpenPrefersLocalFile(t *testing.T) {
	t.Parallel()

	local := fstest.MapFS{
		"Photos/IMG_0001.JPG": {Data: []byte("local")},
	}
	remote := fstest.MapFS{
		"Photos/IMG_0001.JPG": {Data: []byte("remote")},
	}

	fsys := NewPreferredLocalFS(local, remote)
	f, err := fsys.Open("Photos/IMG_0001.JPG")
	require.NoError(t, err)
	defer f.Close()

	b, err := io.ReadAll(f)
	require.NoError(t, err)
	assert.Equal(t, "local", string(b))
}

func TestPreferredLocalFSFallsBackToRemoteOpen(t *testing.T) {
	t.Parallel()

	remote := fstest.MapFS{
		"Photos/IMG_0001.JPG": {Data: []byte("remote")},
	}

	fsys := NewPreferredLocalFS(fstest.MapFS{}, remote)
	f, err := fsys.Open("Photos/IMG_0001.JPG")
	require.NoError(t, err)
	defer f.Close()

	b, err := io.ReadAll(f)
	require.NoError(t, err)
	assert.Equal(t, "remote", string(b))
}

func TestPreferredLocalFSDelegatesEnumerationToRemote(t *testing.T) {
	t.Parallel()

	remote := fstest.MapFS{
		"Photos/Trips/IMG_0001.JPG": {Data: []byte("remote")},
	}

	fsys := NewPreferredLocalFS(fstest.MapFS{}, remote)
	dirEntries, err := fs.ReadDir(fsys, "Photos/Trips")
	require.NoError(t, err)
	require.Len(t, dirEntries, 1)
	assert.Equal(t, "IMG_0001.JPG", dirEntries[0].Name())

	info, err := fs.Stat(fsys, "Photos/Trips/IMG_0001.JPG")
	require.NoError(t, err)
	assert.Equal(t, int64(6), info.Size())
}
