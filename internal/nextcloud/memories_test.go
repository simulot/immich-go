package nextcloud

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetOCSCapabilities(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ocs/v2.php/cloud/capabilities":
			assert.Equal(t, "true", r.Header.Get("OCS-APIRequest"))
			assert.Equal(t, "json", r.URL.Query().Get("format"))
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"ocs":{"meta":{"status":"ok","statuscode":100,"message":"OK"},"data":{"version":{"string":"31.0.0","edition":"community","productname":"Nextcloud"}}}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := mustNewClient(t, server.URL)
	capabilities, err := GetOCSCapabilities(context.Background(), client)
	require.NoError(t, err)
	assert.Equal(t, "31.0.0", capabilities.VersionString)
	assert.Equal(t, "community", capabilities.Edition)
	assert.Equal(t, "Nextcloud", capabilities.ProductName)
}

func TestGetOCSCapabilitiesAcceptsStatusCode200(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/ocs/v2.php/cloud/capabilities", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"ocs":{"meta":{"status":"ok","statuscode":200,"message":"OK"},"data":{"version":{"string":"31.0.0","edition":"community","productname":"Nextcloud"}}}}`)
	}))
	defer server.Close()

	client := mustNewClient(t, server.URL)
	capabilities, err := GetOCSCapabilities(context.Background(), client)
	require.NoError(t, err)
	assert.Equal(t, "31.0.0", capabilities.VersionString)
}

func TestDiscoverMemories(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ocs/v2.php/cloud/capabilities":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"ocs":{"meta":{"status":"ok","statuscode":100,"message":"OK"},"data":{"version":{"string":"31.0.0","edition":"community","productname":"Nextcloud"}}}}`)
		case "/index.php/apps/memories/api/describe":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"version":"7.5.0","baseUrl":"https://cloud.example.com/apps/memories","loginFlowUrl":"https://cloud.example.com/login","uid":"alice"}`)
		case "/index.php/apps/memories/api/config":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"version":"7.5.0","timeline_path":"/Photos;/Scans/;/Photos","folders_path":"/","albums_enabled":true,"systemtags_enabled":true,"preview_generator_enabled":false,"recognize_installed":false,"recognize_enabled":false,"facerecognition_installed":false,"facerecognition_enabled":false}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := mustNewClient(t, server.URL)
	discovery, err := DiscoverMemories(context.Background(), client)
	require.NoError(t, err)

	assert.Equal(t, server.URL, discovery.BaseURL)
	assert.Equal(t, server.URL+"/remote.php/dav", discovery.DAVRoot)
	assert.Equal(t, "31.0.0", discovery.Capabilities.VersionString)
	assert.Equal(t, "7.5.0", discovery.Describe.Version)
	assert.Equal(t, []string{"/Photos", "/Scans"}, discovery.TimelineRoots)
	assert.True(t, discovery.Config.AlbumsEnabled)
	assert.True(t, discovery.Config.SystemTagsEnabled)
	assert.Equal(t, "/", discovery.Config.FoldersPath)
	if assert.NotNil(t, discovery.Describe.UID) {
		assert.Equal(t, "alice", *discovery.Describe.UID)
	}
}

func TestGetMemoriesDaysAppliesTimelineQuery(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/index.php/apps/memories/api/days", r.URL.Path)
		assert.Equal(t, "/Photos", r.URL.Query().Get("folder"))
		assert.Equal(t, "1", r.URL.Query().Get("recursive"))
		assert.Equal(t, "1", r.URL.Query().Get("hidden"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `[{"dayid":19723,"count":2}]`)
	}))
	defer server.Close()

	client := mustNewClient(t, server.URL)
	days, err := GetMemoriesDays(context.Background(), client, MemoriesTimelineQuery{
		Folder:    "/Photos",
		Recursive: true,
		Hidden:    true,
	})
	require.NoError(t, err)
	require.Len(t, days, 1)
	assert.Equal(t, 19723, days[0].DayID)
	assert.Equal(t, 2, days[0].Count)
}

func TestGetMemoriesDayParsesPhotoFlags(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/index.php/apps/memories/api/days/19723,19722", r.URL.Path)
		assert.Equal(t, "/Scans", r.URL.Query().Get("folder"))
		assert.Equal(t, "1", r.URL.Query().Get("archive"))
		assert.Equal(t, "1", r.URL.Query().Get("hidden"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `[{"fileid":42,"basename":"IMG_0042.JPG","mimetype":"image/jpeg","dayid":19723,"datetaken":1700000000,"isfavorite":1,"ishidden":true}]`)
	}))
	defer server.Close()

	client := mustNewClient(t, server.URL)
	photos, err := GetMemoriesDay(context.Background(), client, []int{19723, 19722}, MemoriesTimelineQuery{
		Folder:  "/Scans",
		Archive: true,
		Hidden:  true,
	})
	require.NoError(t, err)
	require.Len(t, photos, 1)
	assert.Equal(t, 42, photos[0].FileID)
	assert.Equal(t, "IMG_0042.JPG", photos[0].Basename)
	assert.Equal(t, int64(1700000000), photos[0].DateTaken)
	assert.True(t, bool(photos[0].IsFavorite))
	assert.True(t, bool(photos[0].IsHidden))
}

func TestGetMemoriesImageInfoRequestsOptionalExpansions(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/index.php/apps/memories/api/image/info/42", r.URL.Path)
		assert.Equal(t, "1", r.URL.Query().Get("tags"))
		assert.Equal(t, "albums,recognize", r.URL.Query().Get("clusters"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"fileid":42,"datetaken":1700000000,"basename":"IMG_0042.JPG","mimetype":"image/jpeg","filename":"/Photos/IMG_0042.JPG","tags":{"1":"Travel"},"exif":{"Description":"sunset","Rating":"5"},"clusters":{"albums":[{"album_id":7,"cluster_id":"alice/Roadtrip","name":"Roadtrip","user":"alice","shared":false}]}}`)
	}))
	defer server.Close()

	client := mustNewClient(t, server.URL)
	info, err := GetMemoriesImageInfo(context.Background(), client, 42, MemoriesImageInfoQuery{
		Tags:     true,
		Clusters: []string{"albums", "recognize"},
	})
	require.NoError(t, err)
	assert.Equal(t, 42, info.FileID)
	assert.Equal(t, "/Photos/IMG_0042.JPG", info.FileName)
	assert.Equal(t, "Travel", info.Tags["1"])
	assert.Equal(t, "sunset", info.Exif["Description"])
	require.Len(t, info.Clusters.Albums, 1)
	assert.Equal(t, "Roadtrip", info.Clusters.Albums[0].Name)
}

func TestGetMemoriesImageInfoAcceptsArrayTags(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/index.php/apps/memories/api/image/info/42", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"fileid":42,"datetaken":1700000000,"basename":"IMG_0042.JPG","mimetype":"image/jpeg","filename":"/Photos/IMG_0042.JPG","tags":["Travel","Family"]}`)
	}))
	defer server.Close()

	client := mustNewClient(t, server.URL)
	info, err := GetMemoriesImageInfo(context.Background(), client, 42, MemoriesImageInfoQuery{Tags: true})
	require.NoError(t, err)
	assert.Equal(t, "Travel", info.Tags["0"])
	assert.Equal(t, "Family", info.Tags["1"])
}

func TestGetMemoriesImageInfoAcceptsNumericAlbumSharedFlag(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/index.php/apps/memories/api/image/info/42", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"fileid":42,"datetaken":1700000000,"basename":"IMG_0042.JPG","mimetype":"image/jpeg","filename":"/Photos/IMG_0042.JPG","clusters":{"albums":[{"album_id":7,"cluster_id":"alice/Roadtrip","name":"Roadtrip","user":"alice","shared":1}]}}`)
	}))
	defer server.Close()

	client := mustNewClient(t, server.URL)
	info, err := GetMemoriesImageInfo(context.Background(), client, 42, MemoriesImageInfoQuery{Clusters: []string{"albums"}})
	require.NoError(t, err)
	require.Len(t, info.Clusters.Albums, 1)
	assert.True(t, bool(info.Clusters.Albums[0].Shared))
}

func TestSplitTimelineRootsNormalizesLeadingSlashes(t *testing.T) {
	t.Parallel()

	roots := splitTimelineRoots("//Photos; /Scans// ;Photos")
	assert.Equal(t, []string{"/Photos", "/Scans"}, roots)
}

func TestDiscoverMemoriesRejectsEmptyTimelineRoots(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ocs/v2.php/cloud/capabilities":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"ocs":{"meta":{"status":"ok","statuscode":100,"message":"OK"},"data":{"version":{"string":"31.0.0","edition":"community","productname":"Nextcloud"}}}}`)
		case "/index.php/apps/memories/api/describe":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"version":"7.5.0","baseUrl":"https://cloud.example.com/apps/memories","loginFlowUrl":"https://cloud.example.com/login","uid":"alice"}`)
		case "/index.php/apps/memories/api/config":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"version":"7.5.0","timeline_path":"","folders_path":"/","albums_enabled":true,"systemtags_enabled":true,"preview_generator_enabled":false}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := mustNewClient(t, server.URL)
	_, err := DiscoverMemories(context.Background(), client)
	require.Error(t, err)
	assert.ErrorContains(t, err, "timeline roots")
}

func TestGetOCSCapabilitiesRejectsBadMetaStatus(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"ocs":{"meta":{"status":"failure","statuscode":997,"message":"Authentication failed"},"data":{}}}`)
	}))
	defer server.Close()

	client := mustNewClient(t, server.URL)
	_, err := GetOCSCapabilities(context.Background(), client)
	require.Error(t, err)
	assert.ErrorContains(t, err, "Authentication failed")
}

func mustNewClient(t *testing.T, baseURL string) *Client {
	t.Helper()

	client, err := NewClient(Config{
		BaseURL:  baseURL,
		Username: "alice",
		Password: "secret",
		Timeout:  time.Minute,
	})
	require.NoError(t, err)
	return client
}
