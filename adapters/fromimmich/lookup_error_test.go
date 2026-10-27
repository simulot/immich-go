package fromimmich

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/assettracker"
	cliflags "github.com/simulot/immich-go/internal/cliFlags"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/fileprocessor"
	"github.com/simulot/immich-go/internal/filetypes"
	"github.com/simulot/immich-go/internal/immichfs"
)

const (
	stackOwner = "11111111-0000-4000-8000-000000000000" // the user whose assets are archived

	s1 = "cccccccc-0000-4000-8000-000000000001" // primary and one member on the timeline, one archived
	s2 = "cccccccc-0000-4000-8000-000000000002" // its only other member archived
	s3 = "cccccccc-0000-4000-8000-000000000003" // its only other member trashed

	p1, m1, a1 = "aaaaaaaa-0000-4000-8000-0000000000a1", "aaaaaaaa-0000-4000-8000-0000000000a2", "aaaaaaaa-0000-4000-8000-0000000000a3"
	p2, a2     = "aaaaaaaa-0000-4000-8000-0000000000b1", "aaaaaaaa-0000-4000-8000-0000000000b2"
	p3, t3     = "aaaaaaaa-0000-4000-8000-0000000000c1", "aaaaaaaa-0000-4000-8000-0000000000c2"
	h4         = "aaaaaaaa-0000-4000-8000-0000000000d2" // hidden, in a stack of which nothing is served
	lone       = "aaaaaaaa-0000-4000-8000-0000000000e1" // in no stack
	skipped    = "aaaaaaaa-0000-4000-8000-0000000000e2" // in no stack; the withStacked=false listing skipped it
)

type fakeAsset struct {
	id, name, visibility string
	trashed              bool
	stack                map[string]any // what GET /assets/{id} reports
}

// fakeStackImmich is the server the lookups below fail on. It serves the assets above as Immich
// v3.2.2 does (checked on a scratch server): GET /assets/{id} reports S1 only (its member on the
// timeline); GET /stacks lists every stack's primary and its members that are not trashed or
// hidden; search/metadata with withStacked=false lists only the assets in no stack (and here
// skips one).
func fakeStackImmich(t *testing.T) *httptest.Server {
	rep := map[string]any{"id": s1, "primaryAssetId": p1, "assetCount": 2}
	all := []fakeAsset{
		{p1, "IMG_0001.JPG", "timeline", false, rep},
		{m1, "IMG_0001.CR2", "timeline", false, rep},
		{a1, "IMG_0001.HEIC", "archive", false, rep},
		{p2, "IMG_0002.JPG", "timeline", false, nil},
		{a2, "IMG_0002.CR2", "archive", false, nil},
		{p3, "IMG_0003.JPG", "timeline", false, nil},
		{t3, "IMG_0003.CR2", "timeline", true, nil},
		{h4, "IMG_0004.MP4", "hidden", false, nil},
		{lone, "IMG_0005.JPG", "timeline", false, nil},
		{skipped, "IMG_0006.JPG", "timeline", false, nil},
	}
	byID := map[string]fakeAsset{}
	for _, a := range all {
		byID[a.id] = a
	}
	js := func(a fakeAsset, withStack bool) map[string]any {
		out := map[string]any{
			"id": a.id, "ownerId": stackOwner, "type": "IMAGE", "originalFileName": a.name,
			"checksum": "c2hhMQ==", "visibility": a.visibility, "isTrashed": a.trashed,
			"isArchived": a.visibility == "archive",
			"exifInfo":   map[string]any{"dateTimeOriginal": "2021-07-04T18:30:00+00:00", "fileSizeInByte": 11},
		}
		if withStack && a.stack != nil {
			out["stack"] = a.stack
		}
		return out
	}
	stacks := `[
		{"id": "` + s1 + `", "primaryAssetId": "` + p1 + `", "assets": [{"id": "` + p1 + `"}, {"id": "` + m1 + `"}, {"id": "` + a1 + `"}]},
		{"id": "` + s2 + `", "primaryAssetId": "` + p2 + `", "assets": [{"id": "` + p2 + `"}, {"id": "` + a2 + `"}]},
		{"id": "` + s3 + `", "primaryAssetId": "` + p3 + `", "assets": [{"id": "` + p3 + `"}]}
	]`
	inNoStack := map[string]bool{lone: true, skipped: true}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/search/metadata":
			var q map[string]any
			_ = json.NewDecoder(r.Body).Decode(&q)
			unstackedOnly := q["withStacked"] == false
			items := []any{}
			for _, a := range all {
				if id, ok := q["id"].(string); ok {
					if id == a.id && (!unstackedOnly || inNoStack[a.id]) {
						items = append(items, js(a, false))
					}
					continue
				}
				if q["visibility"] != a.visibility || (q["trashedAfter"] != nil) != a.trashed {
					continue
				}
				if unstackedOnly && (!inNoStack[a.id] || a.id == skipped) {
					continue
				}
				items = append(items, js(a, false))
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"assets": map[string]any{
				"total": len(items), "count": len(items), "items": items, "nextPage": nil,
			}})
		case r.Method == http.MethodGet && r.URL.Path == "/api/stacks":
			_, _ = w.Write([]byte(stacks))
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/assets/"):
			a, ok := byID[strings.TrimPrefix(r.URL.Path, "/api/assets/")]
			if !ok {
				http.Error(w, "no such asset", http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(w).Encode(js(a, true))
		case r.Method == http.MethodGet && r.URL.Path == "/api/albums":
			_, _ = w.Write([]byte("[]"))
		default:
			t.Errorf("unexpected call %s %s", r.Method, r.URL)
			http.Error(w, "unexpected", http.StatusNotFound)
		}
	}))
}

// newStackTestCmd is a FromImmichCmd for the user stackOwner on the server at url, as the stack
// tests run getAssets: with --on-errors=continue, and its log written to log.
func newStackTestCmd(t *testing.T, url string, log io.Writer) *FromImmichCmd {
	t.Helper()
	ctx := context.Background()
	ic, err := immich.NewImmichClient(url, "test-key")
	if err != nil {
		t.Fatal(err)
	}
	a := app.New(ctx, nil)
	a.OnErrors = cliflags.OnErrorsNeverStop
	a.SetLog(&app.Log{Logger: slog.New(slog.NewTextHandler(log, nil))})
	fic := &FromImmichCmd{app: a}
	fic.client.Immich = ic
	fic.client.User.ID = stackOwner
	fic.processor = fileprocessor.New(assettracker.New(), fileevent.NewRecorder(slog.New(slog.DiscardHandler)))
	fic.ifs = immichfs.NewImmichFS(ctx, url, ic)
	fic.ic = filenames.NewInfoCollector(time.UTC, filetypes.DefaultSupportedMedia)
	return fic
}

func checkStacks(t *testing.T, got map[string]*assets.Metadata, want map[string]*assets.Stack) {
	t.Helper()
	for id, w := range want {
		s := got[id].Stack
		if (s == nil) != (w == nil) || (s != nil && *s != *w) {
			t.Errorf("asset %s: stack %+v, want %+v", id, s, w)
		}
	}
}

// logged tells which of the ids have a log line containing msg.
func logged(logs, msg string, ids ...string) map[string]bool {
	out := map[string]bool{}
	for _, l := range strings.Split(logs, "\n") {
		if strings.Contains(l, msg) {
			for _, id := range ids {
				if strings.Contains(l, id) {
					out[id] = true
				}
			}
		}
	}
	return out
}

const unreported = "asset in a stack that Immich does not report"

// A lookup that fails for one asset (its GET /assets/{id}, the search that tells whether it is in
// a stack, or its albums) is an error for that asset, which is left out: it is never sent on with
// the empty asset a failed GET /assets/{id} leaves, nor with a stack or album list guessed from
// a call that did not answer. The other assets go on (--on-errors=continue).
func TestFailedLookupLeavesAssetOut(t *testing.T) {
	stackSrv := fakeStackImmich(t)
	defer stackSrv.Close()
	var mu sync.Mutex
	var phantom []string
	fail := func(w http.ResponseWriter) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, `{"message": "failed on purpose"}`)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var q map[string]any
		_ = json.Unmarshal(body, &q)
		switch {
		case r.URL.Path == "/api/assets/" || r.URL.Path == "/api/assets" ||
			(r.URL.Path == "/api/albums" && r.URL.Query().Has("assetId") && r.URL.Query().Get("assetId") == "") ||
			(r.URL.Path == "/api/search/metadata" && q["withStacked"] == false && q["visibility"] == nil && q["id"] == nil):
			mu.Lock()
			phantom = append(phantom, r.Method+" "+r.URL.String()+" "+string(body))
			mu.Unlock()
			http.Error(w, "no asset ID", http.StatusBadRequest)
			return
		case r.Method == http.MethodGet && r.URL.Path == "/api/assets/"+p2:
			fail(w)
			return
		case r.Method == http.MethodPost && r.URL.Path == "/api/search/metadata" && q["id"] == skipped:
			fail(w)
			return
		case r.Method == http.MethodGet && r.URL.Path == "/api/albums" && r.URL.Query().Get("assetId") == lone:
			fail(w)
			return
		}
		// Anything else as the fake stack server answers it.
		req, _ := http.NewRequestWithContext(r.Context(), r.Method, stackSrv.URL+r.URL.String(), bytes.NewReader(body))
		req.Header = r.Header.Clone()
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
			return
		}
		defer resp.Body.Close()
		for k, v := range resp.Header {
			w.Header()[k] = v
		}
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
	}))
	defer srv.Close()

	var logBuf bytes.Buffer
	fic := newStackTestCmd(t, srv.URL, &logBuf)
	ch := make(chan *assets.Group, 20)
	if err := fic.getAssets(context.Background(), ch); err != nil {
		t.Fatal(err)
	}
	close(ch)
	got := map[string]*assets.Metadata{}
	for g := range ch {
		for _, x := range g.Assets {
			if x.ID == "" {
				t.Errorf("an asset with no ID was sent on: %+v", x.FromApplication)
				continue
			}
			got[x.ID] = x.FromApplication
		}
	}
	logs := logBuf.String()

	for _, id := range []string{p2, skipped, lone} {
		if _, ok := got[id]; ok {
			t.Errorf("asset %s was sent on although a lookup for it failed", id)
		}
	}
	if len(got) != 7 {
		t.Errorf("got %d assets, want the 7 whose lookups answered", len(got))
	}
	checkStacks(t, got, map[string]*assets.Stack{
		p1: {ID: s1, Primary: true, AssetCount: 3},
		m1: {ID: s1, AssetCount: 3},
		a1: {ID: s1, AssetCount: 3},
		a2: {ID: s2, AssetCount: 2},
		p3: {ID: s3, Primary: true, AssetCount: 1},
		t3: nil, h4: nil,
	})
	if n := strings.Count(logs, "level=ERROR"); n != 3 {
		t.Errorf("%d errors logged, want 3 (one per failed lookup); log:\n%s", n, logs)
	}
	if w := logged(logs, unreported, p2, skipped, lone); len(w) != 0 {
		t.Errorf("warned %v for %q after a failed lookup; log:\n%s", w, unreported, logs)
	}
	if strings.Contains(logs, unreported+" (it is hidden): no stack in its sidecar id= ") {
		t.Errorf("warned about an asset with no ID; log:\n%s", logs)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(phantom) > 0 {
		t.Errorf("calls made for an asset with no ID: %q", phantom)
	}
}
