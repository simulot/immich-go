package bereal

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/simulot/immich-go/app"
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
