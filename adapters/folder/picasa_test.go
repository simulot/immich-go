package folder

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/assettracker"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/fileprocessor"
	"github.com/simulot/immich-go/internal/filetypes"
	"github.com/simulot/immich-go/internal/gen"
)

// A Picasa album file has an extension that is not a media type. If the useless
// file filter runs before the Picasa check, the file is recorded as an unknown
// file and the album name is lost. The Picasa check must run first.
func TestPicasaIniIsReadBeforeTheUselessFilter(t *testing.T) {
	dir := t.TempDir()
	content := "[Picasa]\nname=My Album\ndescription=hello\n"
	if err := os.WriteFile(filepath.Join(dir, "picasa.ini"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	rec := fileevent.NewRecorder(slog.New(slog.NewTextHandler(io.Discard, nil)))
	ifc := &ImportFolderCmd{
		processor:      fileprocessor.New(assettracker.New(), rec),
		supportedMedia: filetypes.DefaultSupportedMedia,
		picasaAlbums:   gen.NewSyncMap[string, PicasaAlbum](),
		PicasaAlbum:    true,
	}

	out := make(chan *assets.Group, 1)
	if err := ifc.parseDir(context.Background(), os.DirFS(dir), ".", out); err != nil {
		t.Fatal(err)
	}

	a, ok := ifc.picasaAlbums.Load(".")
	if !ok {
		t.Fatal("picasa.ini was not read as an album")
	}
	if a.Name != "My Album" {
		t.Fatalf("album name = %q, want %q", a.Name, "My Album")
	}

	counts := rec.GetCounts()
	if counts[fileevent.DiscoveredMetadata] != 1 {
		t.Fatalf("DiscoveredMetadata = %d, want 1", counts[fileevent.DiscoveredMetadata])
	}
	if counts[fileevent.DiscoveredUnknown] != 0 {
		t.Fatalf("DiscoveredUnknown = %d, want 0. The useless filter took the file before the Picasa check", counts[fileevent.DiscoveredUnknown])
	}
}

// The same file must still be treated as a useless file when the Picasa option is
// off. The order change must not turn every .ini file into an album.
func TestPicasaIniIsStillSkippedWhenPicasaAlbumIsOff(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "picasa.ini"), []byte("[Picasa]\nname=My Album\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	rec := fileevent.NewRecorder(slog.New(slog.NewTextHandler(io.Discard, nil)))
	ifc := &ImportFolderCmd{
		processor:      fileprocessor.New(assettracker.New(), rec),
		supportedMedia: filetypes.DefaultSupportedMedia,
		picasaAlbums:   gen.NewSyncMap[string, PicasaAlbum](),
		PicasaAlbum:    false,
	}

	out := make(chan *assets.Group, 1)
	if err := ifc.parseDir(context.Background(), os.DirFS(dir), ".", out); err != nil {
		t.Fatal(err)
	}

	if _, ok := ifc.picasaAlbums.Load("."); ok {
		t.Fatal("an album was stored while PicasaAlbum is off")
	}
	counts := rec.GetCounts()
	if counts[fileevent.DiscoveredUnknown] != 1 {
		t.Fatalf("DiscoveredUnknown = %d, want 1", counts[fileevent.DiscoveredUnknown])
	}
}
