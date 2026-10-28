package fromimmich

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/simulot/immich-go/internal/assets"
)

// The warning for an asset in a stack that Immich does not report gives a cause only when it is
// known: "(it is hidden)" for a hidden asset. Another one gets a neutral cause, such as a partner's
// asset under --from-partners: GET /stacks lists only the user's own stacks.
func TestUnreportedStackWarningCause(t *testing.T) {
	const (
		partner = "22222222-0000-4000-8000-000000000000"
		hidden  = "aaaaaaaa-0000-4000-8000-0000000000f2" // the user's, hidden
		shared  = "aaaaaaaa-0000-4000-8000-0000000000f3" // the partner's, on the timeline
	)
	byID := map[string]map[string]any{
		hidden: unreportedAsset(hidden, stackOwner, "hidden", "IMG_0008.JPG"),
		shared: unreportedAsset(shared, partner, "timeline", "IMG_0009.JPG"),
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/search/metadata":
			// Both assets are in a stack: a search with withStacked=false finds neither.
			var q map[string]any
			_ = json.NewDecoder(r.Body).Decode(&q)
			items := []any{}
			for _, a := range byID {
				if q["withStacked"] == nil && q["trashedAfter"] == nil && q["visibility"] == a["visibility"] {
					items = append(items, a)
				}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"assets": map[string]any{
				"total": len(items), "count": len(items), "items": items, "nextPage": nil,
			}})
		case r.Method == http.MethodGet && r.URL.Path == "/api/stacks":
			_, _ = w.Write([]byte("[]"))
		case r.Method == http.MethodGet && byID[strings.TrimPrefix(r.URL.Path, "/api/assets/")] != nil:
			_ = json.NewEncoder(w).Encode(byID[strings.TrimPrefix(r.URL.Path, "/api/assets/")])
		case r.Method == http.MethodGet && r.URL.Path == "/api/albums":
			_, _ = w.Write([]byte("[]"))
		default:
			t.Errorf("unexpected call %s %s", r.Method, r.URL)
			http.Error(w, "unexpected", http.StatusNotFound)
		}
	}))
	defer srv.Close()

	var logBuf bytes.Buffer
	fic := newStackTestCmd(t, srv.URL, &logBuf)
	fic.IncludePartners = true
	ch := make(chan *assets.Group, 10)
	if err := fic.getAssets(context.Background(), ch); err != nil {
		t.Fatal(err)
	}
	close(ch)
	n := 0
	for g := range ch {
		for _, a := range g.Assets {
			n++
			if a.FromApplication.Stack != nil {
				t.Errorf("asset %s: stack %+v, want none", a.ID, a.FromApplication.Stack)
			}
		}
	}
	if n != len(byID) {
		t.Fatalf("%d assets sent, want %d", n, len(byID))
	}

	warning := map[string]string{}
	for line := range strings.Lines(logBuf.String()) {
		for id := range byID {
			if strings.Contains(line, unreported) && strings.Contains(line, id) {
				warning[id] = line
			}
		}
	}
	if w := warning[hidden]; !strings.Contains(w, "level=WARN") || !strings.Contains(w, unreported+" (it is hidden)") {
		t.Errorf("the hidden asset: warning %q, want one that says it is hidden; log:\n%s", w, logBuf.String())
	}
	if w := warning[shared]; !strings.Contains(w, "level=WARN") || strings.Contains(w, "hidden") {
		t.Errorf("the partner's asset: warning %q, want one that does not say it is hidden; log:\n%s", w, logBuf.String())
	}
}

// unreportedAsset is the asset id of owner, as the search and GET /assets/{id} return it: with no
// stack.
func unreportedAsset(id, owner, visibility, name string) map[string]any {
	return map[string]any{
		"id": id, "ownerId": owner, "type": "IMAGE", "originalFileName": name,
		"checksum": "c2hhMQ==", "visibility": visibility, "isTrashed": false, "isArchived": false,
		"exifInfo": map[string]any{"dateTimeOriginal": "2021-07-04T18:30:00+00:00", "fileSizeInByte": 11},
	}
}
