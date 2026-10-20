package gp

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
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

	if got := len(bySize(all, 100)); got != 1 {
		t.Errorf("got %d assets for the photo, want 1", got)
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
