package nextcloudmemories

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	"github.com/simulot/immich-go/adapters"
	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/nextcloud"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFromNextcloudMemoriesCommandMetadata(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	parent := &cobra.Command{Use: "upload"}
	a := app.New(ctx, parent)
	cmd := NewFromNextcloudMemoriesCommand(ctx, parent, a, nil)

	assert.Equal(t, "from-nextcloud-memories [flags]", cmd.Use)
	assert.Empty(t, cmd.Aliases)
	assert.Contains(t, cmd.Long, "not arbitrary Nextcloud storage paths")
	assert.Contains(t, cmd.Example, "--discover-only")
	assert.Contains(t, cmd.Example, "--timeline-root=/Photos")
	assert.NotNil(t, cmd.Flag("nextcloud-url"))
	assert.NotNil(t, cmd.Flag("nextcloud-user"))
	assert.NotNil(t, cmd.Flag("nextcloud-password"))
	assert.NotNil(t, cmd.Flag("nextcloud-local-dir"))
	assert.NotNil(t, cmd.Flag("discover-only"))
	assert.NotNil(t, cmd.Flag("timeline-root"))
	assert.NotNil(t, cmd.Flag("sync-albums"))
	assert.NotNil(t, cmd.Flag("sync-tags"))
	assert.NotNil(t, cmd.Flag("require-indexed"))
	assert.NotNil(t, cmd.Flag("tag-album-membership"))
	assert.NotNil(t, cmd.Flag("user-map"))
}

func TestCommandValidate(t *testing.T) {
	t.Parallel()

	t.Run("missing required flags", func(t *testing.T) {
		t.Parallel()

		nc := &Command{NextcloudClientTimeout: 5 * time.Minute}
		err := nc.validate()
		require.Error(t, err)
		assert.ErrorContains(t, err, "--nextcloud-url")
		assert.ErrorContains(t, err, "--nextcloud-user")
		assert.ErrorContains(t, err, "--nextcloud-password")
	})

	t.Run("normalizes relative timeline root", func(t *testing.T) {
		t.Parallel()

		nc := &Command{
			NextcloudURL:           "https://cloud.example.com/",
			NextcloudUser:          "alice",
			NextcloudPassword:      "secret",
			NextcloudClientTimeout: 5 * time.Minute,
			TimelineRoots:          []string{"Photos"},
		}
		err := nc.validate()
		require.NoError(t, err)
		assert.Equal(t, []string{"/Photos"}, nc.TimelineRoots)
	})

	t.Run("normalizes url and roots", func(t *testing.T) {
		t.Parallel()
		localDir := t.TempDir()

		nc := &Command{
			NextcloudURL:           " https://cloud.example.com/remote.php/ ",
			NextcloudUser:          " alice ",
			NextcloudPassword:      " secret ",
			NextcloudLocalDir:      " " + localDir + " ",
			NextcloudClientTimeout: 5 * time.Minute,
			TimelineRoots:          []string{" //Photos/ ", "/Photos", "/Scans//"},
		}
		err := nc.validate()
		require.NoError(t, err)
		assert.Equal(t, "https://cloud.example.com/remote.php", nc.NextcloudURL)
		assert.Equal(t, "alice", nc.NextcloudUser)
		assert.Equal(t, "secret", nc.NextcloudPassword)
		assert.Equal(t, localDir, nc.NextcloudLocalDir)
		assert.Equal(t, []string{"/Photos", "/Scans"}, nc.TimelineRoots)
	})

	t.Run("invalid timeout", func(t *testing.T) {
		t.Parallel()

		nc := &Command{
			NextcloudURL:           "https://cloud.example.com",
			NextcloudUser:          "alice",
			NextcloudPassword:      "secret",
			NextcloudClientTimeout: 0,
		}
		err := nc.validate()
		require.Error(t, err)
		assert.ErrorContains(t, err, "--nextcloud-client-timeout")
	})

	t.Run("invalid user map", func(t *testing.T) {
		t.Parallel()

		nc := &Command{
			NextcloudURL:           "https://cloud.example.com",
			NextcloudUser:          "alice",
			NextcloudPassword:      "secret",
			NextcloudClientTimeout: 5 * time.Minute,
			UserMaps:               []string{"bob"},
		}
		err := nc.validate()
		require.Error(t, err)
		assert.ErrorContains(t, err, "--user-map")
	})
}

func TestCommandRunRequiresRunner(t *testing.T) {
	t.Parallel()

	nc := Command{
		NextcloudURL:           "https://cloud.example.com",
		NextcloudUser:          "alice",
		NextcloudPassword:      "secret",
		NextcloudClientTimeout: 5 * time.Minute,
	}

	err := nc.Run(context.Background(), &cobra.Command{}, nil)
	require.Error(t, err)
	assert.ErrorContains(t, err, "runner is not configured")
}

func TestCommandRunImportUsesRunner(t *testing.T) {
	t.Parallel()

	nc := Command{
		NextcloudURL:           "https://cloud.example.com",
		NextcloudUser:          "alice",
		NextcloudPassword:      "secret",
		NextcloudClientTimeout: 5 * time.Minute,
		app:                    newTestApp(t),
		sourceFS: fstest.MapFS{
			"Photos/IMG_0001.JPG": {Data: []byte("image")},
		},
		selectedRoots: []string{"/Photos"},
	}

	called := false
	err := nc.Run(context.Background(), &cobra.Command{}, runnerFunc(func(cmd *cobra.Command, adapter adapters.Reader) error {
		called = true
		groups := collectGroups(adapter.Browse(context.Background()))
		require.Len(t, groups, 1)
		assert.Equal(t, "Photos/IMG_0001.JPG", groups[0].Assets[0].File.Name())
		return nil
	}))
	require.NoError(t, err)
	assert.True(t, called)
}

func TestPrepareImportUsesLocalDirectoryAsPreferredFileSource(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	localDir := t.TempDir()
	photosDir := filepath.Join(localDir, "Photos")
	require.NoError(t, os.MkdirAll(photosDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(photosDir, "IMG_0001.JPG"), []byte("image-local"), 0o644))

	remoteFS := fstest.MapFS{
		"Photos/IMG_0001.JPG": {Data: []byte("image-remote")},
	}

	nc := Command{
		NextcloudURL:           server.URL,
		NextcloudUser:          "alice",
		NextcloudPassword:      "secret",
		NextcloudLocalDir:      localDir,
		NextcloudClientTimeout: 5 * time.Minute,
		newRemoteFS: func(ctx context.Context, client *nextcloud.Client, uid string) (fs.FS, error) {
			return remoteFS, nil
		},
		discover: func(ctx context.Context, cfg nextcloud.Config) (*nextcloud.MemoriesDiscovery, error) {
			return &nextcloud.MemoriesDiscovery{
				TimelineRoots: []string{"/Photos"},
				Describe:      nextcloud.MemoriesDescribe{UID: stringPtr("alice")},
			}, nil
		},
	}

	err := nc.prepareImport(context.Background())
	require.NoError(t, err)
	require.NotNil(t, nc.sourceFS)
	assert.Equal(t, []string{"/Photos"}, nc.selectedRoots)
	nc.metadataIndex = newMemoriesMetadataIndex()

	groups := collectGroups(nc.Browse(context.Background()))
	require.Len(t, groups, 1)
	assert.Equal(t, "Photos/IMG_0001.JPG", groups[0].Assets[0].File.Name())

	f, err := groups[0].Assets[0].File.Open()
	require.NoError(t, err)
	defer f.Close()

	b, err := io.ReadAll(f)
	require.NoError(t, err)
	assert.Equal(t, "image-local", string(b))
}

func TestCommandRunDiscoverOnly(t *testing.T) {
	t.Parallel()

	nc := Command{
		NextcloudURL:           "https://cloud.example.com",
		NextcloudUser:          "alice",
		NextcloudPassword:      "secret",
		NextcloudClientTimeout: 5 * time.Minute,
		DiscoverOnly:           true,
		discover: func(ctx context.Context, cfg nextcloud.Config) (*nextcloud.MemoriesDiscovery, error) {
			assert.Equal(t, "https://cloud.example.com", cfg.BaseURL)
			assert.Equal(t, "alice", cfg.Username)
			assert.Equal(t, "secret", cfg.Password)
			return &nextcloud.MemoriesDiscovery{
				BaseURL:       cfg.BaseURL,
				DAVRoot:       cfg.BaseURL + "/remote.php/dav",
				Capabilities:  nextcloud.OCSCapabilities{VersionString: "31.0.0", ProductName: "Nextcloud", Edition: "community"},
				Describe:      nextcloud.MemoriesDescribe{Version: "7.5.0", BaseURL: cfg.BaseURL + "/index.php/apps/memories", UID: stringPtr("alice")},
				Config:        nextcloud.MemoriesConfig{FoldersPath: "/", AlbumsEnabled: true, SystemTagsEnabled: true, PreviewGeneratorEnabled: false},
				TimelineRoots: []string{"/Photos", "/Scans"},
			}, nil
		},
	}

	var output bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&output)

	err := nc.Run(context.Background(), cmd, nil)
	require.NoError(t, err)
	text := output.String()
	assert.Contains(t, text, "Nextcloud Memories scaffold")
	assert.Contains(t, text, "Nextcloud Memories discovery")
	assert.Contains(t, text, "nextcloud-version: 31.0.0")
	assert.Contains(t, text, "configured-timeline-roots:")
	assert.Contains(t, text, "selected-timeline-roots:")
	assert.Contains(t, text, "    - /Photos")
	assert.Contains(t, text, "albums-enabled: true")
}

func TestCommandRunDiscoverOnlyRejectsUnknownRequestedRoots(t *testing.T) {
	t.Parallel()

	nc := Command{
		NextcloudURL:           "https://cloud.example.com",
		NextcloudUser:          "alice",
		NextcloudPassword:      "secret",
		NextcloudClientTimeout: 5 * time.Minute,
		DiscoverOnly:           true,
		TimelineRoots:          []string{"/Scans"},
		discover: func(ctx context.Context, cfg nextcloud.Config) (*nextcloud.MemoriesDiscovery, error) {
			return &nextcloud.MemoriesDiscovery{TimelineRoots: []string{"/Photos"}}, nil
		},
	}

	err := nc.Run(context.Background(), &cobra.Command{}, nil)
	require.Error(t, err)
	assert.ErrorContains(t, err, "requested --timeline-root")
	assert.ErrorContains(t, err, "/Photos")
}

func TestCommandIntentSummary(t *testing.T) {
	t.Parallel()

	t.Run("auto discover roots", func(t *testing.T) {
		t.Parallel()

		nc := &Command{
			NextcloudURL:      "https://cloud.example.com",
			NextcloudUser:     "alice",
			DiscoverOnly:      true,
			SyncAlbums:        true,
			RequireIndexed:    false,
			TimelineRoots:     nil,
			NextcloudPassword: "secret",
		}

		summary := nc.intentSummary()
		assert.Contains(t, summary, "mode: discover-only")
		assert.Contains(t, summary, "source-files: webdav")
		assert.Contains(t, summary, "timeline-roots: all configured Memories timeline roots")
		assert.Contains(t, summary, "sync-albums: true")
		assert.Contains(t, summary, "tag-album-membership: false")
		assert.Contains(t, summary, "user-maps: 0")
		assert.NotContains(t, summary, "secret")
	})

	t.Run("restricted roots", func(t *testing.T) {
		t.Parallel()

		nc := &Command{
			NextcloudURL:           "https://cloud.example.com",
			NextcloudUser:          "alice",
			NextcloudLocalDir:      "/srv/nextcloud-sync",
			SyncAlbums:             false,
			RequireIndexed:         true,
			TagAlbumMembership:     true,
			NextcloudSkipVerifySSL: true,
			TimelineRoots:          []string{"/Photos", "/Scans"},
		}

		summary := nc.intentSummary()
		assert.Contains(t, summary, "mode: import")
		assert.Contains(t, summary, "source-files: local-first (/srv/nextcloud-sync), fallback webdav")
		assert.Contains(t, summary, "timeline-roots: /Photos, /Scans")
		assert.Contains(t, summary, "sync-albums: false")
		assert.Contains(t, summary, "require-indexed: true")
		assert.Contains(t, summary, "tag-album-membership: true")
		assert.Contains(t, summary, "user-maps: 0")
		assert.Contains(t, summary, "skip-verify-ssl: true")
	})
}

func TestCommandDesiredAlbumUsersMapsCollaborators(t *testing.T) {
	t.Parallel()

	description, err := applyManagedAlbumState("Roadtrip notes", managedAlbumState{
		Source:        "nextcloud-memories",
		SchemaVersion: 1,
		AlbumID:       42,
		OwnerUID:      "alice",
		AlbumName:     "Roadtrip",
	})
	require.NoError(t, err)

	nc := &Command{
		NextcloudURL:           "https://cloud.example.com",
		NextcloudUser:          "alice",
		NextcloudPassword:      "secret",
		NextcloudClientTimeout: 5 * time.Minute,
		UserMaps:               []string{"bob=immich-bob", "carol=immich-carol"},
		client:                 &nextcloud.Client{},
		app:                    newTestApp(t),
		getAlbumCollaborators: func(ctx context.Context, client *nextcloud.Client, ownerUID string, albumName string) ([]nextcloud.AlbumCollaborator, error) {
			assert.Equal(t, "alice", ownerUID)
			assert.Equal(t, "Roadtrip", albumName)
			return []nextcloud.AlbumCollaborator{
				{ID: "bob", Label: "Bob", Type: nextcloud.AlbumCollaboratorTypeUser},
				{ID: "team", Label: "Team", Type: nextcloud.AlbumCollaboratorTypeGroup},
				{ID: "carol", Label: "Carol", Type: nextcloud.AlbumCollaboratorTypeUser},
			}, nil
		},
	}
	require.NoError(t, nc.validate())

	users, err := nc.DesiredAlbumUsers(context.Background(), assets.Album{Title: "Roadtrip", Description: description})
	require.NoError(t, err)
	require.Len(t, users, 2)
	assert.Equal(t, adapters.AlbumUser{UserID: "immich-bob", Role: "editor"}, users[0])
	assert.Equal(t, adapters.AlbumUser{UserID: "immich-carol", Role: "editor"}, users[1])
}

func TestCommandRejectsPositionalArguments(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	parent := &cobra.Command{Use: "upload"}
	a := app.New(ctx, parent)
	_ = NewFromNextcloudMemoriesCommand(ctx, parent, a, nil)

	err := cobra.NoArgs(&cobra.Command{Use: "from-nextcloud-memories"}, []string{"unexpected-path"})
	require.Error(t, err)
	assert.ErrorContains(t, err, "unknown command \"unexpected-path\" for \"from-nextcloud-memories\"")
}

func stringPtr(value string) *string {
	return &value
}

type runnerFunc func(cmd *cobra.Command, adapter adapters.Reader) error

func (fn runnerFunc) Run(cmd *cobra.Command, adapter adapters.Reader) error {
	return fn(cmd, adapter)
}
