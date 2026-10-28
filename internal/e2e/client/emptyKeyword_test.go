//go:build e2e

package client

import (
	"testing"

	"github.com/simulot/immich-go/app/root"
	"github.com/simulot/immich-go/immich"
	e2eutils "github.com/simulot/immich-go/internal/e2e/e2eUtils"
	"github.com/simulot/immich-go/internal/fileevent"
)

// Issue #1451: an XMP sidecar with an empty keyword (osxphotos template
// "{favorite?Favorit,}" on a non-favorite photo) must not create a tag.
// Before the fix, the empty tag reached UpsertTags, the server answered with an
// empty list, and saveTags panicked on r[0].
func Test_EmptyKeywordSidecarTag(t *testing.T) {
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
		"upload", "from-folder",
		"--server=" + ImmichURL,
		"--api-key=" + u1.APIKey,
		"--admin-api-key=" + adm.APIKey,
		"--no-ui",
		"--log-level=debug",
		"DATA/fromFolder/empty-keyword",
	})
	err = c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
	}
	if err != nil {
		t.Error("Unexpected error", err)
		return
	}

	// Only the non-empty keyword reaches the server.
	e2eutils.CheckResults(t, map[fileevent.Code]int64{
		fileevent.ProcessedUploadSuccess: 1,
		fileevent.ProcessedTagged:        1,
	}, false, a.FileProcessor())

	ic, err := immich.NewImmichClient(ImmichURL, u1.APIKey)
	if err != nil {
		t.Fatalf("can't create the client: %v", err)
	}
	serverTags, err := ic.GetAllTags(ctx)
	if err != nil {
		t.Fatalf("can't get the user's tags: %v", err)
	}
	// The empty keyword must not create a tag on the server.
	if len(serverTags) != 1 || serverTags[0].Name != "favorit" {
		t.Errorf("expected exactly one tag [favorit] for the user, got %v", serverTags)
	}
}
