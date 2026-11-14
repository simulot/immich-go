//go:build e2e

package client

import (
	"testing"
	"time"

	"github.com/simulot/immich-go/app/root"
	e2eutils "github.com/simulot/immich-go/internal/e2e/e2eUtils"
	"github.com/simulot/immich-go/internal/fileevent"
)

// The server already has a small copy of photo.jpg (uploaded from DATA/fromGooglePhotos/replaced-copy/before).
// The takeout (.../takeout) then has a bigger photo.jpg with the same date, its Google-edited version,
// and photo(1).jpg, Google's duplicate, byte-identical to the small copy on the server.
//
// The bigger photo.jpg replaces the server's copy, which is deleted. photo(1).jpg has the content of
// that deleted asset, and must be treated as a copy of the replacement, not matched to the deleted
// asset: before the fix it got the deleted asset's ID, and the group's stack was requested with it.
func Test_FromGooglePhotos_ReplacedCopy(t *testing.T) {
	adm, err := getUser("admin@immich.app")
	if err != nil {
		t.Fatalf("can't get admin user: %v", err)
	}
	u1, err := createUser("minimal")
	if err != nil {
		t.Fatalf("can't create user: %v", err)
	}

	ctx := t.Context()
	run := func(dir string, want map[fileevent.Code]int64) {
		t.Helper()
		c, a := root.RootImmichGoCommand(ctx)
		c.SetArgs([]string{
			"upload", "from-google-photos",
			"--server=" + ImmichURL,
			"--api-key=" + u1.APIKey,
			"--admin-api-key=" + adm.APIKey,
			"--no-ui",
			"--api-trace",
			"--log-level=debug",
			"--manage-burst=Stack",
			dir,
		})
		err := c.ExecuteContext(ctx)
		if err != nil && a.Log().GetSLog() != nil {
			a.Log().Error(err.Error())
		}
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", dir, err)
		}
		e2eutils.CheckResults(t, want, false, a.FileProcessor())
	}

	run("DATA/fromGooglePhotos/replaced-copy/before", map[fileevent.Code]int64{
		fileevent.ProcessedUploadSuccess: 1,
	})
	time.Sleep(2 * time.Second) // let the server settle before the replacement

	run("DATA/fromGooglePhotos/replaced-copy/takeout", map[fileevent.Code]int64{
		fileevent.ProcessedUploadSuccess:  1, // photo-edited.jpg
		fileevent.ProcessedUploadUpgraded: 1, // photo.jpg replaces the small copy
		fileevent.ProcessedStacked:        3, // all three are covered by the stack; photo(1).jpg through photo.jpg, which stands for it
	})

	assets, err := e2eutils.GetAllAssetList(u1.Email, u1.Password)
	if err != nil {
		t.Fatal(err)
	}
	if len(assets) != 2 {
		for _, a := range assets {
			t.Logf("asset %s %s trashed=%v", a.ID, a.OriginalFileName, a.IsTrashed)
		}
		t.Errorf("got %d assets on the server, want 2 (the bigger photo.jpg and its edited version)", len(assets))
	}

	stacks, err := e2eutils.GetAllStacks(u1.Email, u1.Password)
	if err != nil {
		t.Fatal(err)
	}
	if len(stacks) != 1 || len(stacks[0].Assets) != 2 {
		t.Errorf("got %d stacks (%v), want one stack of 2 assets", len(stacks), stacks)
	}
}
