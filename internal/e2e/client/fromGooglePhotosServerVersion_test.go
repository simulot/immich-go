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

const editedPairData = "DATA/fromGooglePhotos/edited-pair/Google Photos/edited pair"

func checksumOf(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha1.Sum(b)
	return base64.StdEncoding.EncodeToString(sum[:])
}

// makeTakeout copies the given files of one edited pair, plus its sidecar, into a
// fresh takeout directory.
func makeTakeout(t *testing.T, pair string, names ...string) string {
	t.Helper()
	dir := t.TempDir()
	dst := filepath.Join(dir, "Takeout", "Google Photos", "Pair")
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	files := append([]string{pair + ".jpg.supplemental-metadata.json"}, names...)
	for _, name := range files {
		b, err := os.ReadFile(filepath.Join(editedPairData, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(dst, name), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func runTakeout(t *testing.T, dir string, u1, adm user, want map[fileevent.Code]int64) {
	t.Helper()
	ctx := t.Context()
	c, a := root.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"upload", "from-google-photos",
		"--server=" + ImmichURL,
		"--api-key=" + u1.APIKey,
		"--admin-api-key=" + adm.APIKey,
		"--no-ui",
		"--concurrent-tasks=1",
		"--manage-burst=Stack",
		dir,
	})
	err := c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
	}
	if err != nil {
		t.Fatalf("upload %s: %v", dir, err)
	}
	e2eutils.CheckResults(t, want, false, a.FileProcessor())
}

// When the server already holds a version of a photo and a takeout brings the
// edited version, both versions must end up on the server, in one stack, with the
// edit as the stack primary. The server version must never be replaced.
func Test_FromGooglePhotos_ServerVersion(t *testing.T) {
	adm, err := getUser("admin@immich.app")
	if err != nil {
		t.Fatalf("can't get admin user: %v", err)
	}

	for _, tc := range []struct {
		name        string
		pair        string
		server      []string
		takeout     []string
		wantStacked int64
	}{
		{"edit only over server original", "pair_a1", []string{"pair_a1.jpg"}, []string{"pair_a1-edited.jpg"}, 1},
		{"full pair over server original, original bigger", "pair_a1", []string{"pair_a1.jpg"}, []string{"pair_a1.jpg", "pair_a1-edited.jpg"}, 2},
		{"full pair over server original, edit bigger", "pair_b1", []string{"pair_b1.jpg"}, []string{"pair_b1.jpg", "pair_b1-edited.jpg"}, 2},
		{"full pair over server edit, original bigger", "pair_a1", []string{"pair_a1-edited.jpg"}, []string{"pair_a1.jpg", "pair_a1-edited.jpg"}, 2},
		{"full pair over server edit, edit bigger", "pair_b1", []string{"pair_b1-edited.jpg"}, []string{"pair_b1.jpg", "pair_b1-edited.jpg"}, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u1, err := createUser("minimal")
			if err != nil {
				t.Fatalf("can't create user: %v", err)
			}

			// seed the server with one version of the pair
			serverDir := makeTakeout(t, tc.pair, tc.server...)
			runTakeout(t, serverDir, u1, adm, map[fileevent.Code]int64{
				fileevent.ProcessedUploadSuccess: int64(len(tc.server)),
			})

			// import the takeout
			takeoutDir := makeTakeout(t, tc.pair, tc.takeout...)
			runTakeout(t, takeoutDir, u1, adm, map[fileevent.Code]int64{
				fileevent.ProcessedUploadSuccess:  1,
				fileevent.ProcessedUploadUpgraded: 0,
				fileevent.ProcessedStacked:        tc.wantStacked,
			})

			assets, err := e2eutils.GetAllAssetList(u1.Email, u1.Password)
			if err != nil {
				t.Fatal(err)
			}
			if len(assets) != 2 {
				for _, a := range assets {
					t.Logf("asset %s %s trashed=%v", a.ID, a.OriginalFileName, a.IsTrashed)
				}
				t.Fatalf("got %d assets on the server, want 2 (both versions kept)", len(assets))
			}

			editedChecksum := checksumOf(t, filepath.Join(editedPairData, tc.pair+"-edited.jpg"))
			stacks, err := e2eutils.GetAllStacks(u1.Email, u1.Password)
			if err != nil {
				t.Fatal(err)
			}
			if len(stacks) != 1 {
				t.Fatalf("got %d stacks, want 1", len(stacks))
			}
			if len(stacks[0].Assets) != 2 {
				t.Fatalf("stack has %d assets, want 2", len(stacks[0].Assets))
			}
			primaryFound := false
			for _, a := range stacks[0].Assets {
				if a.ID != stacks[0].PrimaryAssetID {
					continue
				}
				primaryFound = true
				if a.Checksum != editedChecksum {
					t.Errorf("stack primary is %s (%s), want the edited version", a.OriginalFileName, a.Checksum)
				}
			}
			if !primaryFound {
				t.Errorf("stack primary %s is not among its assets", stacks[0].PrimaryAssetID)
			}
		})
	}
}

// Re-importing only the edited file must not split the stack that already holds the
// original and the edit: the server keeps one stack of two, with the edit as primary.
func Test_FromGooglePhotos_ServerVersion_NoSplit(t *testing.T) {
	adm, err := getUser("admin@immich.app")
	if err != nil {
		t.Fatalf("can't get admin user: %v", err)
	}
	u1, err := createUser("minimal")
	if err != nil {
		t.Fatalf("can't create user: %v", err)
	}

	// first import: the full pair
	takeoutDir := makeTakeout(t, "pair_a1", "pair_a1.jpg", "pair_a1-edited.jpg")
	runTakeout(t, takeoutDir, u1, adm, map[fileevent.Code]int64{
		fileevent.ProcessedUploadSuccess: 2,
		fileevent.ProcessedStacked:       2,
	})

	// second import: the edited file alone
	secondDir := makeTakeout(t, "pair_a1", "pair_a1-edited.jpg")
	runTakeout(t, secondDir, u1, adm, map[fileevent.Code]int64{
		fileevent.ProcessedUploadSuccess: 0,
		fileevent.ProcessedStacked:       0, // the base is already in the edit's stack: nothing to merge
	})

	serverAssets, err := e2eutils.GetAllAssetList(u1.Email, u1.Password)
	if err != nil {
		t.Fatal(err)
	}
	if len(serverAssets) != 2 {
		for _, a := range serverAssets {
			t.Logf("asset %s %s trashed=%v", a.ID, a.OriginalFileName, a.IsTrashed)
		}
		t.Fatalf("got %d assets on the server, want 2", len(serverAssets))
	}

	stacks, err := e2eutils.GetAllStacks(u1.Email, u1.Password)
	if err != nil {
		t.Fatal(err)
	}
	if len(stacks) != 1 {
		for _, s := range stacks {
			t.Logf("stack %s primary=%s assets=%d", s.ID, s.PrimaryAssetID, len(s.Assets))
		}
		t.Fatalf("got %d stacks, want 1: the second import must not split the stack", len(stacks))
	}
	if len(stacks[0].Assets) != 2 {
		t.Fatalf("stack has %d assets, want 2", len(stacks[0].Assets))
	}

	editedChecksum := checksumOf(t, filepath.Join(editedPairData, "pair_a1-edited.jpg"))
	primaryFound := false
	for _, a := range stacks[0].Assets {
		if a.ID != stacks[0].PrimaryAssetID {
			continue
		}
		primaryFound = true
		if a.Checksum != editedChecksum {
			t.Errorf("stack primary is %s (%s), want the edited version", a.OriginalFileName, a.Checksum)
		}
	}
	if !primaryFound {
		t.Errorf("stack primary %s is not among its assets", stacks[0].PrimaryAssetID)
	}
}
