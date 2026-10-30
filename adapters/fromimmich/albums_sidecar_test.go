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
	testOwner = "11111111-0000-4000-8000-000000000000"
	testAsset = "aaaaaaaa-0000-4000-8000-000000000001"
)

// fakeImmich answers the calls getAssets makes: one asset on the timeline, in two albums (one of
// them shared with the user by someone else).
func fakeImmich(t *testing.T) *httptest.Server {
	asset := map[string]any{
		"id": testAsset, "ownerId": testOwner, "type": "IMAGE", "originalFileName": "IMG_0001.JPG",
		"checksum": "c2hhMQ==", "visibility": "timeline",
		"exifInfo": map[string]any{"dateTimeOriginal": "2021-07-04T18:30:00+00:00", "fileSizeInByte": 11},
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/search/metadata":
			var q map[string]any
			_ = json.NewDecoder(r.Body).Decode(&q)
			items := []any{}
			if q["visibility"] == "timeline" && q["trashedAfter"] == nil {
				items = append(items, asset)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"assets": map[string]any{
				"total": len(items), "count": len(items), "items": items, "nextPage": nil,
			}})
		case r.Method == http.MethodGet && r.URL.Path == "/api/stacks":
			_, _ = io.WriteString(w, "[]")
		case r.Method == http.MethodGet && r.URL.Path == "/api/assets/"+testAsset:
			_ = json.NewEncoder(w).Encode(asset)
		case r.Method == http.MethodGet && r.URL.Path == "/api/albums" && r.URL.Query().Get("assetId") == testAsset:
			_, _ = io.WriteString(w, `[
				{"id": "bbbbbbbb-0000-4000-8000-000000000002", "albumName": "Summer", "description": "Lake trip"},
				{"id": "bbbbbbbb-0000-4000-8000-000000000001", "albumName": "Shared with Jon"}
			]`)
		default:
			t.Errorf("unexpected call %s %s", r.Method, r.URL)
			http.Error(w, "unexpected", http.StatusNotFound)
		}
	}))
}

func TestSidecarCarriesAlbums(t *testing.T) {
	srv := fakeImmich(t)
	defer srv.Close()
	ctx := context.Background()
	ic, err := immich.NewImmichClient(srv.URL, "test-key")
	if err != nil {
		t.Fatal(err)
	}
	fic := &FromImmichCmd{app: app.New(ctx, nil)}
	fic.client.Immich = ic
	fic.client.User.ID = testOwner
	fic.processor = fileprocessor.New(assettracker.New(), fileevent.NewRecorder(slog.New(slog.DiscardHandler)))
	fic.ifs = immichfs.NewImmichFS(ctx, srv.URL, ic)
	fic.ic = filenames.NewInfoCollector(time.UTC, filetypes.DefaultSupportedMedia)

	ch := make(chan *assets.Group, 10)
	if err := fic.getAssets(ctx, ch); err != nil {
		t.Fatal(err)
	}
	close(ch)
	var got []*assets.Asset
	for g := range ch {
		got = append(got, g.Assets...)
	}
	if len(got) != 1 {
		t.Fatalf("got %d assets, want 1", len(got))
	}
	md := got[0].FromApplication
	if md == nil {
		t.Fatal("no sidecar metadata")
	}
	want := []assets.Album{{Title: "Shared with Jon"}, {Title: "Summer", Description: "Lake trip"}}
	if len(md.Albums) != len(want) {
		t.Fatalf("sidecar albums %+v, want %+v", md.Albums, want)
	}
	for i := range want {
		if md.Albums[i].Title != want[i].Title || md.Albums[i].Description != want[i].Description || md.Albums[i].ID != "" {
			t.Fatalf("sidecar albums %+v, want %+v", md.Albums, want)
		}
	}

	// The written sidecar has them, and upload from-folder's reader gets them back.
	var buf bytes.Buffer
	if err := jsonsidecar.Write(md, &buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"title": "Summer"`) || !strings.Contains(buf.String(), `"description": "Lake trip"`) {
		t.Fatalf("sidecar JSON has no albums:\n%s", buf.String())
	}
	var back assets.Metadata
	if err := jsonsidecar.Read(&buf, &back); err != nil {
		t.Fatal(err)
	}
	if len(back.Albums) != 2 || back.Albums[1].Title != "Summer" || back.Albums[1].Description != "Lake trip" {
		t.Fatalf("read back %+v", back.Albums)
	}
}
