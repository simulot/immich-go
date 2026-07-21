//go:build e2e

package client

import (
	"testing"

	"github.com/simulot/immich-go/app/root"
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
}
