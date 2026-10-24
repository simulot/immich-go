//go:build e2e

package client

import (
	"testing"

	"github.com/simulot/immich-go/app/root"
	"github.com/simulot/immich-go/immich"
	e2eutils "github.com/simulot/immich-go/internal/e2e/e2eUtils"
	"github.com/simulot/immich-go/internal/fileevent"
)

func Test_FromImmich_IntoAlbum(t *testing.T) {
	adm, err := getUser("admin@immich.app")
	if err != nil {
		t.Fatalf("can't get admin user: %v", err)
	}

	u1, err := createUser("all")
	if err != nil {
		t.Fatalf("can't create user1: %v", err)
	}

	u2, err := createUser("all")
	if err != nil {
		t.Fatalf("can't create user2: %v", err)
	}

	ctx := t.Context()

	// Step 1: upload photos into user1's library
	c, a := root.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"upload", "from-folder",
		"--server=" + ImmichURL,
		"--api-key=" + u1.APIKey,
		"--admin-api-key=" + adm.APIKey,
		"--folder-as-album=FOLDER",
		"--no-ui",
		"--log-level=debug",
		"DATA/fromFolder/recursive",
	})
	err = c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
	}
	if err != nil {
		t.Error("Unexpected error uploading to user1", err)
		return
	}

	// Step 2: transfer from user1 to user2 with --into-album
	c, a = root.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"upload", "from-immich",
		"--server=" + ImmichURL,
		"--api-key=" + u2.APIKey,
		"--admin-api-key=" + adm.APIKey,
		"--from-server=" + ImmichURL,
		"--from-api-key=" + u1.APIKey,
		"--from-admin-api-key=" + adm.APIKey,
		"--into-album=bananas",
		"--no-ui",
		"--log-level=debug",
	})
	err = c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
	}
	if err != nil {
		t.Error("Unexpected error on from-immich transfer", err)
		return
	}

	e2u := a.FileProcessor()

	e2eutils.CheckResults(t, map[fileevent.Code]int64{
		fileevent.ProcessedUploadSuccess: 40,
		fileevent.ProcessedAlbumAdded:    40,
		fileevent.ProcessedTagged:        0,
	}, false, e2u)

	// The destination must contain only the requested album. In particular,
	// source folder albums must not be recreated by the transfer.
	ic, err := immich.NewImmichClient(ImmichURL, u2.APIKey)
	if err != nil {
		t.Fatalf("can't create destination client: %v", err)
	}
	albums, err := ic.GetAllAlbums(ctx)
	if err != nil {
		t.Fatalf("can't get destination albums: %v", err)
	}
	if len(albums) != 1 {
		t.Fatalf("expected exactly one destination album, got %d: %v", len(albums), albums)
	}
	if albums[0].AlbumName != "bananas" {
		t.Errorf("expected destination album bananas, got %q", albums[0].AlbumName)
	}
	ids, err := ic.GetAlbumAssetIDs(ctx, albums[0].ID)
	if err != nil {
		t.Fatalf("can't get destination album assets: %v", err)
	}
	if len(ids) != 40 {
		t.Errorf("expected 40 assets in destination album, got %d", len(ids))
	}
}
