//go:build e2e

package client

import (
	"crypto/sha1"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/simulot/immich-go/app/root"
	e2eutils "github.com/simulot/immich-go/internal/e2e/e2eUtils"
	"github.com/simulot/immich-go/internal/fileevent"
)

// A Google Photos takeout with eight photos and their Google-edited versions, one sidecar per pair:
//
//	pair_aN.jpg (bigger than pair_aN-edited.jpg), pair_aN-edited.jpg, pair_aN.jpg.supplemental-metadata.json
//	pair_bN.jpg (smaller than pair_bN-edited.jpg), pair_bN-edited.jpg, pair_bN.jpg.supplemental-metadata.json
//
// All sixteen images must be uploaded and each pair stacked, whichever of a pair is processed
// first. Before the fix for #1285/#877, an edited version processed first was taken for a
// smaller/bigger server copy of the original: the original was then either not uploaded, or
// uploaded as a replacement that deleted the edited version. Which of a pair comes first is
// random (map iteration order in the adapter), hence the eight pairs: without the fix, all
// eight coming out right has probability 1/256.
func Test_FromGooglePhotos_EditedPair(t *testing.T) {
	adm, err := getUser("admin@immich.app")
	if err != nil {
		t.Fatalf("can't get admin user: %v", err)
	}
	u1, err := createUser("minimal")
	if err != nil {
		t.Fatalf("can't create user: %v", err)
	}

	const dir = "DATA/fromGooglePhotos/edited-pair"
	want := map[string]string{} // checksum -> file name
	images, err := filepath.Glob(filepath.Join(dir, "Google Photos", "edited pair", "*.jpg"))
	if err != nil {
		t.Fatal(err)
	}
	for _, image := range images {
		b, err := os.ReadFile(image)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha1.Sum(b)
		want[base64.StdEncoding.EncodeToString(sum[:])] = filepath.Base(image)
	}
	const pairs = 8
	if len(images) != 2*pairs {
		t.Fatalf("got %d images in the fixture, want %d", len(images), 2*pairs)
	}

	ctx := t.Context()
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
	err = c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
	}
	if err != nil {
		t.Error("Unexpected error", err)
		return
	}

	e2eutils.CheckResults(t, map[fileevent.Code]int64{
		fileevent.ProcessedUploadSuccess:  2 * pairs,
		fileevent.ProcessedUploadUpgraded: 0,
		fileevent.ProcessedStacked:        2 * pairs,
		fileevent.DiscardedLocalDuplicate: 0,
	}, false, a.FileProcessor())

	assets, err := e2eutils.GetAllAssetList(u1.Email, u1.Password)
	if err != nil {
		t.Fatal("Unexpected error", err)
	}

	for _, asset := range assets {
		name, ok := want[asset.Checksum]
		if !ok {
			t.Errorf("unexpected asset on the server: %s (%s)", asset.OriginalFileName, asset.ID)
			continue
		}
		delete(want, asset.Checksum)
		if asset.IsTrashed {
			t.Errorf("%s is trashed", name)
		}
	}
	for _, name := range want {
		t.Errorf("%s is missing from the server", name)
	}

	stacks, err := e2eutils.GetAllStacks(u1.Email, u1.Password)
	if err != nil {
		t.Fatal("Unexpected error", err)
	}
	if len(stacks) != pairs {
		t.Errorf("got %d stacks, want %d", len(stacks), pairs)
	}
	for _, s := range stacks {
		if len(s.Assets) != 2 {
			t.Errorf("stack %s has %d assets, want 2", s.ID, len(s.Assets))
		}
	}
}
