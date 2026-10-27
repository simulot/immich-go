package fromimmich

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
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
	"github.com/spf13/cobra"
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

// A failed GET /edits, other than refused, is an error for the asset, which is left out: it is never
// archived without its edits. --on-errors decides whether the run goes on. Whatever it says, the
// failure is logged as an error that names the asset, and recorded as a server error: under
// --on-errors=N, ProcessError logs nothing until the limit is passed.
func TestEditListLookupFails(t *testing.T) {
	for _, tc := range []struct {
		onErrors string
		wantErr  bool
	}{
		{"stop", true},
		{"continue", false},
		{"3", false},
	} {
		t.Run(tc.onErrors, func(t *testing.T) {
			srv := fakeImmich(t, func(asset map[string]any) { asset["isEdited"] = true })
			defer srv.Close()
			srv.editErr = http.StatusInternalServerError
			var log bytes.Buffer
			got, a, err := browseAssets(t, srv.URL, "archive", tc.onErrors, &log)
			if tc.wantErr != (err != nil) || err != nil && !strings.Contains(err.Error(), "/assets/"+testAsset+"/edits") {
				t.Fatalf("getAssets returned %v, want an error: %v", err, tc.wantErr)
			}
			if len(got) != 0 {
				t.Fatalf("%d assets sent, want none", len(got))
			}
			if n := a.NumErrors(); n != 1 {
				t.Fatalf("%d errors counted, want 1", n)
			}
			if n := a.FileProcessor().GetEventCounts()[fileevent.ErrorServerError]; n != 1 {
				t.Errorf("%d server errors recorded, want 1", n)
			}
			logged := false
			for line := range strings.Lines(log.String()) {
				logged = logged || strings.Contains(line, "level=ERROR") && strings.Contains(line, "the edits of asset "+testAsset)
			}
			if !logged {
				t.Errorf("no error logged for the edits of asset %s:\n%s", testAsset, log.String())
			}
		})
	}
}

// A key without asset.edit.get does not stop an archive. GET /edits answers 403 once, and is not
// called again in the run; every asset is sent on, without an edit list; the run logs one warning,
// with the count, and counts no error.
func TestEditListRefused(t *testing.T) {
	t.Setenv("IMMICHGO_TEMPDIR", t.TempDir())
	const n = 3
	srv, editCalls := fakeEditedImmich(t, n)
	var log bytes.Buffer
	got, a, err := browseAssets(t, srv.URL, "archive", "stop", &log)
	if err != nil {
		t.Fatalf("getAssets: %v", err)
	}
	if c := editCalls.Load(); c != 1 {
		t.Fatalf("%d calls to GET /edits, want 1", c)
	}
	if e := a.NumErrors(); e != 0 {
		t.Fatalf("%d errors counted, want 0", e)
	}
	var warnings []string
	for line := range strings.Lines(log.String()) {
		if strings.Contains(line, "asset.edit.get") {
			warnings = append(warnings, line)
		}
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "level=WARN") || !strings.Contains(warnings[0], fmt.Sprintf("count=%d", n)) {
		t.Fatalf("log lines about asset.edit.get %q, want one warning with count=%d:\n%s", warnings, n, log.String())
	}

	if len(got) != n {
		t.Fatalf("%d assets sent, want %d", len(got), n)
	}
	dst := t.TempDir()
	w, err := folder.NewLocalAssetWriter(osfs.DirFS(dst))
	if err != nil {
		t.Fatal(err)
	}
	for _, as := range got {
		err := w.WriteAsset(context.Background(), as)
		_ = as.Close()
		if err != nil {
			t.Fatalf("%s: %v", as.OriginalFileName, err)
		}
	}
	for i := 1; i <= n; i++ {
		p := filepath.Join(dst, "2021", "2021-07", fmt.Sprintf("IMG_%d.JPG", i))
		if b, err := os.ReadFile(p); err != nil || string(b) != fmt.Sprintf("photo %d", i) {
			t.Fatalf("%s holds %q (%v)", p, b, err)
		}
		if b, err := os.ReadFile(p + ".JSON"); err != nil || bytes.Contains(b, []byte(`"edits"`)) {
			t.Fatalf("%s.JSON (%v):\n%s", p, err, b)
		}
	}
}

// An edit list that could not be read is not an empty one: a rerun whose key lacks asset.edit.get
// keeps the list the sidecar has, and does not rewrite it.
func TestEditListRefusedKeepsSidecarEdits(t *testing.T) {
	t.Setenv("IMMICHGO_TEMPDIR", t.TempDir())
	sum := sha1.Sum([]byte(testOriginal))
	dst := t.TempDir()
	sidecar := filepath.Join(dst, "2021", "2021-07", "IMG_0001.JPG.JSON")
	var first []byte
	stamp := time.Now().Add(-time.Hour).Truncate(time.Second)
	for run, refused := range []bool{false, true} {
		srv := fakeImmich(t, func(asset map[string]any) {
			asset["checksum"] = base64.StdEncoding.EncodeToString(sum[:])
			asset["isEdited"] = true
		})
		srv.editList = testEditList("dddddddd-0000-4000-8000-000000000001", "dddddddd-0000-4000-8000-000000000002")
		if refused {
			srv.editErr = http.StatusForbidden
		}
		got, _, err := browseAssets(t, srv.URL, "archive", "stop", io.Discard)
		if err != nil || len(got) != 1 {
			t.Fatalf("run %d: %d assets sent (%v), want 1", run+1, len(got), err)
		}
		w, err := folder.NewLocalAssetWriter(osfs.DirFS(dst))
		if err != nil {
			t.Fatal(err)
		}
		err = w.WriteAsset(context.Background(), got[0])
		_ = got[0].Close()
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
		if run == 0 {
			if !bytes.Contains(b, []byte(`"action": "crop"`)) {
				t.Fatalf("run 1: the sidecar has no edit list:\n%s", b)
			}
			first = b
			if err := os.Chtimes(sidecar, stamp, stamp); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if !bytes.Equal(b, first) {
			t.Fatalf("run 2 changed the sidecar:\n%s\nwas:\n%s", b, first)
		}
		if fi, err := os.Stat(sidecar); err != nil || !fi.ModTime().Equal(stamp) {
			t.Fatalf("run 2 rewrote the sidecar (%v)", err)
		}
	}
}

// fakeEditedImmich answers getAssets and the downloads for n edited photos on the timeline,
// IMG_1.JPG to IMG_n.JPG, whose originals are "photo 1" to "photo n". It refuses GET /edits as
// Immich does for a key without asset.edit.get, and counts those calls.
func fakeEditedImmich(t *testing.T, n int) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	editCalls := &atomic.Int32{}
	byID := map[string]map[string]any{}
	originals := map[string]string{}
	items := make([]any, 0, n)
	for i := 1; i <= n; i++ {
		id := fmt.Sprintf("eeeeeeee-0000-4000-8000-%012d", i)
		original := fmt.Sprintf("photo %d", i)
		sum := sha1.Sum([]byte(original))
		asset := map[string]any{
			"id": id, "ownerId": testOwner, "type": "IMAGE", "originalFileName": fmt.Sprintf("IMG_%d.JPG", i),
			"checksum": base64.StdEncoding.EncodeToString(sum[:]), "visibility": "timeline", "isEdited": true,
			"exifInfo": map[string]any{"dateTimeOriginal": "2021-07-04T18:30:00+00:00", "fileSizeInByte": len(original)},
		}
		byID[id], originals[id] = asset, original
		items = append(items, asset)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		rest, isAsset := strings.CutPrefix(r.URL.Path, "/api/assets/")
		id, sub, _ := strings.Cut(rest, "/")
		isAsset = isAsset && byID[id] != nil && r.Method == http.MethodGet
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/search/metadata":
			var q map[string]any
			_ = json.NewDecoder(r.Body).Decode(&q)
			found := []any{}
			if q["visibility"] == "timeline" && q["trashedAfter"] == nil {
				found = items
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"assets": map[string]any{
				"total": len(found), "count": len(found), "items": found, "nextPage": nil,
			}})
		case r.Method == http.MethodGet && r.URL.Path == "/api/albums":
			_, _ = io.WriteString(w, `[]`)
		case isAsset && sub == "":
			_ = json.NewEncoder(w).Encode(byID[id])
		case isAsset && sub == "original":
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = io.WriteString(w, originals[id])
		case isAsset && sub == "edits":
			editCalls.Add(1)
			http.Error(w, `{"message":"Missing required permission: asset.edit.get"}`, http.StatusForbidden)
		case r.Method == http.MethodGet && r.URL.Path == "/api/stacks":
			_, _ = io.WriteString(w, "[]") // the server has no stacks
		default:
			t.Errorf("unexpected call %s %s", r.Method, r.URL)
			http.Error(w, "unexpected", http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, editCalls
}

// Only archive keeps the edit list. Under upload from-immich, which never uses it, an edited asset is
// sent on as before: no GET /edits call, no edit list, no warning. Under archive, it gets its list.
func TestEditListOnlyUnderArchive(t *testing.T) {
	for _, tc := range []struct {
		parent    string
		editCalls int32
	}{
		{"archive", 1},
		{"upload", 0},
	} {
		t.Run(tc.parent, func(t *testing.T) {
			srv := fakeImmich(t, func(asset map[string]any) { asset["isEdited"] = true })
			defer srv.Close()
			srv.editList = testEditList("dddddddd-0000-4000-8000-000000000001", "dddddddd-0000-4000-8000-000000000002")
			var log bytes.Buffer
			got, a, err := browseAssets(t, srv.URL, tc.parent, "stop", &log)
			if err != nil || len(got) != 1 {
				t.Fatalf("%d assets sent (%v), want 1", len(got), err)
			}
			if n := srv.editCalls.Load(); n != tc.editCalls {
				t.Fatalf("%d calls to GET /edits, want %d", n, tc.editCalls)
			}
			if md := got[0].FromApplication; (md.Edits != nil) != (tc.editCalls > 0) || md.EditsUnread {
				t.Fatalf("edit list %s (unread: %v), want one: %v", md.Edits, md.EditsUnread, tc.editCalls > 0)
			}
			if n := a.NumErrors(); n != 0 || strings.Contains(log.String(), "asset.edit.get") {
				t.Fatalf("%d errors, log:\n%s", n, log.String())
			}
		})
	}
}

// browseAssets runs getAssets against the server at url, as from-immich does under the command
// parent (archive, upload), with --on-errors set to onErrors, its log written to log, and returns
// the assets it sends, the application (its file processor is the run's) and getAssets' error.
func browseAssets(t *testing.T, url, parent, onErrors string, log io.Writer) ([]*assets.Asset, *app.Application, error) {
	t.Helper()
	ctx := context.Background()
	ic, err := immich.NewImmichClient(url, "test-key")
	if err != nil {
		t.Fatal(err)
	}
	a := app.New(ctx, nil)
	a.Log().Logger = slog.New(slog.NewTextHandler(log, nil))
	if err := a.OnErrors.Set(onErrors); err != nil {
		t.Fatal(err)
	}
	fic := &FromImmichCmd{app: a}
	fic.client.Immich = ic
	fic.client.User.ID = testOwner
	// As archive sets it up: the events are logged to the run's log.
	a.SetFileProcessor(fileprocessor.New(assettracker.New(), fileevent.NewRecorder(a.Log().Logger)))
	fic.processor = a.FileProcessor()
	fic.ifs = immichfs.NewImmichFS(ctx, url, ic)
	fic.ic = filenames.NewInfoCollector(time.UTC, filetypes.DefaultSupportedMedia)
	cmd := &cobra.Command{Use: "from-immich"}
	(&cobra.Command{Use: parent}).AddCommand(cmd)
	fic.edits.runUnder(cmd)

	ch := make(chan *assets.Group, 10)
	err = fic.getAssets(ctx, ch)
	close(ch)
	var got []*assets.Asset
	for g := range ch {
		got = append(got, g.Assets...)
	}
	return got, a, err
}
