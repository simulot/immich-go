package fileprocessor

import (
	"context"
	"io/fs"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/simulot/immich-go/internal/assettracker"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/fshelper"
)

// mockFS is a simple mock filesystem for testing
type mockFS struct {
	name string
}

func (m *mockFS) Open(name string) (fs.File, error) {
	return nil, fs.ErrNotExist
}

func (m *mockFS) Name() string {
	return m.name
}

func newTestFile(path string) fshelper.FSAndName {
	return fshelper.FSName(&mockFS{name: "test.zip"}, path)
}

func TestNew(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	tracker := assettracker.New()
	recorder := fileevent.NewRecorder(logger)

	fp := New(tracker, recorder)

	if fp == nil {
		t.Fatal("New() returned nil")
	}
	if fp.Tracker() != tracker {
		t.Error("Tracker() doesn't return the same instance")
	}
	if fp.Logger() != recorder {
		t.Error("Logger() doesn't return the same instance")
	}
}

func TestRecordAssetDiscovered(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	tracker := assettracker.New()
	recorder := fileevent.NewRecorder(logger)
	fp := New(tracker, recorder)

	ctx := context.Background()
	file := newTestFile("/test/image.jpg")

	// Record asset discovery
	fp.RecordAssetDiscovered(ctx, file, 1024, fileevent.DiscoveredImage)

	// Check tracker
	counters := fp.GetAssetCounters()
	if counters.Pending != 1 {
		t.Errorf("Expected 1 pending asset, got %d", counters.Pending)
	}
	if counters.AssetSize != 1024 {
		t.Errorf("Expected 1024 bytes, got %d", counters.AssetSize)
	}

	// Check logger
	eventCounts := fp.GetEventCounts()
	if eventCounts[fileevent.DiscoveredImage] != 1 {
		t.Errorf("Expected 1 DiscoveredImage event, got %d", eventCounts[fileevent.DiscoveredImage])
	}
	eventSizes := fp.GetEventSizes()
	if eventSizes[fileevent.DiscoveredImage] != 1024 {
		t.Errorf("Expected 1024 bytes for DiscoveredImage, got %d", eventSizes[fileevent.DiscoveredImage])
	}
}

func TestRecordAssetDiscardedImmediately(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	tracker := assettracker.New()
	recorder := fileevent.NewRecorder(logger)
	fp := New(tracker, recorder)

	ctx := context.Background()
	file := newTestFile("/test/banned.jpg")

	// Record immediately discarded asset
	fp.RecordAssetDiscardedImmediately(ctx, file, 512, fileevent.DiscardedBanned, "banned filename")

	// Check tracker - should be in discarded state
	counters := fp.GetAssetCounters()
	if counters.Discarded != 1 {
		t.Errorf("Expected 1 discarded asset, got %d", counters.Discarded)
	}
	if counters.Pending != 0 {
		t.Errorf("Expected 0 pending assets, got %d", counters.Pending)
	}
	if counters.DiscardedSize != 512 {
		t.Errorf("Expected 512 bytes discarded, got %d", counters.DiscardedSize)
	}

	// Check logger
	eventCounts := fp.GetEventCounts()
	if eventCounts[fileevent.DiscardedBanned] != 1 {
		t.Errorf("Expected 1 DiscardedBanned event, got %d", eventCounts[fileevent.DiscardedBanned])
	}
}

func TestRecordAssetProcessed(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	tracker := assettracker.New()
	recorder := fileevent.NewRecorder(logger)
	fp := New(tracker, recorder)

	ctx := context.Background()
	file := newTestFile("/test/image.jpg")

	// First discover the asset
	fp.RecordAssetDiscovered(ctx, file, 1024, fileevent.DiscoveredImage)

	// Then mark as processed
	fp.RecordAssetProcessed(ctx, file, fileevent.UploadedSuccess)

	// Check tracker
	counters := fp.GetAssetCounters()
	if counters.Processed != 1 {
		t.Errorf("Expected 1 processed asset, got %d", counters.Processed)
	}
	if counters.Pending != 0 {
		t.Errorf("Expected 0 pending assets, got %d", counters.Pending)
	}
	if counters.ProcessedSize != 1024 {
		t.Errorf("Expected 1024 bytes processed, got %d", counters.ProcessedSize)
	}

	// Check logger
	eventCounts := fp.GetEventCounts()
	if eventCounts[fileevent.UploadedSuccess] != 1 {
		t.Errorf("Expected 1 UploadedSuccess event, got %d", eventCounts[fileevent.UploadedSuccess])
	}
}

func TestRecordAssetDiscarded(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	tracker := assettracker.New()
	recorder := fileevent.NewRecorder(logger)
	fp := New(tracker, recorder)

	ctx := context.Background()
	file := newTestFile("/test/duplicate.jpg")

	// First discover the asset
	fp.RecordAssetDiscovered(ctx, file, 2048, fileevent.DiscoveredImage)

	// Then mark as discarded during processing
	fp.RecordAssetDiscarded(ctx, file, fileevent.DiscardedLocalDuplicate, "duplicate in input")

	// Check tracker
	counters := fp.GetAssetCounters()
	if counters.Discarded != 1 {
		t.Errorf("Expected 1 discarded asset, got %d", counters.Discarded)
	}
	if counters.Pending != 0 {
		t.Errorf("Expected 0 pending assets, got %d", counters.Pending)
	}
	if counters.DiscardedSize != 2048 {
		t.Errorf("Expected 2048 bytes discarded, got %d", counters.DiscardedSize)
	}

	// Check logger
	eventCounts := fp.GetEventCounts()
	if eventCounts[fileevent.DiscardedLocalDuplicate] != 1 {
		t.Errorf("Expected 1 DiscardedLocalDuplicate event, got %d", eventCounts[fileevent.DiscardedLocalDuplicate])
	}
}

func TestRecordAssetError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	tracker := assettracker.New()
	recorder := fileevent.NewRecorder(logger)
	fp := New(tracker, recorder)

	ctx := context.Background()
	file := newTestFile("/test/failed.jpg")

	// First discover the asset
	fp.RecordAssetDiscovered(ctx, file, 512, fileevent.DiscoveredImage)

	// Then mark as errored
	testErr := fs.ErrPermission
	fp.RecordAssetError(ctx, file, fileevent.ErrorUploadFailed, testErr)

	// Check tracker
	counters := fp.GetAssetCounters()
	if counters.Errors != 1 {
		t.Errorf("Expected 1 errored asset, got %d", counters.Errors)
	}
	if counters.Pending != 0 {
		t.Errorf("Expected 0 pending assets, got %d", counters.Pending)
	}
	if counters.ErrorSize != 512 {
		t.Errorf("Expected 512 bytes errored, got %d", counters.ErrorSize)
	}

	// Check logger
	eventCounts := fp.GetEventCounts()
	if eventCounts[fileevent.ErrorUploadFailed] != 1 {
		t.Errorf("Expected 1 ErrorUploadFailed event, got %d", eventCounts[fileevent.ErrorUploadFailed])
	}
}

func TestRecordNonAsset(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	tracker := assettracker.New()
	recorder := fileevent.NewRecorder(logger)
	fp := New(tracker, recorder)

	ctx := context.Background()
	file := newTestFile("/test/metadata.json")

	// Record non-asset file
	fp.RecordNonAsset(ctx, file, 128, fileevent.DiscoveredSidecar)

	// Check tracker - should have nothing tracked
	counters := fp.GetAssetCounters()
	if counters.Total() != 0 {
		t.Errorf("Expected 0 tracked assets, got %d", counters.Total())
	}

	// Check logger - should have the event
	eventCounts := fp.GetEventCounts()
	if eventCounts[fileevent.DiscoveredSidecar] != 1 {
		t.Errorf("Expected 1 DiscoveredSidecar event, got %d", eventCounts[fileevent.DiscoveredSidecar])
	}
	eventSizes := fp.GetEventSizes()
	if eventSizes[fileevent.DiscoveredSidecar] != 128 {
		t.Errorf("Expected 128 bytes for sidecar, got %d", eventSizes[fileevent.DiscoveredSidecar])
	}
}

func TestFinalize(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	tracker := assettracker.New()
	recorder := fileevent.NewRecorder(logger)
	fp := New(tracker, recorder)

	ctx := context.Background()

	// Test successful finalization (no pending assets)
	err := fp.Finalize(ctx)
	if err != nil {
		t.Errorf("Finalize should succeed with no assets, got error: %v", err)
	}

	// Add and process an asset
	file1 := newTestFile("/test/image1.jpg")
	fp.RecordAssetDiscovered(ctx, file1, 1024, fileevent.DiscoveredImage)
	fp.RecordAssetProcessed(ctx, file1, fileevent.UploadedSuccess)

	// Should still succeed
	err = fp.Finalize(ctx)
	if err != nil {
		t.Errorf("Finalize should succeed with all assets processed, got error: %v", err)
	}

	// Add asset but don't process it
	file2 := newTestFile("/test/image2.jpg")
	fp.RecordAssetDiscovered(ctx, file2, 2048, fileevent.DiscoveredImage)

	// Should fail with pending asset
	err = fp.Finalize(ctx)
	if err == nil {
		t.Error("Finalize should fail with pending assets")
	}
	if !strings.Contains(err.Error(), "never reached final state") {
		t.Errorf("Error should mention assets never reached final state, got: %v", err)
	}

	// Check that ErrorIncomplete was logged
	eventCounts := fp.GetEventCounts()
	if eventCounts[fileevent.ErrorIncomplete] != 1 {
		t.Errorf("Expected 1 ErrorIncomplete event, got %d", eventCounts[fileevent.ErrorIncomplete])
	}
}

func TestIsComplete(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	tracker := assettracker.New()
	recorder := fileevent.NewRecorder(logger)
	fp := New(tracker, recorder)

	ctx := context.Background()

	// Initially complete (no assets)
	if !fp.IsComplete() {
		t.Error("Should be complete with no assets")
	}

	// Add asset - not complete
	file := newTestFile("/test/image.jpg")
	fp.RecordAssetDiscovered(ctx, file, 1024, fileevent.DiscoveredImage)
	if fp.IsComplete() {
		t.Error("Should not be complete with pending asset")
	}

	// Process asset - complete again
	fp.RecordAssetProcessed(ctx, file, fileevent.UploadedSuccess)
	if !fp.IsComplete() {
		t.Error("Should be complete when all assets processed")
	}
}

func TestGetPendingAssets(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	tracker := assettracker.New()
	recorder := fileevent.NewRecorder(logger)
	fp := New(tracker, recorder)

	ctx := context.Background()

	// No pending assets initially
	pending := fp.GetPendingAssets()
	if len(pending) != 0 {
		t.Errorf("Expected 0 pending assets, got %d", len(pending))
	}

	// Add some assets
	file1 := newTestFile("/test/image1.jpg")
	file2 := newTestFile("/test/image2.jpg")
	file3 := newTestFile("/test/image3.jpg")

	fp.RecordAssetDiscovered(ctx, file1, 1024, fileevent.DiscoveredImage)
	fp.RecordAssetDiscovered(ctx, file2, 2048, fileevent.DiscoveredImage)
	fp.RecordAssetDiscovered(ctx, file3, 512, fileevent.DiscoveredImage)

	// All should be pending
	pending = fp.GetPendingAssets()
	if len(pending) != 3 {
		t.Errorf("Expected 3 pending assets, got %d", len(pending))
	}

	// Process one
	fp.RecordAssetProcessed(ctx, file1, fileevent.UploadedSuccess)

	pending = fp.GetPendingAssets()
	if len(pending) != 2 {
		t.Errorf("Expected 2 pending assets, got %d", len(pending))
	}

	// Discard another
	fp.RecordAssetDiscarded(ctx, file2, fileevent.DiscardedLocalDuplicate, "duplicate")

	pending = fp.GetPendingAssets()
	if len(pending) != 1 {
		t.Errorf("Expected 1 pending asset, got %d", len(pending))
	}
}

func TestGenerateReport(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	tracker := assettracker.New()
	recorder := fileevent.NewRecorder(logger)
	fp := New(tracker, recorder)

	ctx := context.Background()

	// Add various activities
	file1 := newTestFile("/test/image1.jpg")
	file2 := newTestFile("/test/image2.jpg")
	sidecar := newTestFile("/test/metadata.json")

	fp.RecordAssetDiscovered(ctx, file1, 1024, fileevent.DiscoveredImage)
	fp.RecordAssetProcessed(ctx, file1, fileevent.UploadedSuccess)
	fp.RecordAssetDiscovered(ctx, file2, 2048, fileevent.DiscoveredVideo)
	fp.RecordNonAsset(ctx, sidecar, 128, fileevent.DiscoveredSidecar)

	// Generate report
	report := fp.GenerateReport()

	// Should contain both asset and event reports
	if !strings.Contains(report, "Asset Tracking Report") {
		t.Error("Report should contain Asset Tracking Report")
	}
	if !strings.Contains(report, "Event Report") {
		t.Error("Report should contain Event Report")
	}
	if !strings.Contains(report, "discovered image") {
		t.Error("Report should mention discovered image")
	}
}

func TestSummary(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	tracker := assettracker.New()
	recorder := fileevent.NewRecorder(logger)
	fp := New(tracker, recorder)

	ctx := context.Background()

	// Add various assets
	file1 := newTestFile("/test/processed.jpg")
	file2 := newTestFile("/test/discarded.jpg")
	file3 := newTestFile("/test/error.jpg")
	file4 := newTestFile("/test/pending.jpg")

	fp.RecordAssetDiscovered(ctx, file1, 1024, fileevent.DiscoveredImage)
	fp.RecordAssetProcessed(ctx, file1, fileevent.UploadedSuccess)

	fp.RecordAssetDiscardedImmediately(ctx, file2, 512, fileevent.DiscardedBanned, "banned")

	fp.RecordAssetDiscovered(ctx, file3, 2048, fileevent.DiscoveredImage)
	fp.RecordAssetError(ctx, file3, fileevent.ErrorUploadFailed, fs.ErrPermission)

	fp.RecordAssetDiscovered(ctx, file4, 256, fileevent.DiscoveredImage)

	// Get summary
	summary := fp.Summary()

	if !strings.Contains(summary, "1 processed") {
		t.Errorf("Summary should mention 1 processed, got: %s", summary)
	}
	if !strings.Contains(summary, "1 discarded") {
		t.Errorf("Summary should mention 1 discarded, got: %s", summary)
	}
	if !strings.Contains(summary, "1 errors") {
		t.Errorf("Summary should mention 1 errors, got: %s", summary)
	}
	if !strings.Contains(summary, "1 pending") {
		t.Errorf("Summary should mention 1 pending, got: %s", summary)
	}
}

func TestCompleteWorkflow(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	tracker := assettracker.New()
	recorder := fileevent.NewRecorder(logger)
	fp := New(tracker, recorder)

	ctx := context.Background()

	// Simulate a complete workflow
	// 1. Discover assets
	image1 := newTestFile("/photos/vacation1.jpg")
	image2 := newTestFile("/photos/vacation2.jpg")
	video1 := newTestFile("/photos/video.mp4")
	bannedImage := newTestFile("/photos/.DS_Store.jpg") // banned but is image type
	sidecar := newTestFile("/photos/metadata.json")
	banned := newTestFile("/photos/.DS_Store")

	fp.RecordAssetDiscovered(ctx, image1, 1024000, fileevent.DiscoveredImage)
	fp.RecordAssetDiscovered(ctx, image2, 2048000, fileevent.DiscoveredImage)
	fp.RecordAssetDiscovered(ctx, video1, 5120000, fileevent.DiscoveredVideo)
	fp.RecordAssetDiscardedImmediately(ctx, bannedImage, 100, fileevent.DiscardedBanned, "banned filename")
	fp.RecordNonAsset(ctx, sidecar, 512, fileevent.DiscoveredSidecar)
	fp.RecordNonAsset(ctx, banned, 50, fileevent.DiscoveredBanned, "reason", "banned filename")

	// 2. Process assets
	fp.RecordAssetProcessed(ctx, image1, fileevent.UploadedSuccess)
	fp.RecordAssetDiscarded(ctx, image2, fileevent.UploadedServerDuplicate, "server has duplicate")
	fp.RecordAssetProcessed(ctx, video1, fileevent.UploadedSuccess)

	// 3. Validate final state
	if !fp.IsComplete() {
		t.Error("All assets should be in final state")
	}

	err := fp.Finalize(ctx)
	if err != nil {
		t.Errorf("Finalize should succeed, got error: %v", err)
	}

	// 4. Check counters
	counters := fp.GetAssetCounters()
	if counters.Processed != 2 {
		t.Errorf("Expected 2 processed, got %d", counters.Processed)
	}
	if counters.Discarded != 2 { // image2 + bannedImage
		t.Errorf("Expected 2 discarded, got %d", counters.Discarded)
	}
	if counters.Total() != 4 { // 2 images + 1 video + 1 banned image
		t.Errorf("Expected 4 total assets, got %d", counters.Total())
	}

	// 5. Check event counts
	eventCounts := fp.GetEventCounts()
	if eventCounts[fileevent.DiscoveredImage] != 2 {
		t.Errorf("Expected 2 DiscoveredImage events, got %d", eventCounts[fileevent.DiscoveredImage])
	}
	if eventCounts[fileevent.DiscoveredVideo] != 1 {
		t.Errorf("Expected 1 DiscoveredVideo event, got %d", eventCounts[fileevent.DiscoveredVideo])
	}
	if eventCounts[fileevent.DiscoveredSidecar] != 1 {
		t.Errorf("Expected 1 DiscoveredSidecar event, got %d", eventCounts[fileevent.DiscoveredSidecar])
	}
	if eventCounts[fileevent.DiscoveredBanned] != 1 {
		t.Errorf("Expected 1 DiscoveredBanned event (non-asset), got %d", eventCounts[fileevent.DiscoveredBanned])
	}
	if eventCounts[fileevent.DiscardedBanned] != 1 {
		t.Errorf("Expected 1 DiscardedBanned event (asset), got %d", eventCounts[fileevent.DiscardedBanned])
	}
}
