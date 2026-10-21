package fromimmich

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
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
)

const (
	testOwner    = "11111111-0000-4000-8000-000000000000"
	testAsset    = "aaaaaaaa-0000-4000-8000-000000000001"
	testLibrary  = "cccccccc-0000-4000-8000-000000000003"
	testChecksum = "c2hhMQ=="
	testOriginal = "hello world"
)

type fakeServer struct {
	*httptest.Server
	downloads atomic.Int32 // GET /api/assets/{id}/original
}

// fakeImmich answers the calls getAssets and a download make: one asset on the timeline, in two
// albums (one of them shared with the user by someone else), whose original is testOriginal.
// edit, when set, changes the asset the server describes.
func fakeImmich(t *testing.T, edit func(asset map[string]any)) *fakeServer {
	asset := map[string]any{
		"id": testAsset, "ownerId": testOwner, "type": "IMAGE", "originalFileName": "IMG_0001.JPG",
		"checksum": testChecksum, "visibility": "timeline",
		"exifInfo": map[string]any{"dateTimeOriginal": "2021-07-04T18:30:00+00:00", "fileSizeInByte": len(testOriginal)},
	}
	if edit != nil {
		edit(asset)
	}
	s := &fakeServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/assets/"+testAsset+"/original":
			s.downloads.Add(1)
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = io.WriteString(w, testOriginal)
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
	return s
}

// sidecarMetadata runs getAssets against fakeImmich and returns the metadata the asset's .JSON
// sidecar is written from.
func sidecarMetadata(t *testing.T) *assets.Metadata {
	t.Helper()
	srv := fakeImmich(t, nil)
	defer srv.Close()
	md := archivedAsset(t, srv).FromApplication
	if md == nil {
		t.Fatal("no sidecar metadata")
	}
	return md
}

// archivedAsset runs getAssets against srv and returns the one asset it gives, as the archive
// command gets it: opening its file downloads the original from srv.
func archivedAsset(t *testing.T, srv *fakeServer) *assets.Asset {
	t.Helper()
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
	return got[0]
}

func TestSidecarCarriesAlbums(t *testing.T) {
	md := sidecarMetadata(t)
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

// The sidecar carries the asset's checksum as the API gives it (SHA-1, base64): the archive
// identifies its files by it. The reader gets it back.
func TestSidecarCarriesChecksum(t *testing.T) {
	md := sidecarMetadata(t)
	if md.Checksum != testChecksum {
		t.Fatalf("sidecar checksum %q, want %q", md.Checksum, testChecksum)
	}
	var buf bytes.Buffer
	if err := jsonsidecar.Write(md, &buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"checksum": "`+testChecksum+`"`) {
		t.Fatalf("sidecar JSON has no checksum:\n%s", buf.String())
	}
	var back assets.Metadata
	if err := jsonsidecar.Read(&buf, &back); err != nil {
		t.Fatal(err)
	}
	if back.Checksum != testChecksum {
		t.Fatalf("read back checksum %q, want %q", back.Checksum, testChecksum)
	}
}

// An external-library asset's Immich checksum is a hash of its path, not of its bytes (Immich's
// library service, checksum algorithm sha1-path). The archive keeps it by the size rule: written
// once, its sidecar records no checksum, and a rerun does not download it again.
func TestExternalLibraryAssetArchivedBySize(t *testing.T) {
	t.Setenv("IMMICHGO_TEMPDIR", t.TempDir())
	pathSum := sha1.Sum([]byte("path:/mnt/photos/IMG_0001.JPG"))
	srv := fakeImmich(t, func(asset map[string]any) {
		asset["libraryId"] = testLibrary
		asset["checksum"] = base64.StdEncoding.EncodeToString(pathSum[:])
	})
	defer srv.Close()
	dst := t.TempDir()
	for run := 1; run <= 2; run++ {
		w, err := folder.NewLocalAssetWriter(osfs.DirFS(dst), ".")
		if err != nil {
			t.Fatal(err)
		}
		a := archivedAsset(t, srv)
		err = w.WriteAsset(context.Background(), a)
		_ = a.Close()
		if err != nil {
			t.Fatalf("run %d: %v", run, err)
		}
		if n := srv.downloads.Load(); n != 1 {
			t.Fatalf("run %d: %d downloads in all, want 1", run, n)
		}
	}

	var files []string
	err := filepath.WalkDir(dst, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			r, _ := filepath.Rel(dst, p)
			files = append(files, filepath.ToSlash(r))
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"2021/2021-07/IMG_0001.JPG", "2021/2021-07/IMG_0001.JPG.JSON"}
	if !slices.Equal(files, want) {
		t.Fatalf("files %v, want %v", files, want)
	}
	if b, _ := os.ReadFile(filepath.Join(dst, "2021", "2021-07", "IMG_0001.JPG")); string(b) != testOriginal {
		t.Fatalf("IMG_0001.JPG holds %q", b)
	}
	sidecar, err := os.ReadFile(filepath.Join(dst, "2021", "2021-07", "IMG_0001.JPG.JSON"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(sidecar, []byte(`"checksum"`)) {
		t.Fatalf("the sidecar records the path checksum:\n%s", sidecar)
	}
}
