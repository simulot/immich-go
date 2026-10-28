package fromimmich

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/simulot/immich-go/internal/assets"
)

// A key without stack.read: GET /stacks is refused, and the one warning gives the number of assets
// whose stack could not be checked once the search is done. A search cut short by an error (here,
// the archived assets' query) gives no such number: the run fails with the error, and no warning.
// How a real server answers such a key is tested on one (the e2e tests).
func TestRefusedStackReadWarnsOnlyWhenTheSearchEnds(t *testing.T) {
	const stacked = "aaaaaaaa-0000-4000-8000-0000000000f1" // in a stack that GET /assets/{id} does not report
	for _, tc := range []struct {
		name        string
		searchFails bool
	}{
		{"search done", false},
		{"search cut short", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == http.MethodGet && r.URL.Path == "/api/stacks":
					w.WriteHeader(http.StatusForbidden)
					_, _ = io.WriteString(w, `{"message": "Missing required permission: stack.read"}`)
				case r.Method == http.MethodPost && r.URL.Path == "/api/search/metadata":
					var q map[string]any
					_ = json.NewDecoder(r.Body).Decode(&q)
					if tc.searchFails && q["withStacked"] == nil && q["visibility"] == "archive" {
						w.WriteHeader(http.StatusInternalServerError)
						_, _ = io.WriteString(w, `{"message": "failed on purpose"}`)
						return
					}
					items := []any{}
					if q["withStacked"] == nil && q["visibility"] == "timeline" && q["trashedAfter"] == nil {
						items = append(items, refusedAsset(stacked))
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"assets": map[string]any{
						"total": len(items), "count": len(items), "items": items, "nextPage": nil,
					}})
				case r.Method == http.MethodGet && r.URL.Path == "/api/assets/"+stacked:
					_ = json.NewEncoder(w).Encode(refusedAsset(stacked))
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
			ch := make(chan *assets.Group, 10)
			err := fic.getAssets(context.Background(), ch)
			close(ch)
			if (err != nil) != tc.searchFails {
				t.Fatalf("getAssets: %v, want an error: %v", err, tc.searchFails)
			}

			var warnings []string
			for line := range strings.Lines(logBuf.String()) {
				if strings.Contains(line, "lacks stack.read") {
					warnings = append(warnings, line)
				}
			}
			switch {
			case tc.searchFails && len(warnings) != 0:
				t.Errorf("warned after a search cut short:\n%s", logBuf.String())
			case !tc.searchFails && (len(warnings) != 1 || !strings.Contains(warnings[0], "level=WARN") ||
				!strings.Contains(warnings[0], "whose stack could not be checked") || !strings.Contains(warnings[0], "assets=1")):
				t.Errorf("want one warning with assets=1, got:\n%s", logBuf.String())
			}
		})
	}
}

// refusedAsset is the asset id, on the timeline, as the search and GET /assets/{id} return it: with
// no stack.
func refusedAsset(id string) map[string]any {
	return map[string]any{
		"id": id, "ownerId": stackOwner, "type": "IMAGE", "originalFileName": "IMG_0007.JPG",
		"checksum": "c2hhMQ==", "visibility": "timeline", "isTrashed": false, "isArchived": false,
		"exifInfo": map[string]any{"dateTimeOriginal": "2021-07-04T18:30:00+00:00", "fileSizeInByte": 11},
	}
}
