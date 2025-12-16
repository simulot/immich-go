package bereal

import (
	"context"
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
