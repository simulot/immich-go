//go:build e2e

package client

import (
	"context"
	"testing"

	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/app/root"
	e2eutils "github.com/simulot/immich-go/internal/e2e/e2eUtils"
	"github.com/simulot/immich-go/internal/fileevent"
)

func Test_GetImmichAlbums(t *testing.T) {
	adm, err := getUser("admin@immich.app")
	if err != nil {
		t.Fatalf("can't get admin user: %v", err)
	}
	// A fresh user so the album namespace is clean.
	u1, err := createUser("minimal")
	if err != nil {
		t.Fatalf("can't create user: %v", err)
	}

	const albumName = "getImmichAlbums"
	const expectedAssets = 40

	// uploadIntoAlbum runs `upload from-folder --into-album=<albumName>` and
	// returns the application so the caller can inspect the file processor.
	uploadIntoAlbum := func(ctx context.Context) *app.Application {
		t.Helper()
		c, a := root.RootImmichGoCommand(ctx)
		c.SetArgs([]string{
			"upload", "from-folder",
			"--server=" + ImmichURL,
			"--api-key=" + u1.APIKey,
			"--admin-api-key=" + adm.APIKey,
			"--into-album=" + albumName,
			"--no-ui",
			"--api-trace",
			"--log-level=debug",
			"DATA/fromFolder/recursive",
		})
		if err := c.ExecuteContext(ctx); err != nil {
			if a.Log().GetSLog() != nil {
				a.Log().Error(err.Error())
			}
			t.Fatalf("unexpected error during upload: %v", err)
		}
		return a
	}

	// First run: every asset is uploaded and added to the album.
	a1 := uploadIntoAlbum(t.Context())
	e2eutils.CheckResults(t, map[fileevent.Code]int64{
		fileevent.ProcessedUploadSuccess: expectedAssets,
		fileevent.ProcessedAlbumAdded:    expectedAssets,
	}, false, a1.FileProcessor())

	// Second run: assets already exist on the server and already belong to the
	// album. getImmichAlbums must surface that membership so nothing is
	// re-uploaded and nothing is re-added to the album.
	a2 := uploadIntoAlbum(t.Context())
	e2eutils.CheckResults(t, map[fileevent.Code]int64{
		fileevent.ProcessedUploadSuccess:   0,
		fileevent.ProcessedAlbumAdded:      0,
		fileevent.DiscardedServerDuplicate: expectedAssets,
	}, false, a2.FileProcessor())
}
