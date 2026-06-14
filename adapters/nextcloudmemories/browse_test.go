package nextcloudmemories

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/assettracker"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/fileprocessor"
	"github.com/simulot/immich-go/internal/nextcloud"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBrowseEnumeratesSupportedAssets(t *testing.T) {
	t.Parallel()

	nc := &Command{
		app: newTestApp(t),
		sourceFS: fstest.MapFS{
			"Photos/IMG_0001.JPG":      {Data: []byte("image"), ModTime: time.Unix(1700000000, 0)},
			"Photos/VID_0001.MP4":      {Data: []byte("video"), ModTime: time.Unix(1700000100, 0)},
			"Photos/ignored.json":      {Data: []byte("{}")},
			"Photos/readme.txt":        {Data: []byte("notes")},
			"Photos/.DS_Store":         {Data: []byte("finder")},
			"Photos/Sub/IMG_0002.HEIC": {Data: []byte("heic")},
			"Scans/SCAN_0001.JPG":      {Data: []byte("scan")},
		},
		selectedRoots: []string{"/Photos"},
	}

	groups := collectGroups(nc.Browse(context.Background()))
	require.Len(t, groups, 3)

	assetNames := []string{}
	for _, group := range groups {
		require.Len(t, group.Assets, 1)
		assetNames = append(assetNames, group.Assets[0].File.Name())
	}

	assert.ElementsMatch(t, []string{
		"Photos/IMG_0001.JPG",
		"Photos/VID_0001.MP4",
		"Photos/Sub/IMG_0002.HEIC",
	}, assetNames)

	counts := nc.app.FileProcessor().Logger().GetCounts()
	assert.EqualValues(t, 2, counts[fileevent.DiscoveredImage])
	assert.EqualValues(t, 1, counts[fileevent.DiscoveredVideo])
	assert.EqualValues(t, 1, counts[fileevent.DiscoveredSidecar])
	assert.EqualValues(t, 1, counts[fileevent.DiscoveredUnknown])
	assert.EqualValues(t, 1, counts[fileevent.DiscoveredBanned])
}

func TestBrowseDeduplicatesOverlappingRoots(t *testing.T) {
	t.Parallel()

	nc := &Command{
		app: newTestApp(t),
		sourceFS: fstest.MapFS{
			"Photos/Trips/IMG_0001.JPG": {Data: []byte("image")},
		},
		selectedRoots: []string{"/Photos", "/Photos/Trips"},
	}

	groups := collectGroups(nc.Browse(context.Background()))
	require.Len(t, groups, 1)
	assert.Equal(t, "Photos/Trips/IMG_0001.JPG", groups[0].Assets[0].File.Name())

	counts := nc.app.FileProcessor().Logger().GetCounts()
	assert.EqualValues(t, 1, counts[fileevent.DiscoveredImage])
}

func TestBrowseEnrichesAssetsWithMemoriesMetadata(t *testing.T) {
	t.Parallel()

	nc := &Command{
		app: newTestApp(t),
		sourceFS: fstest.MapFS{
			"Photos/IMG_0001.JPG": {Data: []byte("image"), ModTime: time.Unix(1700000000, 0)},
		},
		selectedRoots: []string{"/Photos"},
		metadataIndex: newMemoriesMetadataIndex(),
	}
	nc.metadataIndex.photosByPath["Photos/IMG_0001.JPG"] = &indexedMemoriesPhoto{
		record: &memoriesAssetRecord{CanonicalPath: "Photos/IMG_0001.JPG"},
		loaded: true,
		metadata: &assets.Metadata{
			Description: "Sunset",
			Favorited:   true,
			Rating:      5,
			Albums:      []assets.Album{assets.NewAlbum("", "Roadtrip", "")},
			Tags:        []assets.Tag{{Name: "Travel", Value: "Travel"}},
		},
	}

	groups := collectGroups(nc.Browse(context.Background()))
	require.Len(t, groups, 1)
	require.Len(t, groups[0].Assets, 1)

	asset := groups[0].Assets[0]
	require.NotNil(t, asset.FromApplication)
	assert.Equal(t, "Sunset", asset.Description)
	assert.True(t, asset.Favorite)
	assert.Equal(t, 5, asset.Rating)
	require.Len(t, asset.Albums, 1)
	assert.Equal(t, "Roadtrip", asset.Albums[0].Title)
	require.Len(t, asset.Tags, 1)
	assert.Equal(t, "Travel", asset.Tags[0].Value)
}

func TestBrowseWarnsAndContinuesForUnindexedAssetsByDefault(t *testing.T) {
	t.Parallel()

	nc := &Command{
		app: newTestApp(t),
		sourceFS: fstest.MapFS{
			"Photos/IMG_0001.JPG": {Data: []byte("image")},
		},
		selectedRoots: []string{"/Photos"},
		metadataIndex: newMemoriesMetadataIndex(),
	}

	groups := collectGroups(nc.Browse(context.Background()))
	require.Len(t, groups, 1)
	assert.Nil(t, groups[0].Assets[0].FromApplication)
}

func TestBrowseRejectsUnindexedAssetsWhenStrictModeIsRequested(t *testing.T) {
	t.Parallel()

	nc := &Command{
		app:            newTestApp(t),
		RequireIndexed: true,
		sourceFS: fstest.MapFS{
			"Photos/IMG_0001.JPG": {Data: []byte("image")},
		},
		selectedRoots: []string{"/Photos"},
		metadataIndex: newMemoriesMetadataIndex(),
	}

	groups := collectGroups(nc.Browse(context.Background()))
	assert.Empty(t, groups)
}

func TestBrowseFallsBackWhenMetadataIndexBuildFails(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.php/apps/memories/api/days":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `[{"dayid":19723,"count":1}]`)
		case "/index.php/apps/memories/api/days/19723":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `[{"fileid":42,"basename":"IMG_0001.JPG","mimetype":"image/jpeg","dayid":19723,"datetaken":1700000000}]`)
		case "/index.php/apps/memories/api/image/info/42":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"fileid":42,"filename":"/Photos/IMG_0001.JPG","tags":123}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := mustNewClient(t, server.URL)

	nc := &Command{
		app: newTestApp(t),
		sourceFS: fstest.MapFS{
			"Photos/IMG_0001.JPG": {Data: []byte("image")},
		},
		selectedRoots: []string{"/Photos"},
		client:        client,
		discovery: &nextcloud.MemoriesDiscovery{
			Config: nextcloud.MemoriesConfig{SystemTagsEnabled: true},
		},
	}

	groups := collectGroups(nc.Browse(context.Background()))
	require.Len(t, groups, 1)
	assert.Nil(t, groups[0].Assets[0].FromApplication)
	assert.NotNil(t, nc.metadataIndex)
	assert.Len(t, nc.metadataIndex.photosByID, 1)
	counts := nc.app.FileProcessor().Logger().GetCounts()
	assert.EqualValues(t, 1, counts[fileevent.DiscoveredImage])
}

func TestBrowseResolvesMetadataAfterCanonicalFilenameHydration(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.php/apps/memories/api/image/info/42":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"fileid":42,"basename":"IMG_0001.JPG","mimetype":"image/jpeg","filename":"/Photos/2017/IMG_0001.JPG","clusters":{"albums":[{"album_id":7,"name":"Familie","user":"alice"}]}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := mustNewClient(t, server.URL)
	nc := &Command{
		app: newTestApp(t),
		sourceFS: fstest.MapFS{
			"Photos/2017/IMG_0001.JPG": {Data: []byte("image"), ModTime: time.Unix(1700000000, 0)},
		},
		selectedRoots: []string{"/Photos"},
		client:        client,
		metadataIndex: newMemoriesMetadataIndex(),
	}
	nc.metadataIndex.client = client
	nc.metadataIndex.selectedRoots = []string{"/Photos"}
	nc.metadataIndex.options = metadataMappingOptions{OwnerUID: "alice", SyncAlbums: true}
	nc.metadataIndex.infoQuery = nextcloud.MemoriesImageInfoQuery{Clusters: []string{"albums"}}
	nc.metadataIndex.Put("", nextcloud.MemoriesPhoto{FileID: 42, Basename: "IMG_0001.JPG", DateTaken: 1700000000})

	groups := collectGroups(nc.Browse(context.Background()))
	require.Len(t, groups, 1)
	asset := groups[0].Assets[0]
	require.NotNil(t, asset.FromApplication)
	require.Len(t, asset.Albums, 1)
	assert.Equal(t, "Familie", asset.Albums[0].Title)
	_, ok, err := nc.metadataIndex.Get(context.Background(), "Photos/2017/IMG_0001.JPG")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Contains(t, nc.metadataIndex.photosByPath, "Photos/2017/IMG_0001.JPG")
}

func TestBrowseResolvesMetadataByUniqueBasenameBeforeCanonicalPathKnown(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.php/apps/memories/api/image/info/42":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"fileid":42,"basename":"19-04-08 17-51-33 0488.jpg","mimetype":"image/jpeg","filename":"/Photos/2019/04/19-04-08 17-51-33 0488.jpg","clusters":{"albums":[{"album_id":7,"name":"Familie","user":"alice"}]}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := mustNewClient(t, server.URL)
	nc := &Command{
		app: newTestApp(t),
		sourceFS: fstest.MapFS{
			"Photos/2019/04/19-04-08 17-51-33 0488.jpg": {Data: []byte("image"), ModTime: time.Unix(1700000000, 0)},
		},
		selectedRoots: []string{"/Photos"},
		client:        client,
		metadataIndex: newMemoriesMetadataIndex(),
	}
	nc.metadataIndex.client = client
	nc.metadataIndex.selectedRoots = []string{"/Photos"}
	nc.metadataIndex.options = metadataMappingOptions{OwnerUID: "alice", SyncAlbums: true}
	nc.metadataIndex.infoQuery = nextcloud.MemoriesImageInfoQuery{Clusters: []string{"albums"}}
	nc.metadataIndex.Put("", nextcloud.MemoriesPhoto{FileID: 42, Basename: "19-04-08 17-51-33 0488.jpg", DateTaken: 1700000000})

	groups := collectGroups(nc.Browse(context.Background()))
	require.Len(t, groups, 1)
	asset := groups[0].Assets[0]
	require.NotNil(t, asset.FromApplication)
	require.Len(t, asset.Albums, 1)
	assert.Equal(t, "Familie", asset.Albums[0].Title)
	assert.Contains(t, nc.metadataIndex.photosByPath, "Photos/2019/04/19-04-08 17-51-33 0488.jpg")
}

func TestMetadataIndexWarmupLearnsCanonicalPathBeforeDirectLookup(t *testing.T) {
	t.Parallel()

	var infoCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.php/apps/memories/api/image/info/42":
			infoCalls.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"fileid":42,"basename":"19-04-08 17-51-33 0488.jpg","mimetype":"image/jpeg","filename":"/Photos/2019/04/19-04-08 17-51-33 0488.jpg","clusters":{"albums":[{"album_id":7,"name":"Familie","user":"alice"}]}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := mustNewClient(t, server.URL)
	idx := newMemoriesMetadataIndex()
	idx.client = client
	idx.selectedRoots = []string{"/Photos"}
	idx.options = metadataMappingOptions{OwnerUID: "alice", SyncAlbums: true}
	idx.infoQuery = nextcloud.MemoriesImageInfoQuery{Clusters: []string{"albums"}}
	idx.Put("", nextcloud.MemoriesPhoto{FileID: 42, Basename: "19-04-08 17-51-33 0488.jpg", DateTaken: 1700000000})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	idx.startWarmup(ctx)

	idx.mu.Lock()
	entry, ok := idx.photosByBase["19-04-08 17-51-33 0488.jpg"]
	require.True(t, ok)
	require.Len(t, entry, 1)
	idx.enqueueWarmLocked(entry[0])
	idx.mu.Unlock()

	require.Eventually(t, func() bool {
		idx.mu.Lock()
		defer idx.mu.Unlock()
		_, ok := idx.photosByPath["Photos/2019/04/19-04-08 17-51-33 0488.jpg"]
		return ok
	}, time.Second, 10*time.Millisecond)

	md, ok, err := idx.Get(context.Background(), "Photos/2019/04/19-04-08 17-51-33 0488.jpg")
	require.NoError(t, err)
	require.True(t, ok)
	require.NotNil(t, md)
	require.Len(t, md.Albums, 1)
	assert.Equal(t, "Familie", md.Albums[0].Title)
	assert.EqualValues(t, 1, infoCalls.Load())
}

func TestMetadataIndexRepeatedLookupIsIdempotent(t *testing.T) {
	t.Parallel()

	var infoCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.php/apps/memories/api/image/info/42":
			infoCalls.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"fileid":42,"basename":"IMG_0001.JPG","mimetype":"image/jpeg","filename":"/Photos/2017/IMG_0001.JPG","clusters":{"albums":[{"album_id":7,"name":"Familie","user":"alice"}]}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := mustNewClient(t, server.URL)
	idx := newMemoriesMetadataIndex()
	idx.client = client
	idx.selectedRoots = []string{"/Photos"}
	idx.options = metadataMappingOptions{OwnerUID: "alice", SyncAlbums: true}
	idx.infoQuery = nextcloud.MemoriesImageInfoQuery{Clusters: []string{"albums"}}
	idx.Put("", nextcloud.MemoriesPhoto{FileID: 42, Basename: "IMG_0001.JPG", DateTaken: 1700000000})

	md1, ok, err := idx.Get(context.Background(), "Photos/2017/IMG_0001.JPG")
	require.NoError(t, err)
	require.True(t, ok)
	require.NotNil(t, md1)

	md2, ok, err := idx.Get(context.Background(), "Photos/2017/IMG_0001.JPG")
	require.NoError(t, err)
	require.True(t, ok)
	require.NotNil(t, md2)

	require.Len(t, md1.Albums, 1)
	require.Len(t, md2.Albums, 1)
	assert.Equal(t, "Familie", md1.Albums[0].Title)
	assert.Equal(t, "Familie", md2.Albums[0].Title)
	assert.EqualValues(t, 1, infoCalls.Load())
}

func TestTimelineRootToFSPath(t *testing.T) {
	t.Parallel()

	assert.Equal(t, ".", timelineRootToFSPath("/"))
	assert.Equal(t, ".", timelineRootToFSPath(" "))
	assert.Equal(t, "Photos", timelineRootToFSPath("/Photos"))
}

func collectGroups(in chan *assets.Group) []*assets.Group {
	groups := []*assets.Group{}
	for group := range in {
		groups = append(groups, group)
	}
	return groups
}

func newTestApp(t *testing.T) *app.Application {
	t.Helper()

	a := app.New(context.Background(), &cobra.Command{})
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	a.Log().Logger = logger
	bus := fileevent.NewBus()
	tracker := assettracker.NewWithBus(logger, false, bus)
	a.SetFileProcessor(fileprocessor.NewWithBus(tracker, logger, bus))
	return a
}

func mustNewClient(t *testing.T, baseURL string) *nextcloud.Client {
	t.Helper()

	client, err := nextcloud.NewClient(nextcloud.Config{
		BaseURL:  baseURL,
		Username: "alice",
		Password: "secret",
		Timeout:  time.Minute,
	})
	require.NoError(t, err)
	return client
}
