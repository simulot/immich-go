package folder

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"testing/fstest"
	"time"

	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/assettracker"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/fileprocessor"
	"github.com/simulot/immich-go/internal/filetypes"
	"github.com/simulot/immich-go/internal/groups"
	"github.com/simulot/immich-go/internal/groups/series"
)

const testXMP = `<?xpacket begin='' id='W5M0MpCehiHzreSzNTczkc9d'?>
<x:xmpmeta xmlns:x='adobe:ns:meta/'>
<rdf:RDF xmlns:rdf='http://www.w3.org/1999/02/22-rdf-syntax-ns#'>
 <rdf:Description rdf:about=''
  xmlns:tiff='http://ns.adobe.com/tiff/1.0/'>
  <tiff:ImageDescription>
   <rdf:Alt>
    <rdf:li xml:lang='x-default'>from the RAW sidecar</rdf:li>
   </rdf:Alt>
  </tiff:ImageDescription>
 </rdf:Description>
 <rdf:Description rdf:about=''
  xmlns:xmp='http://ns.adobe.com/xap/1.0/'>
  <xmp:Rating>4</xmp:Rating>
 </rdf:Description>
</rdf:RDF>
</x:xmpmeta>
<?xpacket end='w'?>`

// newTestFolderCmd builds the minimal ImportFolderCmd needed to call parseDir.
func newTestFolderCmd(ignoreSidecars bool) *ImportFolderCmd {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	a := app.New(context.Background(), nil)
	a.SetFileProcessor(fileprocessor.New(assettracker.New(), fileevent.NewRecorder(logger)))

	return &ImportFolderCmd{
		IgnoreSideCarFiles: ignoreSidecars,
		Recursive:          false,
		app:                a,
		processor:          a.FileProcessor(),
		tz:                 time.UTC,
		supportedMedia:     filetypes.DefaultSupportedMedia,
		infoCollector:      filenames.NewInfoCollector(time.UTC, filetypes.DefaultSupportedMedia),
		groupers:           []groups.Grouper{series.Group},
	}
}

func collectAssets(t *testing.T, ifc *ImportFolderCmd, fsys fstest.MapFS) map[string]*assets.Asset {
	t.Helper()
	ctx := context.Background()
	gOut := make(chan *assets.Group)
	errc := make(chan error, 1)
	go func() {
		errc <- ifc.parseDir(ctx, fsys, ".", gOut)
		close(gOut)
	}()

	found := map[string]*assets.Asset{}
	for g := range gOut {
		for _, a := range g.Assets {
			found[a.OriginalFileName] = a
		}
	}
	if err := <-errc; err != nil {
		t.Fatalf("parseDir: %v", err)
	}
	return found
}

// TestIgnoreSidecarFiles checks that --ignore-sidecar-files prevents XMP files
// from being attached to assets.
//
// Layout reproduced from a real library: a RAW+JPG pair where the XMP written
// by the RAW editor uses the short name (DSC0001.xmp). The JPG is imported
// alone (RAW excluded), so the XMP must not be attached to it when sidecars
// are ignored.
func TestIgnoreSidecarFiles(t *testing.T) {
	fsys := fstest.MapFS{
		"DSC0001.JPG": {Data: []byte("jpg"), ModTime: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
		"DSC0001.xmp": {Data: []byte(testXMP), ModTime: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
	}

	t.Run("sidecars used by default", func(t *testing.T) {
		found := collectAssets(t, newTestFolderCmd(false), fsys)
		a, ok := found["DSC0001.JPG"]
		if !ok {
			t.Fatalf("DSC0001.JPG not found, got %v", found)
		}
		if a.FromSideCar == nil {
			t.Fatal("expected the XMP sidecar to be attached")
		}
		if a.FromSideCar.Rating != 4 {
			t.Errorf("expected rating 4 from the sidecar, got %d", a.FromSideCar.Rating)
		}
	})

	t.Run("sidecars ignored with --ignore-sidecar-files", func(t *testing.T) {
		found := collectAssets(t, newTestFolderCmd(true), fsys)
		a, ok := found["DSC0001.JPG"]
		if !ok {
			t.Fatalf("DSC0001.JPG not found, got %v", found)
		}
		if a.FromSideCar != nil {
			t.Errorf("expected no sidecar with --ignore-sidecar-files, got %q", a.FromSideCar.File.Name())
		}
		if a.Rating != 0 || a.Description != "" {
			t.Errorf("metadata leaked from the ignored sidecar: rating=%d description=%q", a.Rating, a.Description)
		}
	})
}
