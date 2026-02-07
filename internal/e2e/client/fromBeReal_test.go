//go:build e2e

package client

import (
	"testing"

	"github.com/simulot/immich-go/app/root"
	e2eutils "github.com/simulot/immich-go/internal/e2e/e2eUtils"
	"github.com/simulot/immich-go/internal/fileevent"
)

func Test_FromBeReal(t *testing.T) {
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
		"upload", "from-bereal",
		"--server=" + ImmichURL,
		"--api-key=" + u1.APIKey,
		"--admin-api-key=" + adm.APIKey,
		"--no-ui",
		"--log-level=debug",
		ProjectDir + "/adapters/bereal/DATA",
	})
	err = c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
	}

	if err != nil {
		t.Error("Unexpected error", err)
		return
	}

	// Expecting front+back -> 2 assets; tags, album, and stacking per asset
	// Verify that upload, album, tagging, and stacking events occurred
	e2eutils.CheckResults(t, map[fileevent.Code]int64{
		fileevent.ProcessedUploadSuccess: 2,
		fileevent.ProcessedAlbumAdded:    2,
		fileevent.ProcessedTagged:        2,
		fileevent.ProcessedStacked:       2, // Both assets should be stacked
	}, false, a.FileProcessor())
}
