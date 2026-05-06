package flickr

import (
	"context"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/assettracker"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/fileprocessor"
)

// TestExtractPhotoID verifies both filename patterns found in real Flickr exports.
func TestExtractPhotoID(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantID   string
		wantOK   bool
	}{
		// Primary pattern: slug_ID_o.ext (current Flickr format)
		{
			name:     "primary pattern simple",
			filename: "my-photo_12345678_o.jpg",
			wantID:   "12345678",
			wantOK:   true,
		},
		{
			name:     "primary pattern multi-word slug",
			filename: "sunset-at-the-beach_987654321_o.jpeg",
			wantID:   "987654321",
			wantOK:   true,
		},
		{
			name:     "primary pattern with directory prefix stripped",
			filename: "some/dir/photo_42_o.png",
			wantID:   "42",
			wantOK:   true,
		},
		// Plan table: slug_ID_o.ext variations
		{
			name:     "plan table - photo_spring_67890_o.jpg",
			filename: "photo_spring_67890_o.jpg",
			wantID:   "67890",
			wantOK:   true,
		},
		{
			name:     "plan table - photo_title_12345_o.png",
			filename: "photo_title_12345_o.png",
			wantID:   "12345",
			wantOK:   true,
		},
		// Fallback pattern: ID_hash_o.ext (older Flickr format)
		{
			name:     "fallback pattern",
			filename: "12345678_ab1cd2ef3a_o.jpg",
			wantID:   "12345678",
			wantOK:   true,
		},
		{
			name:     "fallback pattern long hash",
			filename: "987654321_0123456789ab_o.jpg",
			wantID:   "987654321",
			wantOK:   true,
		},
		// Plan table: fallback ID_hash_o.ext
		{
			name:     "plan table - 98765_a1b2c3d4_o.jpg",
			filename: "98765_a1b2c3d4_o.jpg",
			wantID:   "98765",
			wantOK:   true,
		},
		// Non-matching inputs
		{
			name:     "no match - plain name",
			filename: "photo.jpg",
			wantID:   "",
			wantOK:   false,
		},
		{
			name:     "no match - missing _o suffix",
			filename: "photo_12345_orig.jpg",
			wantID:   "",
			wantOK:   false,
		},
		{
			name:     "no match - empty string",
			filename: "",
			wantID:   "",
			wantOK:   false,
		},
		// Plan table: non-matching inputs
		{
			name:     "plan table - some_file.jpg",
			filename: "some_file.jpg",
			wantID:   "",
			wantOK:   false,
		},
		{
			name:     "plan table - readme.txt",
			filename: "readme.txt",
			wantID:   "",
			wantOK:   false,
		},
		// Plan table: 12345_o.jpg — missing hash segment for fallback, no slug for primary
		{
			name:     "plan table - 12345_o.jpg no hash segment",
			filename: "12345_o.jpg",
			wantID:   "",
			wantOK:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotID, gotOK := extractPhotoID(tc.filename)
			if gotOK != tc.wantOK {
				t.Errorf("extractPhotoID(%q): ok = %v, want %v", tc.filename, gotOK, tc.wantOK)
			}
			if gotID != tc.wantID {
				t.Errorf("extractPhotoID(%q): id = %q, want %q", tc.filename, gotID, tc.wantID)
			}
		})
	}
}

// makeFS is a helper that builds an in-memory FS with the given file names present.
func makeFS(files ...string) fs.FS {
	m := fstest.MapFS{}
	for _, f := range files {
		m[f] = &fstest.MapFile{}
	}
	return m
}

// TestClassifyArchives verifies the metadata / image FS partitioning logic.
func TestClassifyArchives(t *testing.T) {
	metaFS := makeFS("albums.json", "photo_data.json")
	imageFS1 := makeFS("photo_12345_o.jpg", "photo_67890_o.jpg")
	imageFS2 := makeFS("photo_11111_o.jpg")

	t.Run("one meta one image", func(t *testing.T) {
		meta, imgs, err := classifyArchives([]fs.FS{metaFS, imageFS1})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if meta == nil {
			t.Error("expected non-nil metaFS")
		}
		// Verify the returned metaFS is the correct one — albums.json must be accessible.
		if _, statErr := fs.Stat(meta, "albums.json"); statErr != nil {
			t.Errorf("returned metaFS does not contain albums.json: %v", statErr)
		}
		if len(imgs) != 1 {
			t.Errorf("got %d image FSes, want 1", len(imgs))
		}
	})

	t.Run("one meta two image", func(t *testing.T) {
		meta, imgs, err := classifyArchives([]fs.FS{metaFS, imageFS1, imageFS2})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if meta == nil {
			t.Error("expected non-nil metaFS")
		}
		// Verify the returned metaFS is the correct one.
		if _, statErr := fs.Stat(meta, "albums.json"); statErr != nil {
			t.Errorf("returned metaFS does not contain albums.json: %v", statErr)
		}
		if len(imgs) != 2 {
			t.Errorf("got %d image FSes, want 2", len(imgs))
		}
	})

	t.Run("no metadata archive", func(t *testing.T) {
		_, _, err := classifyArchives([]fs.FS{imageFS1, imageFS2})
		if err == nil {
			t.Error("expected error for missing metadata archive, got nil")
		}
		if err != nil && !strings.Contains(err.Error(), "no metadata archive") {
			t.Errorf("error message %q should contain %q", err.Error(), "no metadata archive")
		}
	})

	t.Run("multiple metadata archives", func(t *testing.T) {
		meta2 := makeFS("albums.json")
		_, _, err := classifyArchives([]fs.FS{metaFS, meta2, imageFS1})
		if err == nil {
			t.Error("expected error for multiple metadata archives, got nil")
		}
		if err != nil && !strings.Contains(err.Error(), "multiple metadata archives") {
			t.Errorf("error message %q should contain %q", err.Error(), "multiple metadata archives")
		}
	})

	t.Run("only metadata archive no images", func(t *testing.T) {
		meta, imgs, err := classifyArchives([]fs.FS{metaFS})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if meta == nil {
			t.Error("expected non-nil metaFS")
		}
		if len(imgs) != 0 {
			t.Errorf("got %d image FSes, want 0", len(imgs))
		}
	})

	t.Run("single image-only FS no metadata", func(t *testing.T) {
		_, _, err := classifyArchives([]fs.FS{imageFS1})
		if err == nil {
			t.Error("expected error for single image-only FS, got nil")
		}
		if err != nil && !strings.Contains(err.Error(), "no metadata archive") {
			t.Errorf("error message %q should contain %q", err.Error(), "no metadata archive")
		}
	})

	t.Run("empty input", func(t *testing.T) {
		_, _, err := classifyArchives([]fs.FS{})
		if err == nil {
			t.Error("expected error for empty FS list, got nil")
		}
	})
}

// newTestFlickrCmd constructs a FlickrCmd with a real FileProcessor backed by
// in-memory tracker/recorder — no app.Application needed for unit tests.
func newTestFlickrCmd(metaFS fs.FS, imageFSes ...fs.FS) *FlickrCmd {
	recorder := fileevent.NewRecorder(nil)
	tracker := assettracker.New()
	proc := fileprocessor.New(tracker, recorder)

	fsyss := make([]fs.FS, 0, 1+len(imageFSes))
	fsyss = append(fsyss, metaFS)
	fsyss = append(fsyss, imageFSes...)

	return &FlickrCmd{
		processor: proc,
		fsyss:     fsyss,
	}
}

// drainBrowse runs Browse() and returns all assets collected across all groups.
func drainBrowse(f *FlickrCmd) ([]*assets.Asset, int) {
	ctx := context.Background()
	ch := f.Browse(ctx)
	var all []*assets.Asset
	groupCount := 0
	for g := range ch {
		if g != nil && len(g.Assets) > 0 {
			groupCount++
			all = append(all, g.Assets...)
		}
	}
	return all, groupCount
}

// TestBrowse validates Browse() end-to-end using fstest.MapFS inline fixtures.
func TestBrowse(t *testing.T) {
	const photoJSON = `{
		"id": "12345678",
		"name": "Sunset at the Beach",
		"description": "A beautiful sunset",
		"date_taken": "2009-08-02 14:05:00",
		"date_upload": "1249204972",
		"tags": [{"tag": "travel"}, {"tag": "sunset"}],
		"albums": []
	}`
	const albumJSON = `{
		"albums": [{
			"id": "72157XXXXXXX",
			"title": "Vacation 2009",
			"description": "",
			"photos": ["12345678"]
		}]
	}`

	t.Run("happy path - title, date, tags, albums", func(t *testing.T) {
		metaFS := fstest.MapFS{
			"photo_12345678.json": {Data: []byte(photoJSON)},
			"albums.json":         {Data: []byte(albumJSON)},
		}
		imageFS := fstest.MapFS{
			"my-photo_12345678_o.jpg": {Data: []byte{0xFF, 0xD8}},
		}

		assetList, groupCount := drainBrowse(newTestFlickrCmd(metaFS, imageFS))

		if groupCount != 1 {
			t.Fatalf("expected 1 group, got %d", groupCount)
		}
		if len(assetList) != 1 {
			t.Fatalf("expected 1 asset, got %d", len(assetList))
		}

		a := assetList[0]
		wantDate := time.Date(2009, 8, 2, 14, 5, 0, 0, time.Local)
		if !a.CaptureDate.Equal(wantDate) {
			t.Errorf("CaptureDate = %v, want %v", a.CaptureDate, wantDate)
		}
		if a.OriginalFileName != "Sunset at the Beach" {
			t.Errorf("OriginalFileName = %q, want %q", a.OriginalFileName, "Sunset at the Beach")
		}
		if len(a.Tags) != 2 {
			t.Errorf("len(Tags) = %d, want 2", len(a.Tags))
		} else {
			tagValues := make(map[string]bool, len(a.Tags))
			for _, tag := range a.Tags {
				tagValues[tag.Value] = true
			}
			if !tagValues["travel"] {
				t.Errorf("tag 'travel' not found in %v", a.Tags)
			}
			if !tagValues["sunset"] {
				t.Errorf("tag 'sunset' not found in %v", a.Tags)
			}
		}
		if len(a.Albums) != 1 {
			t.Errorf("len(Albums) = %d, want 1", len(a.Albums))
		} else if a.Albums[0].Title != "Vacation 2009" {
			t.Errorf("Albums[0].Title = %q, want %q", a.Albums[0].Title, "Vacation 2009")
		}
	})

	t.Run("image with no JSON - still emitted, no crash", func(t *testing.T) {
		metaFS := fstest.MapFS{
			"albums.json": {Data: []byte(`{"albums":[]}`)},
		}
		imageFS := fstest.MapFS{
			"my-photo_99999999_o.jpg": {Data: []byte{0xFF, 0xD8}},
		}

		assetList, _ := drainBrowse(newTestFlickrCmd(metaFS, imageFS))

		if len(assetList) != 1 {
			t.Fatalf("expected 1 asset, got %d", len(assetList))
		}
		a := assetList[0]
		if len(a.Tags) != 0 {
			t.Errorf("expected 0 tags, got %d: %v", len(a.Tags), a.Tags)
		}
		if len(a.Albums) != 0 {
			t.Errorf("expected 0 albums, got %d: %v", len(a.Albums), a.Albums)
		}
	})

	t.Run("JSON with no image - no group emitted", func(t *testing.T) {
		metaFS := fstest.MapFS{
			"photo_12345678.json": {Data: []byte(photoJSON)},
			"albums.json":         {Data: []byte(albumJSON)},
		}
		// Empty imageFS — no image files, so no catalog entries, no groups.
		imageFS := fstest.MapFS{}

		assetList, _ := drainBrowse(newTestFlickrCmd(metaFS, imageFS))

		if len(assetList) != 0 {
			t.Errorf("expected 0 assets, got %d", len(assetList))
		}
	})

	t.Run("date_taken fallback to date_upload", func(t *testing.T) {
		const noDateJSON = `{
			"id": "12345678",
			"name": "No Date Photo",
			"description": "",
			"date_taken": "",
			"date_upload": "1249204972",
			"tags": [],
			"albums": []
		}`
		metaFS := fstest.MapFS{
			"photo_12345678.json": {Data: []byte(noDateJSON)},
			"albums.json":         {Data: []byte(`{"albums":[]}`)},
		}
		imageFS := fstest.MapFS{
			"my-photo_12345678_o.jpg": {Data: []byte{0xFF, 0xD8}},
		}

		assetList, _ := drainBrowse(newTestFlickrCmd(metaFS, imageFS))

		if len(assetList) != 1 {
			t.Fatalf("expected 1 asset, got %d", len(assetList))
		}
		wantDate := time.Unix(1249204972, 0).In(time.Local)
		if !assetList[0].CaptureDate.Equal(wantDate) {
			t.Errorf("CaptureDate = %v, want %v", assetList[0].CaptureDate, wantDate)
		}
	})

	t.Run("context cancelled - Browse exits cleanly", func(t *testing.T) {
		metaFS := fstest.MapFS{
			"photo_12345678.json": {Data: []byte(photoJSON)},
			"albums.json":         {Data: []byte(albumJSON)},
		}
		imageFS := fstest.MapFS{
			"my-photo_12345678_o.jpg": {Data: []byte{0xFF, 0xD8}},
		}

		f := newTestFlickrCmd(metaFS, imageFS)
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // cancel immediately before Browse even starts

		ch := f.Browse(ctx)

		done := make(chan struct{})
		go func() {
			for range ch {
			}
			close(done)
		}()

		select {
		case <-done:
			// Browse() channel closed cleanly — pass
		case <-time.After(3 * time.Second):
			t.Fatal("Browse() did not close the channel within 3s after context cancellation")
		}
	})
}
