package gp

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
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
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/simulot/immich-go/internal/gen"
	"github.com/simulot/immich-go/internal/groups"
	"github.com/simulot/immich-go/internal/groups/burst"
	"github.com/simulot/immich-go/internal/groups/epsonfastfoto"
	"github.com/simulot/immich-go/internal/groups/series"
	"github.com/spf13/cobra"
)

// Google Takeout splits a large export into several series of zip parts
// (takeout-<date>-1-001.zip, takeout-<date>-1-002.zip, ..., takeout-<date>-2-001.zip, ...).
// Each series numbers its own name clashes "(1)", "(2)"... from scratch, so two
// different photos can have the same path inside the export, one per series,
// each with a sidecar JSON of the same name in its own series.

const collisionDir = "Takeout/Google Photos/Photos from 2025"

// namedMapFS is an in-memory takeout part. Its name is what fshelper gives a zip
// part: the base name without the extension.
type namedMapFS struct {
	fstest.MapFS
	name string
}

func (n namedMapFS) Name() string { return n.name }

func sidecarJSON(title string, taken time.Time) []byte {
	return fmt.Appendf(nil, `{"title": %q, "photoTakenTime": {"timestamp": "%d"}}`, title, taken.Unix())
}

func newCollisionTakeoutCmd(t *testing.T, parts ...fs.FS) (*TakeoutCmd, *fileevent.Recorder) {
	t.Helper()
	discard := slog.New(slog.NewTextHandler(io.Discard, nil))
	rec := fileevent.NewRecorder(discard)
	a := app.New(context.Background(), &cobra.Command{})
	a.SetLog(&app.Log{Logger: discard})
	processor := fileprocessor.New(assettracker.New(), rec)
	a.SetFileProcessor(processor)
	return &TakeoutCmd{
		app:            a,
		processor:      processor,
		supportedMedia: filetypes.DefaultSupportedMedia,
		infoCollector:  filenames.NewInfoCollector(time.UTC, filetypes.DefaultSupportedMedia),
		tz:             time.UTC,
		fsyss:          parts,
		catalogs:       map[string]directoryCatalog{},
		albums:         map[string]assets.Album{},
		fileTracker:    gen.NewSyncMap[fileKeyTracker, trackingInfo](),
		KeepJSONLess:   true,
		groupers:       []groups.Grouper{series.Group}, // as in the command's RunE
	}, rec
}

func browseGroups(t *testing.T, toc *TakeoutCmd) []*assets.Group {
	t.Helper()
	var out []*assets.Group
	for g := range toc.Browse(t.Context()) {
		out = append(out, g)
	}
	return out
}

func browseAll(t *testing.T, toc *TakeoutCmd) []*assets.Asset {
	t.Helper()
	var out []*assets.Asset
	for _, g := range browseGroups(t, toc) {
		out = append(out, g.Assets...)
	}
	return out
}

// bySize returns the assets that came from a file of the given size.
func bySize(all []*assets.Asset, size int) []*assets.Asset {
	var out []*assets.Asset
	for _, a := range all {
		if a.FileSize == size {
			out = append(out, a)
		}
	}
	return out
}

func dateOf(t *testing.T, all []*assets.Asset, size int) time.Time {
	t.Helper()
	as := bySize(all, size)
	if len(as) != 1 {
		t.Fatalf("%d-byte file: got %d assets, want 1", size, len(as))
	}
	if as[0].FromApplication == nil {
		t.Fatalf("%d-byte file: no sidecar metadata attached", size)
	}
	return as[0].FromApplication.DateTaken
}

// writeZip writes a takeout part into dir.
func writeZip(t *testing.T, dir, name string, files map[string][]byte) {
	t.Helper()
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for n, b := range files {
		w, err := zw.Create(n)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(b); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

// writeDir writes a takeout part extracted into the folder dir/name.
func writeDir(t *testing.T, dir, name string, files map[string][]byte) {
	t.Helper()
	for n, b := range files {
		p := filepath.Join(dir, name, filepath.FromSlash(n))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

// Two different photos with the same path in two series of real zip parts, opened
// the way the command opens them, are both imported, each with the date from the
// sidecar of its own series.
func TestSamePathInTwoSeriesOfRealZips(t *testing.T) {
	takenS1 := time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC)
	takenS2 := time.Date(2025, 8, 20, 18, 30, 0, 0, time.UTC)
	dir := t.TempDir()
	writeZip(t, dir, "takeout-20251016T092000Z-1-001.zip", map[string][]byte{
		collisionDir + "/IMG_0001(1).jpg":                            []byte(strings.Repeat("a", 100)),
		collisionDir + "/IMG_0001.jpg.supplemental-metadata(1).json": sidecarJSON("IMG_0001.jpg", takenS1),
	})
	writeZip(t, dir, "takeout-20251016T092000Z-2-001.zip", map[string][]byte{
		collisionDir + "/IMG_0001(1).jpg":                            []byte(strings.Repeat("b", 200)),
		collisionDir + "/IMG_0001.jpg.supplemental-metadata(1).json": sidecarJSON("IMG_0001.jpg", takenS2),
	})
	parts, err := fshelper.ParsePath([]string{filepath.Join(dir, "takeout-*.zip")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = fshelper.CloseFSs(parts) })
	toc, rec := newCollisionTakeoutCmd(t, parts...)

	all := browseAll(t, toc)

	if n := rec.GetCounts()[fileevent.DiscardedLocalDuplicate]; n != 0 {
		t.Errorf("%d file(s) discarded as local duplicates, want 0: the two files differ", n)
	}
	if got := dateOf(t, all, 100); !got.Equal(takenS1) {
		t.Errorf("series-1 photo: date %v, want %v from its own sidecar", got, takenS1)
	}
	if got := dateOf(t, all, 200); !got.Equal(takenS2) {
		t.Errorf("series-2 photo: date %v, want %v from its own sidecar", got, takenS2)
	}
}

// The same with the parts extracted into folders named after their zips (as Windows
// does) and each passed as an argument: the folder name gives the series.
func TestSamePathInTwoSeriesOfExtractedPartFolders(t *testing.T) {
	takenS1 := time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC)
	takenS2 := time.Date(2025, 8, 20, 18, 30, 0, 0, time.UTC)
	dir := t.TempDir()
	writeDir(t, dir, "takeout-20251016T092000Z-1-001", map[string][]byte{
		collisionDir + "/IMG_0001(1).jpg":                            []byte(strings.Repeat("a", 100)),
		collisionDir + "/IMG_0001.jpg.supplemental-metadata(1).json": sidecarJSON("IMG_0001.jpg", takenS1),
	})
	writeDir(t, dir, "takeout-20251016T092000Z-2-001", map[string][]byte{
		collisionDir + "/IMG_0001(1).jpg":                            []byte(strings.Repeat("b", 200)),
		collisionDir + "/IMG_0001.jpg.supplemental-metadata(1).json": sidecarJSON("IMG_0001.jpg", takenS2),
	})
	parts, err := fshelper.ParsePath([]string{
		filepath.Join(dir, "takeout-20251016T092000Z-1-001"),
		filepath.Join(dir, "takeout-20251016T092000Z-2-001"),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = fshelper.CloseFSs(parts) })
	toc, rec := newCollisionTakeoutCmd(t, parts...)

	all := browseAll(t, toc)

	if n := rec.GetCounts()[fileevent.DiscardedLocalDuplicate]; n != 0 {
		t.Errorf("%d file(s) discarded as local duplicates, want 0: the two files differ", n)
	}
	if got := dateOf(t, all, 100); !got.Equal(takenS1) {
		t.Errorf("series-1 photo: date %v, want %v from its own sidecar", got, takenS1)
	}
	if got := dateOf(t, all, 200); !got.Equal(takenS2) {
		t.Errorf("series-2 photo: date %v, want %v from its own sidecar", got, takenS2)
	}
}

// Two different files kept under one name are marked, so that the upload compares them
// with the server by checksum only: name, date and size cannot tell them apart from
// versions of one photo. Other files are not marked.
func TestFilesOfTheSameNameInTwoSeriesAreMarkedAsNamesakes(t *testing.T) {
	taken := time.Date(2025, 3, 26, 11, 26, 52, 0, time.UTC)
	s1 := namedMapFS{name: "takeout-20251016T092000Z-1-001", MapFS: fstest.MapFS{
		collisionDir + "/IMG_3285.MOV":                            {Data: []byte(strings.Repeat("a", 100))},
		collisionDir + "/IMG_3285.MOV.supplemental-metadata.json": {Data: sidecarJSON("IMG_3285.MOV", taken)},
		collisionDir + "/IMG_3286.MOV":                            {Data: []byte(strings.Repeat("c", 300))},
		collisionDir + "/IMG_3286.MOV.supplemental-metadata.json": {Data: sidecarJSON("IMG_3286.MOV", taken)},
	}}
	s2 := namedMapFS{name: "takeout-20251016T092000Z-2-001", MapFS: fstest.MapFS{
		collisionDir + "/IMG_3285.MOV":                            {Data: []byte(strings.Repeat("b", 200))},
		collisionDir + "/IMG_3285.MOV.supplemental-metadata.json": {Data: sidecarJSON("IMG_3285.MOV", taken)},
	}}
	toc, _ := newCollisionTakeoutCmd(t, s1, s2)

	all := browseAll(t, toc)

	for _, c := range []struct {
		size int
		want bool
	}{{100, true}, {200, true}, {300, false}} {
		as := bySize(all, c.size)
		if len(as) != 1 {
			t.Fatalf("%d-byte file: got %d assets, want 1", c.size, len(as))
		}
		if as[0].HasNamesake != c.want {
			t.Errorf("%d-byte file: HasNamesake %v, want %v", c.size, as[0].HasNamesake, c.want)
		}
	}
}

// Album folders hold copies of the files of the year folders. The copy of a namesake is
// marked too: when the album folder comes first, the copy is the one uploaded.
func TestAlbumCopyOfANamesakeIsMarked(t *testing.T) {
	taken := time.Date(2025, 3, 26, 11, 26, 52, 0, time.UTC)
	album := "Takeout/Google Photos/Album X"
	s1 := namedMapFS{name: "takeout-20251016T092000Z-1-001", MapFS: fstest.MapFS{
		collisionDir + "/IMG_3285.MOV":                            {Data: []byte(strings.Repeat("a", 100))},
		collisionDir + "/IMG_3285.MOV.supplemental-metadata.json": {Data: sidecarJSON("IMG_3285.MOV", taken)},
	}}
	s2 := namedMapFS{name: "takeout-20251016T092000Z-2-001", MapFS: fstest.MapFS{
		collisionDir + "/IMG_3285.MOV":                            {Data: []byte(strings.Repeat("b", 200))},
		collisionDir + "/IMG_3285.MOV.supplemental-metadata.json": {Data: sidecarJSON("IMG_3285.MOV", taken)},
		album + "/IMG_3285.MOV":                                   {Data: []byte(strings.Repeat("b", 200))},
		album + "/IMG_3285.MOV.supplemental-metadata.json":        {Data: sidecarJSON("IMG_3285.MOV", taken)},
		album + "/IMG_4000.MOV":                                   {Data: []byte(strings.Repeat("c", 300))},
		album + "/IMG_4000.MOV.supplemental-metadata.json":        {Data: sidecarJSON("IMG_4000.MOV", taken)},
		album + "/metadata.json":                                  {Data: []byte(`{"title": "Album X"}`)},
	}}
	toc, _ := newCollisionTakeoutCmd(t, s1, s2)

	all := browseAll(t, toc)

	for _, a := range all {
		if want := a.FileSize != 300; a.HasNamesake != want {
			t.Errorf("%s: HasNamesake %v, want %v", a.File.Name(), a.HasNamesake, want)
		}
	}
	// the album folder sorts first: its copy is emitted, the year folder's is a local duplicate
	b := bySize(all, 200)
	if len(b) != 1 || b[0].File.Name() != album+"/IMG_3285.MOV" {
		t.Errorf("series-2 file: got %d assets, want 1, the album's copy", len(b))
	}
}

// The upload compares files by their title, the name in the sidecar. A file titled like a
// namesake is marked as well: here IMG_0001.jpg, as the title of the two different
// IMG_0001(1).jpg, can be taken by name and date for one of them.
func TestFileTitledLikeANamesakeIsMarked(t *testing.T) {
	taken := time.Date(2025, 3, 26, 11, 26, 52, 0, time.UTC)
	s1 := namedMapFS{name: "takeout-20251016T092000Z-1-001", MapFS: fstest.MapFS{
		collisionDir + "/IMG_0001.jpg":                               {Data: []byte(strings.Repeat("a", 100))},
		collisionDir + "/IMG_0001.jpg.supplemental-metadata.json":    {Data: sidecarJSON("IMG_0001.jpg", taken)},
		collisionDir + "/IMG_0001(1).jpg":                            {Data: []byte(strings.Repeat("c", 300))},
		collisionDir + "/IMG_0001.jpg.supplemental-metadata(1).json": {Data: sidecarJSON("IMG_0001.jpg", taken.Add(3*time.Second))},
		collisionDir + "/IMG_0002.jpg":                               {Data: []byte(strings.Repeat("e", 500))},
		collisionDir + "/IMG_0002.jpg.supplemental-metadata.json":    {Data: sidecarJSON("IMG_0002.jpg", taken)},
	}}
	s2 := namedMapFS{name: "takeout-20251016T092000Z-2-001", MapFS: fstest.MapFS{
		collisionDir + "/IMG_0001(1).jpg":                            {Data: []byte(strings.Repeat("d", 400))},
		collisionDir + "/IMG_0001.jpg.supplemental-metadata(1).json": {Data: sidecarJSON("IMG_0001.jpg", taken.Add(4*time.Second))},
	}}
	toc, _ := newCollisionTakeoutCmd(t, s1, s2)

	all := browseAll(t, toc)

	for _, c := range []struct {
		size int
		want bool
	}{{100, true}, {300, true}, {400, true}, {500, false}} {
		as := bySize(all, c.size)
		if len(as) != 1 {
			t.Fatalf("%d-byte file: got %d assets, want 1", c.size, len(as))
		}
		if as[0].HasNamesake != c.want {
			t.Errorf("%d-byte file: HasNamesake %v, want %v", c.size, as[0].HasNamesake, c.want)
		}
	}
}

// Two different photos of one name and one capture date, one per series, are not one
// series group: they would be stacked, one hidden under the other.
func TestNamesakeImagesOfOneDateAreNotOneGroup(t *testing.T) {
	taken := time.Date(2025, 3, 26, 11, 26, 52, 0, time.UTC)
	part := func(name string, content byte, size int) namedMapFS {
		return namedMapFS{name: name, MapFS: fstest.MapFS{
			collisionDir + "/IMG_0001.jpg":                            {Data: []byte(strings.Repeat(string(content), size))},
			collisionDir + "/IMG_0001.jpg.supplemental-metadata.json": {Data: sidecarJSON("IMG_0001.jpg", taken)},
		}}
	}
	toc, _ := newCollisionTakeoutCmd(t,
		part("takeout-20251016T092000Z-1-001", 'a', 400),
		part("takeout-20251016T092000Z-2-001", 'b', 500))

	gs := browseGroups(t, toc)

	n := 0
	for _, g := range gs {
		n += len(g.Assets)
		if len(g.Assets) > 1 {
			t.Errorf("the two photos are in one group of %d", len(g.Assets))
		}
	}
	if n != 2 {
		t.Errorf("got %d assets, want 2", n)
	}
}

// partOf names the takeout part an asset comes from.
func partOf(a *assets.Asset) string {
	return a.File.FS().(namedMapFS).name
}

// assertNoGroupMixesParts fails when a group holds files of two parts: with namesakes, two
// different photos would be stacked as one.
func assertNoGroupMixesParts(t *testing.T, gs []*assets.Group) {
	t.Helper()
	for _, g := range gs {
		for _, a := range g.Assets[1:] {
			if partOf(a) != partOf(g.Assets[0]) {
				t.Errorf("a group of %d mixes %s and %s", len(g.Assets), partOf(g.Assets[0]), partOf(a))
				break
			}
		}
	}
}

// groupOf returns the sizes of the files in the group that holds the file of the given size.
func groupOf(gs []*assets.Group, size int) []int {
	for _, g := range gs {
		if len(bySize(g.Assets, size)) > 0 {
			sizes := []int{}
			for _, a := range g.Assets {
				sizes = append(sizes, a.FileSize)
			}
			slices.Sort(sizes)
			return sizes
		}
	}
	return nil
}

// The companions of a namesake (its edited copy, its RAW) stay with it, in its own series.
func TestCompanionsOfNamesakesAreGroupedInTheirOwnSeries(t *testing.T) {
	taken := time.Date(2025, 3, 26, 11, 26, 52, 0, time.UTC)
	t.Run("edited copy", func(t *testing.T) {
		s1 := namedMapFS{name: "takeout-20251016T092000Z-1-001", MapFS: fstest.MapFS{
			collisionDir + "/IMG_0001.jpg":                            {Data: []byte(strings.Repeat("a", 100))},
			collisionDir + "/IMG_0001.jpg.supplemental-metadata.json": {Data: sidecarJSON("IMG_0001.jpg", taken)},
		}}
		s2 := namedMapFS{name: "takeout-20251016T092000Z-2-001", MapFS: fstest.MapFS{
			collisionDir + "/IMG_0001.jpg":                            {Data: []byte(strings.Repeat("b", 200))},
			collisionDir + "/IMG_0001-edited.jpg":                     {Data: []byte(strings.Repeat("e", 250))},
			collisionDir + "/IMG_0001.jpg.supplemental-metadata.json": {Data: sidecarJSON("IMG_0001.jpg", taken)},
		}}
		toc, _ := newCollisionTakeoutCmd(t, s1, s2)

		gs := browseGroups(t, toc)

		assertNoGroupMixesParts(t, gs)
		if got := groupOf(gs, 250); !slices.Equal(got, []int{200, 250}) {
			t.Errorf("the edited copy is grouped with %v, want with its original: [200 250]", got)
		}
	})
	t.Run("RAW and JPG", func(t *testing.T) {
		part := func(name string, raw, jpg int) namedMapFS {
			return namedMapFS{name: name, MapFS: fstest.MapFS{
				collisionDir + "/IMG_0001.CR2":                            {Data: []byte(strings.Repeat("r", raw))},
				collisionDir + "/IMG_0001.CR2.supplemental-metadata.json": {Data: sidecarJSON("IMG_0001.CR2", taken)},
				collisionDir + "/IMG_0001.JPG":                            {Data: []byte(strings.Repeat("j", jpg))},
				collisionDir + "/IMG_0001.JPG.supplemental-metadata.json": {Data: sidecarJSON("IMG_0001.JPG", taken)},
			}}
		}
		toc, _ := newCollisionTakeoutCmd(t,
			part("takeout-20251016T092000Z-1-001", 1000, 100),
			part("takeout-20251016T092000Z-2-001", 2000, 200))

		gs := browseGroups(t, toc)

		assertNoGroupMixesParts(t, gs)
		for _, want := range [][]int{{100, 1000}, {200, 2000}} {
			if got := groupOf(gs, want[1]); !slices.Equal(got, want) {
				t.Errorf("the %d-byte RAW is grouped with %v, want %v", want[1], got, want)
			}
		}
	})
}

// The Epson FastFoto grouper groups a scan with its enhanced (_a) and back (_b) copies by
// name alone, whatever the dates. Two different scans of one name, one per series, each
// stay with their own copies.
func TestNamesakeScansKeepTheirOwnEpsonCopies(t *testing.T) {
	part := func(name string, copyName string, size int, taken time.Time) namedMapFS {
		return namedMapFS{name: name, MapFS: fstest.MapFS{
			collisionDir + "/Scan_0001.jpg":                               {Data: []byte(strings.Repeat("s", size))},
			collisionDir + "/Scan_0001.jpg.supplemental-metadata.json":    {Data: sidecarJSON("Scan_0001.jpg", taken)},
			collisionDir + "/" + copyName:                                 {Data: []byte(strings.Repeat("c", size+1))},
			collisionDir + "/" + copyName + ".supplemental-metadata.json": {Data: sidecarJSON(copyName, taken)},
		}}
	}
	toc, _ := newCollisionTakeoutCmd(t,
		part("takeout-20251016T092000Z-1-001", "Scan_0001_a.jpg", 100, time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)),
		part("takeout-20251016T092000Z-2-001", "Scan_0001_b.jpg", 200, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)))
	toc.groupers = []groups.Grouper{epsonfastfoto.Group{}.Group, series.Group} // as with --manage-epson-fastfoto

	gs := browseGroups(t, toc)

	assertNoGroupMixesParts(t, gs)
	for _, want := range [][]int{{100, 101}, {200, 201}} {
		if got := groupOf(gs, want[0]); !slices.Equal(got, want) {
			t.Errorf("the %d-byte scan is grouped with %v, want %v", want[0], got, want)
		}
	}
}

// In a directory split by series, an edit whose sidecar is its original's, in another series
// (Takeout rarely splits them), is grouped with the original: as siblings, the upload never
// takes one for a smaller or bigger version of the other.
func TestEditWithItsOriginalsSidecarInAnotherSeriesStaysWithIt(t *testing.T) {
	taken := time.Date(2025, 3, 26, 11, 26, 52, 0, time.UTC)
	other := time.Date(2025, 7, 1, 9, 0, 0, 0, time.UTC)
	s1 := namedMapFS{name: "takeout-20251016T092000Z-1-001", MapFS: fstest.MapFS{
		collisionDir + "/IMG_3285.MOV":                            {Data: []byte(strings.Repeat("a", 100))},
		collisionDir + "/IMG_3285.MOV.supplemental-metadata.json": {Data: sidecarJSON("IMG_3285.MOV", other)},
		collisionDir + "/X.jpg":                                   {Data: []byte(strings.Repeat("o", 300))},
		collisionDir + "/X.jpg.supplemental-metadata.json":        {Data: sidecarJSON("X.jpg", taken)},
	}}
	s2 := namedMapFS{name: "takeout-20251016T092000Z-2-001", MapFS: fstest.MapFS{
		collisionDir + "/IMG_3285.MOV":                            {Data: []byte(strings.Repeat("b", 200))},
		collisionDir + "/IMG_3285.MOV.supplemental-metadata.json": {Data: sidecarJSON("IMG_3285.MOV", other)},
		collisionDir + "/X-edited.jpg":                            {Data: []byte(strings.Repeat("e", 250))}, // no sidecar of its own
	}}
	toc, _ := newCollisionTakeoutCmd(t, s1, s2)

	gs := browseGroups(t, toc)

	if got := groupOf(gs, 250); !slices.Equal(got, []int{250, 300}) {
		t.Errorf("the edit is grouped with %v, want with its original: [250 300]", got)
	}
}

// The split is decided on the directory's files before the local duplicates are dropped: here
// photo A's JPG was already sent with its album copy, and A's HEIC must still not be paired with
// B, the other photo of the name.
func TestCompanionOfANamesakeSentEarlierIsNotPairedWithTheOther(t *testing.T) {
	taken := time.Date(2025, 3, 26, 11, 26, 52, 0, time.UTC)
	album := "Takeout/Google Photos/Holiday" // sorts before "Photos from 2025"
	s1 := namedMapFS{name: "takeout-20251016T092000Z-1-001", MapFS: fstest.MapFS{
		album + "/X.jpg": {Data: []byte(strings.Repeat("a", 100))},
		album + "/X.jpg.supplemental-metadata.json":         {Data: sidecarJSON("X.jpg", taken)},
		album + "/metadata.json":                            {Data: []byte(`{"title": "Holiday"}`)},
		collisionDir + "/X.jpg":                             {Data: []byte(strings.Repeat("a", 100))},
		collisionDir + "/X.jpg.supplemental-metadata.json":  {Data: sidecarJSON("X.jpg", taken)},
		collisionDir + "/X.HEIC":                            {Data: []byte(strings.Repeat("h", 150))},
		collisionDir + "/X.HEIC.supplemental-metadata.json": {Data: sidecarJSON("X.HEIC", taken)},
	}}
	s2 := namedMapFS{name: "takeout-20251016T092000Z-2-001", MapFS: fstest.MapFS{
		collisionDir + "/X.jpg":                            {Data: []byte(strings.Repeat("b", 200))},
		collisionDir + "/X.jpg.supplemental-metadata.json": {Data: sidecarJSON("X.jpg", taken)},
	}}
	toc, _ := newCollisionTakeoutCmd(t, s1, s2)

	gs := browseGroups(t, toc)

	assertNoGroupMixesParts(t, gs)
}

// A namesake that is a frame of a burst stays in its burst.
func TestNamesakeFrameStaysInItsBurst(t *testing.T) {
	taken := time.Date(2025, 3, 26, 11, 26, 52, 0, time.UTC)
	frame := func(name string, size int) map[string]*fstest.MapFile {
		return map[string]*fstest.MapFile{
			collisionDir + "/" + name:                                 {Data: []byte(strings.Repeat("f", size))},
			collisionDir + "/" + name + ".supplemental-metadata.json": {Data: sidecarJSON(name, taken)},
		}
	}
	s1 := namedMapFS{name: "takeout-20251016T092000Z-1-001", MapFS: fstest.MapFS{}}
	for i, n := range []string{"IMG_0001.jpg", "IMG_0002.jpg", "IMG_0003.jpg"} {
		maps.Copy(s1.MapFS, frame(n, 100+i))
	}
	s2 := namedMapFS{name: "takeout-20251016T092000Z-2-001", MapFS: fstest.MapFS{
		collisionDir + "/IMG_0002.jpg":                            {Data: []byte(strings.Repeat("o", 500))}, // another photo
		collisionDir + "/IMG_0002.jpg.supplemental-metadata.json": {Data: sidecarJSON("IMG_0002.jpg", taken.AddDate(0, 0, 5))},
	}}
	toc, _ := newCollisionTakeoutCmd(t, s1, s2)
	toc.groupers = []groups.Grouper{burst.Group, series.Group} // as with --manage-burst

	gs := browseGroups(t, toc)

	assertNoGroupMixesParts(t, gs)
	if got := groupOf(gs, 101); !slices.Equal(got, []int{100, 101, 102}) {
		t.Errorf("the burst's second frame is grouped with %v, want the three frames [100 101 102]", got)
	}
}

// Rarely, Takeout puts a sidecar in another series than its photo (4 cases in
// ~68 000 sidecars of real exports). The photo gets that sidecar.
func TestSidecarOnlyInAnotherSeriesIsUsed(t *testing.T) {
	taken := time.Date(2019, 7, 14, 20, 3, 11, 0, time.UTC)
	s1 := namedMapFS{name: "takeout-20251016T092000Z-1-001", MapFS: fstest.MapFS{
		collisionDir + "/VID_20190714_200311.mp4.supplemental-metadata.json": {Data: sidecarJSON("VID_20190714_200311.mp4", taken)},
	}}
	s2 := namedMapFS{name: "takeout-20251016T092000Z-2-001", MapFS: fstest.MapFS{
		collisionDir + "/VID_20190714_200311.mp4": {Data: []byte(strings.Repeat("v", 300))},
	}}
	toc, _ := newCollisionTakeoutCmd(t, s1, s2)

	all := browseAll(t, toc)

	if got := dateOf(t, all, 300); !got.Equal(taken) {
		t.Errorf("video: date %v, want %v from the sidecar in series 1", got, taken)
	}
}

// A sidecar in another series belongs to the file of the same name in that series,
// which is a different photo. A photo without its own sidecar does not borrow it.
func TestSidecarOfTheSameNameInAnotherSeriesIsNotBorrowed(t *testing.T) {
	takenA := time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC)
	s1 := namedMapFS{name: "takeout-20251016T092000Z-1-001", MapFS: fstest.MapFS{
		collisionDir + "/IMG_0001(1).jpg":                            {Data: []byte(strings.Repeat("a", 100))},
		collisionDir + "/IMG_0001.jpg.supplemental-metadata(1).json": {Data: sidecarJSON("IMG_0001.jpg", takenA)},
	}}
	s2 := namedMapFS{name: "takeout-20251016T092000Z-2-001", MapFS: fstest.MapFS{
		collisionDir + "/IMG_0001(1).jpg": {Data: []byte(strings.Repeat("b", 200))}, // its sidecar is missing
	}}
	toc, _ := newCollisionTakeoutCmd(t, s1, s2)

	all := browseAll(t, toc)

	b := bySize(all, 200)
	if len(b) != 1 {
		t.Fatalf("photo without a sidecar: got %d assets, want 1 (KeepJSONLess)", len(b))
	}
	if b[0].FromApplication != nil {
		t.Errorf("photo without a sidecar got metadata %+v, which belongs to the other series' photo", b[0].FromApplication)
	}
}

// Matchers keep their priority across series: a photo whose exact sidecar is in
// another series gets it, rather than a looser match in its own series.
func TestExactSidecarInAnotherSeriesBeatsALooseOneInTheOwnSeries(t *testing.T) {
	takenPortrait := time.Date(2022, 4, 5, 9, 1, 23, 0, time.UTC)
	takenSibling := time.Date(2022, 4, 5, 9, 2, 0, 0, time.UTC)
	s1 := namedMapFS{name: "takeout-20251016T092000Z-1-001", MapFS: fstest.MapFS{
		collisionDir + "/PXL_20220405_090123740.PORTRAIT.jpg.supplemental-metadata.json": {Data: sidecarJSON("PXL_20220405_090123740.PORTRAIT.jpg", takenPortrait)},
	}}
	s2 := namedMapFS{name: "takeout-20251016T092000Z-2-001", MapFS: fstest.MapFS{
		collisionDir + "/PXL_20220405_090123740.PORTRAIT.jpg":                   {Data: []byte(strings.Repeat("p", 120))},
		collisionDir + "/PXL_20220405_090123740.jpg":                            {Data: []byte(strings.Repeat("s", 130))},
		collisionDir + "/PXL_20220405_090123740.jpg.supplemental-metadata.json": {Data: sidecarJSON("PXL_20220405_090123740.jpg", takenSibling)},
	}}
	toc, _ := newCollisionTakeoutCmd(t, s1, s2)

	all := browseAll(t, toc)

	if got := dateOf(t, all, 120); !got.Equal(takenPortrait) {
		t.Errorf("portrait: date %v, want %v from its exact sidecar in series 1", got, takenPortrait)
	}
	if got := dateOf(t, all, 130); !got.Equal(takenSibling) {
		t.Errorf("sibling: date %v, want %v", got, takenSibling)
	}
}

// A RAW and its JPG that land in different series are still one group.
func TestRawAndJpgInDifferentSeriesAreStillGrouped(t *testing.T) {
	taken := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	s1 := namedMapFS{name: "takeout-20251016T092000Z-1-001", MapFS: fstest.MapFS{
		collisionDir + "/DSC_0100.ARW":                            {Data: []byte(strings.Repeat("r", 400))},
		collisionDir + "/DSC_0100.ARW.supplemental-metadata.json": {Data: sidecarJSON("DSC_0100.ARW", taken)},
	}}
	s2 := namedMapFS{name: "takeout-20251016T092000Z-2-001", MapFS: fstest.MapFS{
		collisionDir + "/DSC_0100.JPG":                            {Data: []byte(strings.Repeat("j", 150))},
		collisionDir + "/DSC_0100.JPG.supplemental-metadata.json": {Data: sidecarJSON("DSC_0100.JPG", taken)},
	}}
	toc, _ := newCollisionTakeoutCmd(t, s1, s2)

	gs := browseGroups(t, toc)

	sizes := []int{}
	for _, g := range gs {
		if len(g.Assets) == 2 {
			return
		}
		sizes = append(sizes, len(g.Assets))
	}
	t.Errorf("RAW and JPG not grouped together: group sizes %v, want one group of 2", sizes)
}

func TestTakeoutSeriesFromPartName(t *testing.T) {
	for _, c := range []struct{ part, want string }{
		{"takeout-20261016T092209Z-3-017", "takeout-20261016T092209Z-3"}, // a zip part, as fshelper names it
		{"takeout-20261016T142712Z-001", "takeout-20261016T142712Z"},     // a small export has a single series
		{"Takeout", ""},   // a name that is not a takeout part says nothing:
		{"Takeout 2", ""}, // all such parts share one series, as before
	} {
		if got := takeoutSeries(namedMapFS{name: c.part}); got != c.want {
			t.Errorf("takeoutSeries(%q) = %q, want %q", c.part, got, c.want)
		}
	}
}

// The same file with its sidecar in two takeouts (two exports, or a part downloaded
// twice) is still one photo: the second copy is a local duplicate.
func TestIdenticalFileInTwoTakeoutsIsALocalDuplicate(t *testing.T) {
	taken := time.Date(2020, 5, 1, 8, 0, 0, 0, time.UTC)
	files := func() fstest.MapFS {
		return fstest.MapFS{
			collisionDir + "/IMG_0001.jpg":                            {Data: []byte(strings.Repeat("a", 100))},
			collisionDir + "/IMG_0001.jpg.supplemental-metadata.json": {Data: sidecarJSON("IMG_0001.jpg", taken)},
		}
	}
	t1 := namedMapFS{name: "takeout-20251016T092000Z-001", MapFS: files()}
	t2 := namedMapFS{name: "takeout-20251016T092209Z-001", MapFS: files()}
	toc, rec := newCollisionTakeoutCmd(t, t1, t2)

	all := browseAll(t, toc)

	kept := bySize(all, 100)
	if len(kept) != 1 {
		t.Fatalf("got %d assets for the photo, want 1", len(kept))
	}
	if kept[0].HasNamesake {
		t.Error("the photo is marked as having a namesake, but its copy was a duplicate")
	}
	if n := rec.GetCounts()[fileevent.DiscardedLocalDuplicate]; n != 1 {
		t.Errorf("%d local duplicate(s), want 1", n)
	}
}

// Takeout numbers name clashes "(1)", "(2)"... within a series, so an indexed name
// never borrows a sidecar from another series: there it belongs to a different photo.
// Here an unrelated video would take the sidecar of the still with the same index.
func TestIndexedNameDoesNotBorrowASidecarFromAnotherSeries(t *testing.T) {
	takenStill := time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC)
	s1 := namedMapFS{name: "takeout-20251016T092000Z-1-001", MapFS: fstest.MapFS{
		collisionDir + "/IMG_0001(1).HEIC":                            {Data: []byte(strings.Repeat("h", 100))},
		collisionDir + "/IMG_0001.HEIC.supplemental-metadata(1).json": {Data: sidecarJSON("IMG_0001.HEIC", takenStill)},
	}}
	s2 := namedMapFS{name: "takeout-20251016T092000Z-2-001", MapFS: fstest.MapFS{
		collisionDir + "/IMG_0001(1).MP4": {Data: []byte(strings.Repeat("m", 300))}, // an unrelated video
	}}
	toc, _ := newCollisionTakeoutCmd(t, s1, s2)

	all := browseAll(t, toc)

	v := bySize(all, 300)
	if len(v) != 1 {
		t.Fatalf("video: got %d assets, want 1 (KeepJSONLess)", len(v))
	}
	if v[0].FromApplication != nil {
		t.Errorf("video got the series-1 still's metadata %+v", v[0].FromApplication)
	}
}

// Two different files with the same name and date from two series come out in the
// same order on every run.
func TestSameNameFilesFromTwoSeriesComeOutInAStableOrder(t *testing.T) {
	taken := time.Date(2025, 3, 26, 11, 26, 52, 0, time.UTC)
	parts := func() []fs.FS {
		return []fs.FS{
			namedMapFS{name: "takeout-20251016T092000Z-1-001", MapFS: fstest.MapFS{
				collisionDir + "/IMG_1874.MOV":                            {Data: []byte(strings.Repeat("a", 100))},
				collisionDir + "/IMG_1874.MOV.supplemental-metadata.json": {Data: sidecarJSON("IMG_1874.MOV", taken)},
			}},
			namedMapFS{name: "takeout-20251016T092000Z-2-001", MapFS: fstest.MapFS{
				collisionDir + "/IMG_1874.MOV":                            {Data: []byte(strings.Repeat("b", 200))},
				collisionDir + "/IMG_1874.MOV.supplemental-metadata.json": {Data: sidecarJSON("IMG_1874.MOV", taken)},
			}},
		}
	}
	order := func() string {
		toc, _ := newCollisionTakeoutCmd(t, parts()...)
		all := browseAll(t, toc)
		sizes := make([]string, 0, len(all))
		for _, a := range all {
			sizes = append(sizes, fmt.Sprint(a.FileSize))
		}
		return strings.Join(sizes, ",")
	}
	first := order()
	for range 50 {
		if got := order(); got != first {
			t.Fatalf("order changed between runs: %s then %s", first, got)
		}
	}
}

// Two different files with the same name and the same size in two series are both
// kept: equal size is not proof of equal content (uncompressed RAW files, for one).
func TestSameNameAndSizeButDifferentContentInTwoSeriesAreBothKept(t *testing.T) {
	takenS1 := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	takenS2 := time.Date(2024, 7, 9, 15, 0, 0, 0, time.UTC)
	s1 := namedMapFS{name: "takeout-20251016T092000Z-1-001", MapFS: fstest.MapFS{
		collisionDir + "/DSC_0100(1).ARW":                            {Data: []byte(strings.Repeat("x", 400))},
		collisionDir + "/DSC_0100.ARW.supplemental-metadata(1).json": {Data: sidecarJSON("DSC_0100.ARW", takenS1)},
	}}
	s2 := namedMapFS{name: "takeout-20251016T092000Z-2-001", MapFS: fstest.MapFS{
		collisionDir + "/DSC_0100(1).ARW":                            {Data: []byte(strings.Repeat("y", 400))},
		collisionDir + "/DSC_0100.ARW.supplemental-metadata(1).json": {Data: sidecarJSON("DSC_0100.ARW", takenS2)},
	}}
	toc, rec := newCollisionTakeoutCmd(t, s1, s2)

	all := browseAll(t, toc)

	if got := len(bySize(all, 400)); got != 2 {
		t.Errorf("got %d assets, want 2: the files differ although their sizes are equal", got)
	}
	if n := rec.GetCounts()[fileevent.DiscardedLocalDuplicate]; n != 0 {
		t.Errorf("%d local duplicate(s), want 0", n)
	}
}

// Parts whose names do not follow the takeout pattern (folders extracted as
// "Takeout", "Takeout 2") are matched together, as before: a Live Photo video in one
// still gets the sidecar of its still in the other.
func TestPartsWithoutTakeoutNamesAreMatchedTogether(t *testing.T) {
	taken := time.Date(2023, 9, 1, 9, 0, 0, 0, time.UTC)
	p1 := namedMapFS{name: "Takeout", MapFS: fstest.MapFS{
		collisionDir + "/IMG_4488.HEIC":                            {Data: []byte(strings.Repeat("h", 100))},
		collisionDir + "/IMG_4488.HEIC.supplemental-metadata.json": {Data: sidecarJSON("IMG_4488.HEIC", taken)},
	}}
	p2 := namedMapFS{name: "Takeout 2", MapFS: fstest.MapFS{
		collisionDir + "/IMG_4488.MP4": {Data: []byte(strings.Repeat("m", 300))},
	}}
	toc, _ := newCollisionTakeoutCmd(t, p1, p2)

	all := browseAll(t, toc)

	if got := dateOf(t, all, 300); !got.Equal(taken) {
		t.Errorf("Live Photo video: date %v, want %v from its still's sidecar", got, taken)
	}
}

// A part without a takeout name (here a folder some parts were unzipped into, passed
// along with the remaining zips) may be of any series, so an indexed name does not cross
// between it and a takeout part either. By names alone this case cannot be told from a
// Live Photo split over the two parts; a missing date is reported, another photo's date
// would not be.
func TestIndexedNameDoesNotCrossAPartWithoutTakeoutName(t *testing.T) {
	takenStill := time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC)
	extracted := namedMapFS{name: "unzipped", MapFS: fstest.MapFS{
		collisionDir + "/IMG_0001(1).HEIC":                            {Data: []byte(strings.Repeat("h", 100))},
		collisionDir + "/IMG_0001.HEIC.supplemental-metadata(1).json": {Data: sidecarJSON("IMG_0001.HEIC", takenStill)},
	}}
	s2 := namedMapFS{name: "takeout-20251016T092000Z-2-001", MapFS: fstest.MapFS{
		collisionDir + "/IMG_0001(1).MP4": {Data: []byte(strings.Repeat("m", 300))}, // an unrelated video
	}}
	toc, _ := newCollisionTakeoutCmd(t, extracted, s2)

	all := browseAll(t, toc)

	if got := dateOf(t, all, 100); !got.Equal(takenStill) {
		t.Errorf("still: date %v, want %v from its own sidecar", got, takenStill)
	}
	v := bySize(all, 300)
	if len(v) != 1 {
		t.Fatalf("video: got %d assets, want 1 (KeepJSONLess)", len(v))
	}
	if v[0].FromApplication != nil {
		t.Errorf("video got the extracted still's metadata %+v", v[0].FromApplication)
	}
}

// One path can be in more than two series (one path of the real exports is in four).
// A copy is compared with every earlier one: the fourth copy here equals the second,
// neither the first nor the latest, and is the only duplicate.
func TestFourthCopyOfAPathIsComparedWithEachEarlierOne(t *testing.T) {
	taken := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	part := func(name string, content byte) namedMapFS {
		return namedMapFS{name: name, MapFS: fstest.MapFS{
			collisionDir + "/DSC_0100(1).ARW":                            {Data: []byte(strings.Repeat(string(content), 400))},
			collisionDir + "/DSC_0100.ARW.supplemental-metadata(1).json": {Data: sidecarJSON("DSC_0100.ARW", taken)},
		}}
	}
	toc, rec := newCollisionTakeoutCmd(t,
		part("takeout-20251016T092000Z-1-001", 'x'),
		part("takeout-20251016T092000Z-2-001", 'y'),
		part("takeout-20251016T092000Z-3-001", 'z'),
		part("takeout-20251016T092000Z-4-001", 'y'))

	all := browseAll(t, toc)

	if got := len(bySize(all, 400)); got != 3 {
		t.Errorf("got %d assets, want 3: the three different files, the fourth being a copy of the second", got)
	}
	if n := rec.GetCounts()[fileevent.DiscardedLocalDuplicate]; n != 1 {
		t.Errorf("%d local duplicate(s), want 1", n)
	}
}

// Parts of one series share their sidecars: the video half of a Live Photo in part
// -1-002 gets the sidecar of its still in part -1-001.
func TestLivePhotoSplitOverTwoPartsOfOneSeries(t *testing.T) {
	taken := time.Date(2023, 9, 1, 9, 0, 0, 0, time.UTC)
	p1 := namedMapFS{name: "takeout-20251016T092000Z-1-001", MapFS: fstest.MapFS{
		collisionDir + "/IMG_4488(1).HEIC":                            {Data: []byte(strings.Repeat("h", 100))},
		collisionDir + "/IMG_4488.HEIC.supplemental-metadata(1).json": {Data: sidecarJSON("IMG_4488.HEIC", taken)},
	}}
	p2 := namedMapFS{name: "takeout-20251016T092000Z-1-002", MapFS: fstest.MapFS{
		collisionDir + "/IMG_4488(1).MP4": {Data: []byte(strings.Repeat("m", 300))},
	}}
	toc, _ := newCollisionTakeoutCmd(t, p1, p2)

	all := browseAll(t, toc)

	if got := dateOf(t, all, 300); !got.Equal(taken) {
		t.Errorf("Live Photo video: date %v, want %v from its still's sidecar", got, taken)
	}
}

// A sidecar name present in several series is ambiguous. Here two long names are
// truncated by Takeout to the same 46-character sidecar name, in series 1 (photo A)
// and series 3 (photo B, whose file is in series 2). B must not get A's date.
func TestAmbiguousSidecarNameIsNotBorrowed(t *testing.T) {
	prefix := strings.Repeat("p", 46)
	takenA := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	takenB := time.Date(2025, 6, 6, 0, 0, 0, 0, time.UTC)
	s1 := namedMapFS{name: "takeout-20251016T092000Z-1-001", MapFS: fstest.MapFS{
		collisionDir + "/" + prefix + "_AAA.jpg": {Data: []byte(strings.Repeat("a", 100))},
		collisionDir + "/" + prefix + ".json":    {Data: sidecarJSON(prefix+"_AAA.jpg", takenA)},
	}}
	s2 := namedMapFS{name: "takeout-20251016T092000Z-2-001", MapFS: fstest.MapFS{
		collisionDir + "/" + prefix + "_BBB.jpg": {Data: []byte(strings.Repeat("b", 200))},
	}}
	s3 := namedMapFS{name: "takeout-20251016T092000Z-3-001", MapFS: fstest.MapFS{
		collisionDir + "/" + prefix + ".json": {Data: sidecarJSON(prefix+"_BBB.jpg", takenB)},
	}}
	toc, _ := newCollisionTakeoutCmd(t, s1, s2, s3)

	all := browseAll(t, toc)

	b := bySize(all, 200)
	if len(b) != 1 {
		t.Fatalf("photo B: got %d assets, want 1", len(b))
	}
	if md := b[0].FromApplication; md != nil && md.DateTaken.Equal(takenA) {
		t.Errorf("photo B got photo A's date %v", md.DateTaken)
	}
}
