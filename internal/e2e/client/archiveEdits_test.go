//go:build e2e

package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestArchiveKeepsEditList crops and rotates one photo in Immich, then archives the account. The
// archive holds the photo's original, and its .JSON file the two actions, in order, without their
// ids; the other photos' .JSON files have no edit list. The edit list is asked for only for the
// edited photo, and a second run rewrites nothing.
func TestArchiveKeepsEditList(t *testing.T) {
	const assetCount, edited = 10, "mountains_04.jpg"
	adm, u := e2eAdmin(t), e2eNewUser(t)
	uploadFromFolder(t, u, adm, "DATA/fromFolder/recursive/a/b/c")
	checksums := serverChecksums(t, u, assetCount)
	tk := login(t, u)

	var found struct {
		Assets struct {
			Items []struct {
				ID string `json:"id"`
			} `json:"items"`
		} `json:"assets"`
	}
	callAPI(t, tk, http.MethodPost, "/search/metadata", map[string]any{"originalFileName": edited}, &found)
	if len(found.Assets.Items) != 1 {
		t.Fatalf("%d assets named %s, want 1", len(found.Assets.Items), edited)
	}
	id := found.Assets.Items[0].ID
	var info struct {
		ExifInfo struct {
			Width  int `json:"exifImageWidth"`
			Height int `json:"exifImageHeight"`
		} `json:"exifInfo"`
	}
	callAPI(t, tk, http.MethodGet, "/assets/"+id, nil, &info)
	w, h := info.ExifInfo.Width, info.ExifInfo.Height
	if w < 4 || h < 4 {
		t.Fatalf("%s: %dx%d, too small to crop", edited, w, h)
	}
	crop := map[string]any{"x": w / 4, "y": h / 4, "width": w / 2, "height": h / 2}
	var saved struct {
		Edits []map[string]any `json:"edits"`
	}
	callAPI(t, tk, http.MethodPut, "/assets/"+id+"/edits", map[string]any{"edits": []any{
		map[string]any{"action": "crop", "parameters": crop},
		map[string]any{"action": "rotate", "parameters": map[string]any{"angle": 90}},
	}}, &saved)
	if len(saved.Edits) != 2 || saved.Edits[0]["id"] == nil || saved.Edits[1]["id"] == nil {
		t.Fatalf("PUT /assets/%s/edits saved %v, want 2 actions, each with an id", id, saved.Edits)
	}

	// The edit list the archive must keep: the actions as saved, without their ids.
	var want []map[string]any
	wantJSON, err := json.Marshal([]any{
		map[string]any{"action": "crop", "parameters": crop},
		map[string]any{"action": "rotate", "parameters": map[string]any{"angle": 90}},
	})
	if err == nil {
		err = json.Unmarshal(wantJSON, &want)
	}
	if err != nil {
		t.Fatal(err)
	}

	proxy := newEditsProxy(t)
	dir := t.TempDir()
	if _, err := archiveFromImmich(t.Context(), proxy.URL, u, adm, dir); err != nil {
		t.Fatalf("first run: %v", err)
	}
	if got := proxy.calls(); !slices.Equal(got, []string{id}) {
		t.Errorf("first run: GET /edits for %v, want only the edited asset %s", got, id)
	}
	archived := checkArchivedFiles(t, dir, checksums)
	if len(archived) != assetCount {
		t.Fatalf("%d assets archived, want %d", len(archived), assetCount)
	}
	for name, p := range archived {
		var sidecar struct {
			Edits json.RawMessage `json:"edits"`
		}
		b, err := os.ReadFile(p + ".JSON")
		if err == nil {
			err = json.Unmarshal(b, &sidecar)
		}
		if err != nil {
			t.Fatalf("%s.JSON: %v", p, err)
		}
		if name != edited {
			if sidecar.Edits != nil {
				t.Errorf("%s.JSON: edit list %s, want none", p, sidecar.Edits)
			}
			continue
		}
		var got []map[string]any
		if err := json.Unmarshal(sidecar.Edits, &got); err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("%s.JSON: edit list %s (%v), want %s", p, sidecar.Edits, err, wantJSON)
		}
		if strings.Contains(string(b), "isEdited") {
			t.Errorf("%s.JSON has an isEdited field:\n%s", p, b)
		}
	}

	// Back-date every file, so that a file the second run writes again gets a new mtime whatever
	// the resolution of the file system's clock.
	past := time.Date(2001, 2, 3, 4, 5, 6, 0, time.UTC)
	before := listFiles(t, dir)
	for p := range before {
		if err := os.Chtimes(p, past, past); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := archiveFromImmich(t.Context(), proxy.URL, u, adm, dir); err != nil {
		t.Fatalf("second run: %v", err)
	}
	if got := proxy.calls(); !slices.Equal(got, []string{id, id}) {
		t.Errorf("both runs: GET /edits for %v, want only the edited asset %s, once a run", got, id)
	}
	after := listFiles(t, dir)
	for p, mtime := range after {
		if !mtime.Equal(past) {
			t.Errorf("%s: written by the second run", filepath.Base(p))
		}
	}
	if len(after) != len(before) {
		t.Errorf("%d files after the second run, %d before", len(after), len(before))
	}
}

// editsCall matches the call for an asset's edits, and captures the asset's ID.
var editsCall = regexp.MustCompile(`^/api/assets/([^/]+)/edits$`)

// editsProxy is a reverse proxy to the e2e server that records the assets whose edits are asked for.
type editsProxy struct {
	*httptest.Server
	mu  sync.Mutex
	ids []string // the asset of each GET /api/assets/{id}/edits, in order
}

func newEditsProxy(t *testing.T) *editsProxy {
	t.Helper()
	target, err := url.Parse(ImmichURL)
	if err != nil {
		t.Fatal(err)
	}
	rp := &httputil.ReverseProxy{Rewrite: func(r *httputil.ProxyRequest) { r.SetURL(target) }}
	p := &editsProxy{}
	p.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m := editsCall.FindStringSubmatch(r.URL.Path); m != nil && r.Method == http.MethodGet {
			p.mu.Lock()
			p.ids = append(p.ids, m[1])
			p.mu.Unlock()
		}
		rp.ServeHTTP(w, r)
	}))
	t.Cleanup(p.Close)
	return p
}

func (p *editsProxy) calls() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.ids)
}
