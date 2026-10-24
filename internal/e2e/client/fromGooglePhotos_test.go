//go:build e2e

package client

import (
	"testing"

	"github.com/simulot/immich-go/app/root"
	"github.com/simulot/immich-go/immich"
	e2eutils "github.com/simulot/immich-go/internal/e2e/e2eUtils"
	"github.com/simulot/immich-go/internal/fileevent"
)

func Test_FromGooglePhotos(t *testing.T) {
	adm, err := getUser("admin@immich.app")
	if err != nil {
		t.Fatalf("can't get admin user: %v", err)
	}
	// A fresh user for a new test
	u1, err := createUser("minimal")
	if err != nil {
		t.Fatalf("can't create user: %v", err)
	}

	ctx := t.Context()
	c, a := root.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"upload", "from-google-photos",
		"--server=" + ImmichURL,
		"--api-key=" + u1.APIKey,
		"--admin-api-key=" + adm.APIKey,
		"--no-ui",
		// "--api-trace",
		"--log-level=debug",
		"DATA/fromGooglePhotos/gophers",
	})
	err = c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
	}

	if err != nil {
		t.Error("Unexpected error", err)
		return
	}

	e2eutils.CheckResults(t, map[fileevent.Code]int64{
		fileevent.ProcessedUploadSuccess: 5,
		fileevent.ProcessedAlbumAdded:    5,
		fileevent.ProcessedTagged:        5,
	}, false, a.FileProcessor())
}

func Test_FromGooglePhotos_IntoAlbum(t *testing.T) {
	adm, err := getUser("admin@immich.app")
	if err != nil {
		t.Fatalf("can't get admin user: %v", err)
	}
	u1, err := createUser("minimal")
	if err != nil {
		t.Fatalf("can't create user: %v", err)
	}

	ctx := t.Context()
	c, a := root.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"upload", "from-google-photos",
		"--server=" + ImmichURL,
		"--api-key=" + u1.APIKey,
		"--admin-api-key=" + adm.APIKey,
		"--into-album=bananas",
		"--no-ui",
		"--log-level=debug",
		"DATA/fromGooglePhotos/gophers",
	})
	if err := c.ExecuteContext(ctx); err != nil {
		if a.Log().GetSLog() != nil {
			a.Log().Error(err.Error())
		}
		t.Fatalf("unexpected error: %v", err)
	}

	e2eutils.CheckResults(t, map[fileevent.Code]int64{
		fileevent.ProcessedUploadSuccess: 5,
		fileevent.ProcessedAlbumAdded:    5,
		fileevent.ProcessedTagged:        5,
	}, false, a.FileProcessor())

	ic, err := immich.NewImmichClient(ImmichURL, u1.APIKey)
	if err != nil {
		t.Fatalf("can't create client: %v", err)
	}
	albums, err := ic.GetAllAlbums(ctx)
	if err != nil {
		t.Fatalf("can't get albums: %v", err)
	}
	if len(albums) != 1 {
		t.Fatalf("expected exactly one album, got %d: %v", len(albums), albums)
	}
	if albums[0].AlbumName != "bananas" {
		t.Errorf("expected album bananas, got %q", albums[0].AlbumName)
	}
	ids, err := ic.GetAlbumAssetIDs(ctx, albums[0].ID)
	if err != nil {
		t.Fatalf("can't get album assets: %v", err)
	}
	if len(ids) != 5 {
		t.Errorf("expected 5 assets in album, got %d", len(ids))
	}
}
