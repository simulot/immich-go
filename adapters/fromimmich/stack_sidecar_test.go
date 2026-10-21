package fromimmich

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/assettracker"
	"github.com/simulot/immich-go/internal/exif/sidecars/jsonsidecar"
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

// fakeStackImmich serves the assets above as Immich v3.2.2 does (checked on a scratch server):
// GET /assets/{id} reports S1 only (its member on the timeline); GET /stacks lists every stack's
// primary and its members that are not trashed or hidden, or answers 403 without stack.read;
// search/metadata with withStacked=false lists only the assets in no stack (and here skips one).
func fakeStackImmich(t *testing.T, stackRead bool) *httptest.Server {
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
			if !stackRead {
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"message": "Missing required permission: stack.read"}`))
				return
			}
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

func archiveStacks(t *testing.T, stackRead bool) (map[string]*assets.Metadata, string) {
	srv := fakeStackImmich(t, stackRead)
	defer srv.Close()
	ctx := context.Background()
	ic, err := immich.NewImmichClient(srv.URL, "test-key")
	if err != nil {
		t.Fatal(err)
	}
	var logBuf bytes.Buffer
	a := app.New(ctx, nil)
	a.SetLog(&app.Log{Logger: slog.New(slog.NewTextHandler(&logBuf, nil))})
	fic := &FromImmichCmd{app: a}
	fic.client.Immich = ic
	fic.client.User.ID = stackOwner
	fic.processor = fileprocessor.New(assettracker.New(), fileevent.NewRecorder(slog.New(slog.DiscardHandler)))
	fic.ifs = immichfs.NewImmichFS(ctx, srv.URL, ic)
	fic.ic = filenames.NewInfoCollector(time.UTC, filetypes.DefaultSupportedMedia)
	ch := make(chan *assets.Group, 20)
	if err := fic.getAssets(ctx, ch); err != nil {
		t.Fatal(err)
	}
	close(ch)
	got := map[string]*assets.Metadata{}
	for g := range ch {
		for _, x := range g.Assets {
			got[x.ID] = x.FromApplication
		}
	}
	if len(got) != 10 {
		t.Fatalf("got %d assets, want 10", len(got))
	}
	return got, logBuf.String()
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

const (
	unreported = "asset in a stack that Immich does not report"
	trashedMsg = "trashed stack member that Immich does not report"
	noRead     = "the API key lacks stack.read"
)

func TestSidecarCarriesStack(t *testing.T) {
	got, logs := archiveStacks(t, true)
	checkStacks(t, got, map[string]*assets.Stack{
		// Reported per asset; GET /stacks also counts the archived member.
		p1: {ID: s1, Primary: true, AssetCount: 3},
		m1: {ID: s1, AssetCount: 3},
		a1: {ID: s1, AssetCount: 3},
		// Not reported per asset: placed by GET /stacks.
		p2:   {ID: s2, Primary: true, AssetCount: 2},
		a2:   {ID: s2, AssetCount: 2},
		p3:   {ID: s3, Primary: true, AssetCount: 1},
		t3:   nil, // trashed: GET /stacks leaves it out
		h4:   nil, // hidden: reported nowhere
		lone: nil, skipped: nil,
	})
	all := []string{p1, m1, a1, p2, a2, p3, t3, h4, lone, skipped}
	if w := logged(logs, unreported, all...); len(w) != 1 || !w[h4] {
		t.Errorf("warned %v for %q, want only the hidden member; log:\n%s", w, unreported, logs)
	}
	if w := logged(logs, trashedMsg, all...); len(w) != 1 || !w[t3] {
		t.Errorf("warned %v for %q, want only the trashed member; log:\n%s", w, trashedMsg, logs)
	}
	if strings.Contains(logs, noRead) {
		t.Errorf("warned about stack.read, which the key has; log:\n%s", logs)
	}

	// In the written sidecar: the stack, with its source ID and the primary flag; none for an
	// asset in no stack. upload from-folder's reader gets it back.
	var buf bytes.Buffer
	if err := jsonsidecar.Write(got[a2], &buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"stack": {`) || !strings.Contains(buf.String(), `"id": "`+s2+`"`) ||
		!strings.Contains(buf.String(), `"primary": false`) {
		t.Fatalf("member sidecar has no stack:\n%s", buf.String())
	}
	var back assets.Metadata
	if err := jsonsidecar.Read(&buf, &back); err != nil {
		t.Fatal(err)
	}
	if back.Stack == nil || *back.Stack != (assets.Stack{ID: s2, AssetCount: 2}) {
		t.Fatalf("read back %+v", back.Stack)
	}
	buf.Reset()
	if err := jsonsidecar.Write(got[lone], &buf); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "stack") {
		t.Fatalf("sidecar of an asset in no stack names a stack:\n%s", buf.String())
	}
}

// Without stack.read: one warning about the key, the stacks GET /assets/{id} reports still
// archived, and every other stacked asset warned about, none guessed.
func TestSidecarStackWithoutStackRead(t *testing.T) {
	got, logs := archiveStacks(t, false)
	checkStacks(t, got, map[string]*assets.Stack{
		p1: {ID: s1, Primary: true, AssetCount: 2},
		m1: {ID: s1, AssetCount: 2},
		a1: {ID: s1, AssetCount: 2},
		p2: nil, a2: nil, p3: nil, t3: nil, h4: nil, lone: nil, skipped: nil,
	})
	if n := strings.Count(logs, noRead); n != 1 {
		t.Errorf("%d warnings about stack.read, want 1; log:\n%s", n, logs)
	}
	all := []string{p1, m1, a1, p2, a2, p3, t3, h4, lone, skipped}
	w := logged(logs, unreported, all...)
	if len(w) != 4 || !w[p2] || !w[a2] || !w[p3] || !w[h4] {
		t.Errorf("warned %v for %q, want p2, a2, p3, h4; log:\n%s", w, unreported, logs)
	}
	if w := logged(logs, trashedMsg, all...); len(w) != 1 || !w[t3] {
		t.Errorf("warned %v for %q, want only the trashed member; log:\n%s", w, trashedMsg, logs)
	}
}
