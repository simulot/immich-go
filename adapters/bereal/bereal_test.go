package bereal

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/stretchr/testify/assert"
)

func dataDirFromThisFile() string {
	_, fp, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(fp), "DATA")
}

func TestPathResolutionVariants(t *testing.T) {
	dataDir := dataDirFromThisFile()

	log := &app.Log{}
	log.SetLogWriter(io.Discard)

	ba, err := NewBeRealAdapter(os.DirFS(dataDir), log)
	assert.NoError(t, err)

	// CountAssets should find 2 assets (front + back)
	cnt, err := ba.CountAssets()
	assert.NoError(t, err)
	assert.Equal(t, 2, cnt)

	// Browse and ensure the asset filenames from DATA are resolved
	ch := ba.Browse(context.Background())
	found := map[string]bool{}
	for g := range ch {
		for _, a := range g.Assets {
			found[a.OriginalFileName] = true
		}
	}
	assert.True(t, found["6lNLfy.webp"])      // front
	assert.True(t, found["1111555JKKL.webp"]) // back
}

func TestPairingCreatesTwoAssets(t *testing.T) {
	dataDir := dataDirFromThisFile()

	log := &app.Log{}
	log.SetLogWriter(io.Discard)

	ba, err := NewBeRealAdapter(os.DirFS(dataDir), log)
	assert.NoError(t, err)

	cnt, err := ba.CountAssets()
	assert.NoError(t, err)
	assert.Equal(t, 2, cnt)
}

func TestLocationIsAddedToAsset(t *testing.T) {
	dataDir := dataDirFromThisFile()

	log := &app.Log{}
	log.SetLogWriter(io.Discard)

	ba, err := NewBeRealAdapter(os.DirFS(dataDir), log)
	assert.NoError(t, err)

	// Browse and check that location is properly added to assets
	ch := ba.Browse(context.Background())
	assetCount := 0
	for g := range ch {
		for _, a := range g.Assets {
			assetCount++
			// Verify location coordinates are present
			assert.NotZero(t, a.Latitude, "latitude should not be zero")
			assert.NotZero(t, a.Longitude, "longitude should not be zero")
			// Check specific expected values from test data
			assert.InDelta(t, 52.8912, a.Latitude, 0.001, "latitude should match test data")
			assert.InDelta(t, 13.0007, a.Longitude, 0.001, "longitude should match test data")
		}
	}
	assert.Equal(t, 2, assetCount, "should have 2 assets with location data")
}

func TestMetadataIsMarkedAsFromApplication(t *testing.T) {
	dataDir := dataDirFromThisFile()

	log := &app.Log{}
	log.SetLogWriter(io.Discard)

	ba, err := NewBeRealAdapter(os.DirFS(dataDir), log)
	assert.NoError(t, err)

	// Browse and check that FromApplication metadata is set (required for location to be uploaded)
	ch := ba.Browse(context.Background())
	assetCount := 0
	for g := range ch {
		for _, a := range g.Assets {
			assetCount++
			// The key issue: FromApplication must be set for metadata to be uploaded
			// If FromApplication is nil, the location won't be sent to the server
			assert.NotNil(t, a.FromApplication, "FromApplication metadata should be set for location to be uploaded")
			if a.FromApplication != nil {
				assert.NotZero(t, a.FromApplication.Latitude, "FromApplication latitude should not be zero")
				assert.NotZero(t, a.FromApplication.Longitude, "FromApplication longitude should not be zero")
			}
		}
	}
	assert.Equal(t, 2, assetCount, "should have 2 assets with application metadata")
}

func TestAssetsAreGroupedForStacking(t *testing.T) {
	dataDir := dataDirFromThisFile()

	log := &app.Log{}
	log.SetLogWriter(io.Discard)

	ba, err := NewBeRealAdapter(os.DirFS(dataDir), log)
	assert.NoError(t, err)

	// Browse and check that assets are grouped together (front + back should be in same group)
	ch := ba.Browse(context.Background())
	groupCount := 0
	for g := range ch {
		groupCount++
		// Front and back images should be in the same group for stacking
		assert.Equal(t, 2, len(g.Assets), "each group should have 2 assets (front + back)")

		// Check group type - using Grouping field instead of Type
		assert.Equal(t, assets.GroupByDualCamera, g.Grouping, "group should be of type GroupByDualCamera for stacking")

		// Check that both assets have location
		for i, a := range g.Assets {
			assert.NotZero(t, a.Latitude, "asset %d latitude should not be zero", i)
			assert.NotZero(t, a.Longitude, "asset %d longitude should not be zero", i)
		}
	}
	assert.Equal(t, 1, groupCount, "should have 1 group (front + back pair)")
}

func TestCaptionIsPushedAsDescription(t *testing.T) {
	dataDir := dataDirFromThisFile()

	log := &app.Log{}
	log.SetLogWriter(io.Discard)

	ba, err := NewBeRealAdapter(os.DirFS(dataDir), log)
	assert.NoError(t, err)

	// Browse and check that caption is added to assets and marked for upload
	ch := ba.Browse(context.Background())
	assetCount := 0
	for g := range ch {
		for _, a := range g.Assets {
			assetCount++
			// Verify description is set from the caption
			assert.Equal(t, "Test caption", a.Description, "description should match the caption from memories.json")

			// Verify description is also in FromApplication (marked for server upload)
			assert.NotNil(t, a.FromApplication, "FromApplication should be set")
			assert.Equal(t, a.Description, a.FromApplication.Description, "description should be in FromApplication for upload to server")
		}
	}
	assert.Equal(t, 2, assetCount, "should have 2 assets with descriptions")
}

func TestAssetsAreAddedToBeRealAlbum(t *testing.T) {
	dataDir := dataDirFromThisFile()

	log := &app.Log{}
	log.SetLogWriter(io.Discard)

	ba, err := NewBeRealAdapter(os.DirFS(dataDir), log)
	assert.NoError(t, err)

	// Every asset belongs to the single "BeReal" album, no per-date album
	ch := ba.Browse(context.Background())
	assetCount := 0
	for g := range ch {
		for _, a := range g.Assets {
			assetCount++
			if assert.Len(t, a.Albums, 1, "asset %s should belong to exactly one album", a.OriginalFileName) {
				assert.Equal(t, "BeReal", a.Albums[0].Title)
			}
		}
	}
	assert.Equal(t, 2, assetCount, "should have 2 assets in the BeReal album")
}

// media describes a file of a synthetic BeReal export.
type media struct {
	name      string
	mediaType string // "image" or "video"
	audio     bool   // for videos, the file holds an audio track
}

// writeExport writes a one memory BeReal export in a temporary folder and returns an adapter for it.
// The files are stored in Photos/post, while memories.json uses the Photos/<userId>/post form.
func writeExport(t *testing.T, front, back media, bts *media) *BeRealAdapter {
	t.Helper()
	dir := t.TempDir()
	postDir := filepath.Join(dir, "Photos", "post")
	assert.NoError(t, os.MkdirAll(postDir, 0o755))

	info := func(m media) map[string]any {
		content := []byte("fake image " + m.name)
		mime := "image/webp"
		if m.mediaType == "video" {
			mime = "video/mp4"
			moov := box("moov", track("vide"))
			if m.audio {
				moov = box("moov", track("vide"), track("soun"))
			}
			content = bytes.Join([][]byte{box("ftyp", []byte("isom"), make([]byte, 4)), box("mdat", []byte(m.name)), moov}, nil)
		}
		assert.NoError(t, os.WriteFile(filepath.Join(postDir, m.name), content, 0o644))
		return map[string]any{"path": "/Photos/user/post/" + m.name, "mediaType": m.mediaType, "mimeType": mime, "width": 1080, "height": 1440}
	}

	memory := map[string]any{
		"frontImage":   info(front),
		"backImage":    info(back),
		"takenTime":    "2025-12-12T13:06:43.585Z",
		"date":         "2025-12-12T00:00:00.000Z",
		"berealMoment": "2025-12-12T13:06:05.259Z",
	}
	if bts != nil {
		memory["btsMedia"] = info(*bts)
	}
	data, err := json.Marshal([]any{memory})
	assert.NoError(t, err)
	assert.NoError(t, os.WriteFile(filepath.Join(dir, "memories.json"), data, 0o644))

	log := &app.Log{}
	log.SetLogWriter(io.Discard)
	ba, err := NewBeRealAdapter(os.DirFS(dir), log)
	assert.NoError(t, err)
	return ba
}

// browseOne returns the only group of the adapter
func browseOne(t *testing.T, ba *BeRealAdapter) *assets.Group {
	t.Helper()
	var groups []*assets.Group
	for g := range ba.Browse(context.Background()) {
		groups = append(groups, g)
	}
	if !assert.Len(t, groups, 1) {
		t.FailNow()
	}
	return groups[0]
}

func TestBehindTheScenesVideoIsStacked(t *testing.T) {
	ba := writeExport(t,
		media{name: "front.webp", mediaType: "image"},
		media{name: "back.webp", mediaType: "image"},
		&media{name: "bts.mp4", mediaType: "video", audio: true},
	)

	count, err := ba.CountAssets()
	assert.NoError(t, err)
	assert.Equal(t, 3, count)

	g := browseOne(t, ba)
	if !assert.Len(t, g.Assets, 3) {
		return
	}
	assert.Equal(t, assets.GroupByDualCamera, g.Grouping)
	assert.Equal(t, 0, g.CoverIndex, "the back camera image is the cover")

	tags := map[string]string{}
	for _, a := range g.Assets {
		if assert.Len(t, a.Tags, 1) {
			tags[a.OriginalFileName] = a.Tags[0].Value
		}
		if assert.Len(t, a.Albums, 1) {
			assert.Equal(t, "BeReal", a.Albums[0].Title)
		}
	}
	assert.Equal(t, map[string]string{
		"back.webp":  "BeReal_Main",
		"front.webp": "BeReal_Selfie",
		"bts.mp4":    "BeReal_BTS",
	}, tags)
}

func TestVideoCover(t *testing.T) {
	video := func(name string, audio bool) media { return media{name: name, mediaType: "video", audio: audio} }
	tests := []struct {
		name        string
		front, back media
		wantCover   string
	}{
		{"back video is silent, front video has sound", video("front.mp4", true), video("back.mp4", false), "front.mp4"},
		{"both videos have sound", video("front.mp4", true), video("back.mp4", true), "back.mp4"},
		{"back video has sound", video("front.mp4", false), video("back.mp4", true), "back.mp4"},
		{"both videos are silent", video("front.mp4", false), video("back.mp4", false), "back.mp4"},
		{"images", media{name: "front.webp", mediaType: "image"}, media{name: "back.webp", mediaType: "image"}, "back.webp"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := browseOne(t, writeExport(t, tt.front, tt.back, nil))
			if assert.Len(t, g.Assets, 2) {
				assert.Equal(t, tt.wantCover, g.Assets[g.CoverIndex].OriginalFileName)
			}
		})
	}
}
