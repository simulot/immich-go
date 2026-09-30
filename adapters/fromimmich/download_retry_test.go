package fromimmich

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/simulot/immich-go/adapters/folder"
	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/assettracker"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/fileprocessor"
	"github.com/simulot/immich-go/internal/filetypes"
	"github.com/simulot/immich-go/internal/fshelper/osfs"
	"github.com/simulot/immich-go/internal/immichfs"
)

// cutImmich serves one asset whose original is content; cut(n) is how many bytes of the n-th
// download (from 1) it sends before it drops the connection, or -1 for all of them.
func cutImmich(t *testing.T, content []byte, cut func(n int) int) (*httptest.Server, *atomic.Int32) {
	var downloads atomic.Int32
	asset := map[string]any{
		"id": testAsset, "ownerId": testOwner, "type": "IMAGE", "originalFileName": "IMG_0001.JPG",
		"checksum": "c2hhMQ==", "visibility": "timeline",
		"exifInfo": map[string]any{"dateTimeOriginal": "2021-07-04T18:30:00+00:00", "fileSizeInByte": len(content)},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		case r.Method == http.MethodGet && r.URL.Path == "/api/albums":
			_, _ = io.WriteString(w, "[]")
		case r.Method == http.MethodGet && r.URL.Path == "/api/assets/"+testAsset+"/original":
			n := int(downloads.Add(1))
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Length", strconv.Itoa(len(content)))
			if c := cut(n); c >= 0 {
				_, _ = w.Write(content[:c])
				w.(http.Flusher).Flush()
				panic(http.ErrAbortHandler)
			}
			_, _ = w.Write(content)
		default:
			t.Errorf("unexpected call %s %s", r.Method, r.URL)
			http.Error(w, "unexpected", http.StatusNotFound)
		}
	}))
	return srv, &downloads
}

// archiveOne archives the one asset of srv into dir, as archive from-immich does (the download's
// temp copy in the test's own dir).
func archiveOne(t *testing.T, srv *httptest.Server, dir string) error {
	t.Helper()
	t.Setenv("IMMICHGO_TEMPDIR", t.TempDir())
	ctx := context.Background()
	ic, err := immich.NewImmichClient(srv.URL, "test-key", immich.OptionRetries(5, time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
	a := app.New(ctx, nil)
	a.SetLog(&app.Log{Logger: slog.New(slog.DiscardHandler)})
	fic := &FromImmichCmd{app: a}
	fic.client.Immich = ic
	fic.client.User.ID = testOwner
	fic.processor = fileprocessor.New(assettracker.New(), fileevent.NewRecorder(slog.New(slog.DiscardHandler)))
	fic.ifs = immichfs.NewImmichFS(ctx, srv.URL, ic)
	fic.ic = filenames.NewInfoCollector(time.UTC, filetypes.DefaultSupportedMedia)
	ch := make(chan *assets.Group, 1)
	if err := fic.getAssets(ctx, ch); err != nil {
		t.Fatal(err)
	}
	close(ch)
	w, err := folder.NewLocalAssetWriter(osfs.DirFS(dir), ".")
	if err != nil {
		t.Fatal(err)
	}
	var werr error
	for g := range ch {
		for _, x := range g.Assets {
			werr = w.WriteAsset(ctx, x)
			if cerr := x.Close(); werr == nil {
				werr = cerr
			}
		}
	}
	return werr
}

func filesIn(t *testing.T, dir string) []string {
	var out []string
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			rel, _ := filepath.Rel(dir, p)
			out = append(out, rel)
		}
		return nil
	})
	return out
}

// A download that breaks off midway is fetched again from its first byte: the archive gets the
// whole original under its name (through the .partial file), and its sidecar.
func TestArchiveDownloadCutOnce(t *testing.T) {
	content := bytes.Repeat([]byte("0123456789abcdef"), 1<<14)
	srv, downloads := cutImmich(t, content, func(n int) int {
		if n == 1 {
			return len(content) / 2
		}
		return -1
	})
	defer srv.Close()
	dir := t.TempDir()
	if err := archiveOne(t, srv, dir); err != nil {
		t.Fatalf("archive: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "2021", "2021-07", "IMG_0001.JPG"))
	if err != nil || !bytes.Equal(got, content) {
		t.Fatalf("archived original: %d bytes, %v; want the %d of the asset", len(got), err, len(content))
	}
	if files := filesIn(t, dir); len(files) != 2 || !strings.HasSuffix(files[1], "IMG_0001.JPG.JSON") {
		t.Fatalf("files %q, want the original and its sidecar only", files)
	}
	if downloads.Load() != 2 {
		t.Errorf("%d downloads, want 2", downloads.Load())
	}
}

// A download cut every time fails the asset after 5 transfers: nothing under its name, no
// .partial file and no sidecar is left.
func TestArchiveDownloadAlwaysCut(t *testing.T) {
	content := bytes.Repeat([]byte("0123456789abcdef"), 1<<14)
	srv, downloads := cutImmich(t, content, func(int) int { return len(content) / 2 })
	defer srv.Close()
	dir := t.TempDir()
	if err := archiveOne(t, srv, dir); err == nil {
		t.Fatal("archive: no error")
	}
	if files := filesIn(t, dir); len(files) != 0 {
		t.Fatalf("files %q left, want none", files)
	}
	if downloads.Load() != 5 {
		t.Errorf("%d downloads, want 5", downloads.Load())
	}
}
