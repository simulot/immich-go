//go:build e2e

package client

import (
	"io/fs"
	"maps"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	e2eutils "github.com/simulot/immich-go/internal/e2e/e2eUtils"
)

// TestArchiveRestoreStacks makes three stacks on an account: one whose primary is not its first
// asset by name, one with an archived member, and one whose assets are archived in two month
// folders. It archives the account and restores the archive into a new account with
// `upload from-folder`. The same stacks come back, with the same primaries, and no others. Running
// the upload again makes no stack.
func TestArchiveRestoreStacks(t *testing.T) {
	const assetCount = 20
	adm, u1 := e2eAdmin(t), e2eNewUser(t)
	uploadFromFolder(t, u1, adm, "DATA/fromFolder/recursive/a/b")
	checksums := serverChecksums(t, u1, assetCount)
	tk := login(t, u1)
	ids := assetIDs(t, tk)

	// mountains_01 and mountains_02 are taken in two months, so they are archived in two folders.
	months := map[string]time.Time{
		"mountains_01.jpg": time.Date(2020, 1, 15, 12, 0, 0, 0, time.UTC),
		"mountains_02.jpg": time.Date(2020, 2, 15, 12, 0, 0, 0, time.UTC),
	}
	for name, d := range months {
		updateAsset(t, tk, ids[name], map[string]any{"dateTimeOriginal": d.Format(time.RFC3339)})
	}
	// Each stack's assets by name, its primary first.
	stacks := [][]string{
		{"horses_03.jpg", "horses_01.jpg", "horses_02.jpg"}, // the primary is not the first by name
		{"horses_05.jpg", "horses_06.jpg"},                  // horses_06 is archived below
		{"mountains_01.jpg", "mountains_02.jpg"},            // in two month folders
	}
	want := map[string][]string{}
	for _, names := range stacks {
		members := make([]string, 0, len(names))
		memberIDs := make([]string, 0, len(names))
		for _, name := range names {
			members = append(members, checksums[name])
			memberIDs = append(memberIDs, ids[name])
		}
		makeStack(t, tk, memberIDs)
		slices.Sort(members)
		want[checksums[names[0]]] = members
	}
	// With its only other member archived, GET /assets/{id} does not report the stack: only
	// GET /stacks does.
	updateAsset(t, tk, ids["horses_06.jpg"], map[string]any{"visibility": "archive"})
	waitForCaptureDates(t, tk, months)
	if got, _ := userStacks(t, u1); !maps.EqualFunc(got, want, slices.Equal[[]string]) {
		t.Fatalf("stacks before the archive: %v, want %v", got, want)
	}

	dir := t.TempDir()
	if _, err := archiveFromImmich(t.Context(), ImmichURL, u1, adm, dir); err != nil {
		t.Fatalf("archive: %v", err)
	}
	archived := archivedPaths(t, dir, checksums)
	for name, d := range months {
		folder, err := filepath.Rel(dir, filepath.Dir(archived[name]))
		if wantFolder := d.Format("2006/2006-01"); err != nil || filepath.ToSlash(folder) != wantFolder {
			t.Fatalf("%s: archived in %q (%v), want %s", name, folder, err, wantFolder)
		}
	}

	u2 := e2eNewUser(t)
	uploadFromFolder(t, u2, adm, dir)
	got, stackIDs := userStacks(t, u2)
	if !maps.EqualFunc(got, want, slices.Equal[[]string]) {
		t.Errorf("restored stacks: %v, want %v", got, want)
	}

	uploadFromFolder(t, u2, adm, dir)
	if got, again := userStacks(t, u2); !maps.EqualFunc(got, want, slices.Equal[[]string]) || !slices.Equal(again, stackIDs) {
		t.Errorf("stacks after the upload ran again: %v with IDs %v, want %v with IDs %v", got, again, want, stackIDs)
	}
}

// archivedPaths walks the archive in dir and returns the path of each archived asset by its name.
// It fails the test on a file that is no asset's (a ~N copy), an asset archived twice, and content
// whose SHA-1 is not the asset's checksum.
func archivedPaths(t *testing.T, dir string, checksums map[string]string) map[string]string {
	t.Helper()
	archived := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		name := d.Name()
		switch {
		case strings.HasSuffix(name, ".JSON"):
			// the asset's sidecar
		case checksums[name] == "":
			t.Errorf("%s: no asset has this name", p)
		case archived[name] != "":
			t.Errorf("%s: %s is already archived as %s", p, name, archived[name])
		default:
			archived[name] = p
			if sum, err := computeSHA1(p); err != nil || sum != checksums[name] {
				t.Errorf("%s: SHA-1 %q (%v), want the asset's checksum %s", p, sum, err, checksums[name])
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return archived
}

// assetIDs returns the IDs of the user's assets by file name.
func assetIDs(t *testing.T, tk e2eutils.Token) map[string]string {
	t.Helper()
	ids := map[string]string{}
	for _, a := range searchAssets(t, tk, map[string]any{}) {
		if _, ok := ids[a.OriginalFileName]; ok {
			t.Fatalf("two assets are named %s", a.OriginalFileName)
		}
		ids[a.OriginalFileName] = a.ID
	}
	return ids
}

// updateAsset sets the asset's fields with PUT /assets/{id}.
func updateAsset(t *testing.T, tk e2eutils.Token, id string, fields map[string]any) {
	t.Helper()
	callAPI(t, tk, http.MethodPut, "/assets/"+id, fields, &struct{}{})
}

// makeStack stacks the assets, the first one its primary.
func makeStack(t *testing.T, tk e2eutils.Token, ids []string) {
	t.Helper()
	var st struct {
		ID string `json:"id"`
	}
	callAPI(t, tk, http.MethodPost, "/stacks", map[string]any{"assetIds": ids}, &st)
	if st.ID == "" {
		t.Fatalf("POST /stacks %v: no stack ID", ids)
	}
}

// waitForCaptureDates waits until the server reports the capture date of each asset, given by
// name.
func waitForCaptureDates(t *testing.T, tk e2eutils.Token, dates map[string]time.Time) {
	t.Helper()
	deadline := time.Now().Add(time.Minute)
	for {
		done := 0
		for _, a := range searchAssets(t, tk, map[string]any{"withExif": true}) {
			d, ok := dates[a.OriginalFileName]
			if !ok || a.ExifInfo == nil || a.ExifInfo.DateTimeOriginal == nil {
				continue
			}
			if got, err := time.Parse(time.RFC3339, *a.ExifInfo.DateTimeOriginal); err == nil && got.Equal(d) {
				done++
			}
		}
		if done == len(dates) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("%d of the capture dates %v set after a minute", done, dates)
		}
		time.Sleep(time.Second)
	}
}

// userStacks returns the user's stacks by the checksum of their primary asset, each with the
// sorted checksums of its assets, the primary included, and the sorted IDs of the stacks.
func userStacks(t *testing.T, u user) (map[string][]string, []string) {
	t.Helper()
	var stacks []struct {
		ID             string        `json:"id"`
		PrimaryAssetID string        `json:"primaryAssetId"`
		Assets         []serverAsset `json:"assets"`
	}
	callAPI(t, login(t, u), http.MethodGet, "/stacks", nil, &stacks)
	members := map[string][]string{}
	ids := make([]string, 0, len(stacks))
	for _, st := range stacks {
		var primary string
		checksums := make([]string, 0, len(st.Assets))
		for _, a := range st.Assets {
			if a.ID == st.PrimaryAssetID {
				primary = a.Checksum
			}
			checksums = append(checksums, a.Checksum)
		}
		if primary == "" {
			t.Fatalf("stack %s: its primary %s is not among its assets", st.ID, st.PrimaryAssetID)
		}
		if _, ok := members[primary]; ok {
			t.Fatalf("two stacks have the primary %s", primary)
		}
		slices.Sort(checksums)
		members[primary] = checksums
		ids = append(ids, st.ID)
	}
	slices.Sort(ids)
	return members, ids
}
