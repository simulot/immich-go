package upload

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/assettracker"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/fileprocessor"
	"github.com/spf13/cobra"
)

// handleGroup must return the error of every failed asset in the group, not just the last
// asset's: uploadLoop feeds its result to ProcessError, which drives --on-errors accounting.
func TestHandleGroupKeepsEarlierAssetErrors(t *testing.T) {
	ctx := context.Background()
	a := app.New(ctx, &cobra.Command{})
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	a.SetFileProcessor(fileprocessor.New(assettracker.New(), fileevent.NewRecorder(logger)))

	uc := &UpCmd{app: a, assetIndex: newAssetIndex()}

	// Fails: no checksum and no file to compute it from, so handleAsset errors in ShouldUpload.
	failing := &assets.Asset{}
	// Succeeds: its checksum is indexed as already uploaded, so handleAsset takes the
	// AlreadyProcessed path and returns nil without needing an Immich client.
	succeeding := &assets.Asset{Checksum: "abcd"}
	uc.assetIndex.byChecksum.Store(succeeding.Checksum, succeeding)
	uc.assetIndex.uploadsChecksum.Add(succeeding.Checksum)

	// GroupByNone skips the stacking step, which would need an Immich client.
	g := &assets.Group{Assets: []*assets.Asset{failing, succeeding}, Grouping: assets.GroupByNone}

	if err := uc.handleGroup(ctx, g); err == nil {
		t.Error("handleGroup returned nil, dropping the first asset's error")
	}
}
