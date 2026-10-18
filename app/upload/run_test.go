package upload

import (
	"context"
	"io"
	"log/slog"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/spf13/cobra"

	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/assets/cache"
	"github.com/simulot/immich-go/internal/assettracker"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/fileprocessor"
	"github.com/simulot/immich-go/internal/fshelper"
)

// newTestUpCmd returns an UpCmd wired with the collaborators handleAsset needs, and the
// map where the album cache records what it would have sent to the server.
func newTestUpCmd(t *testing.T) (*UpCmd, map[string][]string) {
	t.Helper()

	a := app.New(context.Background(), &cobra.Command{})
	a.SetFileProcessor(fileprocessor.New(
		assettracker.New(),
		fileevent.NewRecorder(slog.New(slog.NewTextHandler(io.Discard, nil))),
	))

	savedAlbums := map[string][]string{}
	uc := &UpCmd{
		app:        a,
		assetIndex: newAssetIndex(),
	}
	uc.albumsCache = cache.NewCollectionCache(50, func(album assets.Album, ids []string) (assets.Album, error) {
		savedAlbums[album.Title] = append(savedAlbums[album.Title], ids...)
		return album, nil
	})
	uc.tagsCache = cache.NewCollectionCache(50, func(tag assets.Tag, ids []string) (assets.Tag, error) {
		return tag, nil
	})
	return uc, savedAlbums
}

// A file that duplicates one already uploaded during the same run must join the albums with the
// ID of the asset it duplicates. Sending an empty ID makes the server reject the whole batch.
func TestHandleAssetAlreadyProcessedJoinsAlbumsWithTheServerID(t *testing.T) {
	const (
		serverID  = "9a2fff7a-f226-48e8-a888-fdac199f3d56"
		albumName = "an album"
	)

	content := []byte("the very same photo, twice")
	fsys := fstest.MapFS{
		"photo.jpg":     &fstest.MapFile{Data: content},
		"photo (1).jpg": &fstest.MapFile{Data: content},
	}

	uc, savedAlbums := newTestUpCmd(t)
	ctx := context.Background()

	uploaded := &assets.Asset{
		ID:               serverID,
		File:             fshelper.FSName(fsys, "photo.jpg"),
		OriginalFileName: "photo.jpg",
	}
	if _, err := uploaded.GetChecksum(); err != nil {
		t.Fatalf("can't compute the checksum of the uploaded asset: %v", err)
	}
	if _, added := uc.assetIndex.addLocalAsset(uploaded); !added {
		t.Fatal("the uploaded asset should have been added to the index")
	}

	duplicate := &assets.Asset{
		File:             fshelper.FSName(fsys, "photo (1).jpg"),
		OriginalFileName: "photo (1).jpg",
		Albums:           []assets.Album{assets.NewAlbum("", albumName, "")},
	}
	uc.app.FileProcessor().RecordAssetDiscovered(ctx, duplicate.File, int64(len(content)), fileevent.DiscoveredImage)

	advice, err := uc.assetIndex.ShouldUpload(duplicate, uc)
	if err != nil {
		t.Fatalf("ShouldUpload: %v", err)
	}
	if advice.Advice != AlreadyProcessed {
		t.Fatalf("expected the advice %v, got %v", AlreadyProcessed, advice.Advice)
	}

	if err := uc.handleAsset(ctx, duplicate); err != nil {
		t.Fatalf("handleAsset: %v", err)
	}
	if duplicate.ID != serverID {
		t.Errorf("expected the asset ID %q, got %q", serverID, duplicate.ID)
	}

	uc.albumsCache.Close() // flush what the cache holds to the save function
	if got := savedAlbums[albumName]; !slices.Equal(got, []string{serverID}) {
		t.Errorf("expected the album to receive %q, got %q", []string{serverID}, got)
	}
}
