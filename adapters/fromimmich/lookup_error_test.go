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

// A lookup that fails for one asset (its GET /assets/{id}, the search that tells whether it is in
// a stack, or its albums) is an error for that asset, which is left out: it is never sent on with
// the empty asset a failed GET /assets/{id} leaves, nor with a stack or album list guessed from
// a call that did not answer. The other assets go on (--on-errors=continue).
func TestFailedLookupLeavesAssetOut(t *testing.T) {
	stackSrv := fakeStackImmich(t, true)
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

	ctx := context.Background()
	ic, err := immich.NewImmichClient(srv.URL, "test-key")
	if err != nil {
		t.Fatal(err)
	}
	var logBuf bytes.Buffer
	a := app.New(ctx, nil)
	a.OnErrors = cliflags.OnErrorsNeverStop
	a.SetLog(&app.Log{Logger: slog.New(slog.NewTextHandler(&logBuf, nil))})
	fic := &FromImmichCmd{app: a}
	fic.client.Immich = ic
	fic.client.User.ID = testOwner
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
	if strings.Contains(logs, unreported+" (it is hidden, or GET /stacks was refused): no stack in its sidecar id= ") {
		t.Errorf("warned about an asset with no ID; log:\n%s", logs)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(phantom) > 0 {
		t.Errorf("calls made for an asset with no ID: %q", phantom)
	}
}
