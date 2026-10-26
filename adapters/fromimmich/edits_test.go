package fromimmich

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/simulot/immich-go/adapters/folder"
	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/assettracker"
	"github.com/simulot/immich-go/internal/exif/sidecars/jsonsidecar"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/fileprocessor"
	"github.com/simulot/immich-go/internal/filetypes"
	"github.com/simulot/immich-go/internal/fshelper/osfs"
	"github.com/simulot/immich-go/internal/immichfs"
)

// testEditList is GET /assets/{id}/edits for an asset cropped, then rotated: the actions in the
// order Immich applies them, each with its id.
func testEditList(cropID, rotateID string) string {
	return fmt.Sprintf(`{"assetId": %q, "edits": [
		{"id": %q, "action": "crop", "parameters": {"x": 10, "y": 20, "width": 300, "height": 200}},
		{"id": %q, "action": "rotate", "parameters": {"angle": 90}}
	]}`, testAsset, cropID, rotateID)
}

// An edited asset's sidecar carries its edit list: the actions in the order Immich applies them,
// with their parameters, without their ids, and without an isEdited flag. A rerun where the server
// gives the same actions other ids (as it does when the edits are saved again) rewrites nothing.
func TestSidecarCarriesEditList(t *testing.T) {
	t.Setenv("IMMICHGO_TEMPDIR", t.TempDir())
	sum := sha1.Sum([]byte(testOriginal))
	dst := t.TempDir()
	sidecar := filepath.Join(dst, "2021", "2021-07", "IMG_0001.JPG.JSON")
	const want = `[{"action":"crop","parameters":{"x":10,"y":20,"width":300,"height":200}},{"action":"rotate","parameters":{"angle":90}}]`
	var stamp time.Time
	for run, ids := range [][2]string{
		{"dddddddd-0000-4000-8000-000000000001", "dddddddd-0000-4000-8000-000000000002"},
		{"dddddddd-0000-4000-8000-000000000003", "dddddddd-0000-4000-8000-000000000004"},
	} {
		srv := fakeImmich(t, func(asset map[string]any) {
			asset["checksum"] = base64.StdEncoding.EncodeToString(sum[:])
			asset["isEdited"] = true
		})
		srv.editList = testEditList(ids[0], ids[1])
		w, err := folder.NewLocalAssetWriter(osfs.DirFS(dst))
		if err != nil {
			t.Fatal(err)
		}
		a := archivedAsset(t, srv)
		err = w.WriteAsset(context.Background(), a)
		_ = a.Close()
		srv.Close()
		if err != nil {
			t.Fatalf("run %d: %v", run+1, err)
		}
		if n := srv.editCalls.Load(); n != 1 {
			t.Fatalf("run %d: %d calls to GET /edits, want 1", run+1, n)
		}
		b, err := os.ReadFile(sidecar)
		if err != nil {
			t.Fatal(err)
		}
		var md assets.Metadata
		if err := jsonsidecar.Read(bytes.NewReader(b), &md); err != nil {
			t.Fatal(err)
		}
		var got bytes.Buffer
		if err := json.Compact(&got, md.Edits); err != nil || got.String() != want {
			t.Fatalf("run %d: sidecar edits %s (%v), want %s:\n%s", run+1, got.String(), err, want, b)
		}
		if bytes.Contains(b, []byte("isEdited")) {
			t.Fatalf("run %d: the sidecar has an isEdited field:\n%s", run+1, b)
		}
		if run == 0 {
			stamp = time.Now().Add(-time.Hour).Truncate(time.Second)
			if err := os.Chtimes(sidecar, stamp, stamp); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if fi, err := os.Stat(sidecar); err != nil || !fi.ModTime().Equal(stamp) {
			t.Fatalf("run 2 rewrote the sidecar (%v)", err)
		}
	}
}

// GET /edits is called only for an edited asset, and a sidecar has an edit list only when there are
// edits: none for an asset Immich does not report edited, none when the list is empty.
func TestSidecarWithoutEditList(t *testing.T) {
	for _, tc := range []struct {
		name      string
		isEdited  bool
		editCalls int32
	}{
		{"not edited", false, 0},
		{"edited, empty list", true, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := fakeImmich(t, func(asset map[string]any) { asset["isEdited"] = tc.isEdited })
			defer srv.Close()
			srv.editList = `{"assetId": "` + testAsset + `", "edits": []}`
			md := archivedAsset(t, srv).FromApplication
			if n := srv.editCalls.Load(); n != tc.editCalls {
				t.Fatalf("%d calls to GET /edits, want %d", n, tc.editCalls)
			}
			var buf bytes.Buffer
			if err := jsonsidecar.Write(md, &buf); err != nil {
				t.Fatal(err)
			}
			if md.Edits != nil || strings.Contains(buf.String(), `"edits"`) {
				t.Fatalf("the sidecar has an edit list:\n%s", buf.String())
			}
		})
	}
}

// A failed GET /edits is an error for the asset, which is left out: it is never archived without
// its edits. --on-errors decides whether the run goes on.
func TestEditListLookupFails(t *testing.T) {
	for _, tc := range []struct {
		onErrors string
		wantErr  bool
	}{
		{"stop", true},
		{"continue", false},
	} {
		t.Run(tc.onErrors, func(t *testing.T) {
			srv := fakeImmich(t, func(asset map[string]any) { asset["isEdited"] = true })
			defer srv.Close()
			got, a, err := browseAssets(t, srv, tc.onErrors)
			if tc.wantErr != (err != nil) || err != nil && !strings.Contains(err.Error(), "/assets/"+testAsset+"/edits") {
				t.Fatalf("getAssets returned %v, want an error: %v", err, tc.wantErr)
			}
			if len(got) != 0 {
				t.Fatalf("%d assets sent, want none", len(got))
			}
			if n := a.NumErrors(); n != 1 {
				t.Fatalf("%d errors counted, want 1", n)
			}
		})
	}
}

// browseAssets runs getAssets against srv with --on-errors set to onErrors, and returns the assets
// it sends, the application and getAssets' error.
func browseAssets(t *testing.T, srv *fakeServer, onErrors string) ([]*assets.Asset, *app.Application, error) {
	t.Helper()
	ctx := context.Background()
	ic, err := immich.NewImmichClient(srv.URL, "test-key")
	if err != nil {
		t.Fatal(err)
	}
	a := app.New(ctx, nil)
	a.Log().Logger = slog.New(slog.DiscardHandler)
	if err := a.OnErrors.Set(onErrors); err != nil {
		t.Fatal(err)
	}
	fic := &FromImmichCmd{app: a}
	fic.client.Immich = ic
	fic.client.User.ID = testOwner
	fic.processor = fileprocessor.New(assettracker.New(), fileevent.NewRecorder(slog.New(slog.DiscardHandler)))
	fic.ifs = immichfs.NewImmichFS(ctx, srv.URL, ic)
	fic.ic = filenames.NewInfoCollector(time.UTC, filetypes.DefaultSupportedMedia)

	ch := make(chan *assets.Group, 10)
	err = fic.getAssets(ctx, ch)
	close(ch)
	var got []*assets.Asset
	for g := range ch {
		got = append(got, g.Assets...)
	}
	return got, a, err
}
