package assettracker

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/fshelper"
)

// mockFS implements a simple fs.FS for testing
type mockFS struct{}

func (m mockFS) Open(name string) (fs.File, error) {
	return nil, fs.ErrNotExist
}

func (m mockFS) Name() string {
	return "test.zip"
}

func TestNew(t *testing.T) {
	tracker := New()
	if tracker == nil {
		t.Fatal("New() returned nil")
	}
	if tracker.assets == nil {
		t.Error("assets map not initialized")
	}
	if tracker.IsComplete() != true {
		t.Error("new tracker should be complete (no assets)")
	}
}

func TestDiscoverAsset(t *testing.T) {
	tracker := New()
	file := fshelper.FSName(mockFS{}, "test.jpg")

	tracker.DiscoverAsset(file, 1024, fileevent.DiscoveredImage)

	counters := tracker.GetCounters()
	if counters.Pending != 1 {
		t.Errorf("expected 1 pending asset, got %d", counters.Pending)
	}
	if counters.AssetSize != 1024 {
		t.Errorf("expected asset size 1024, got %d", counters.AssetSize)
	}
	if counters.Total() != 1 {
		t.Errorf("expected total 1, got %d", counters.Total())
	}
	if tracker.IsComplete() {
		t.Error("tracker should not be complete with pending assets")
	}
}

func TestDiscoverAndDiscard(t *testing.T) {
	tracker := New()
	file := fshelper.FSName(mockFS{}, "banned.jpg")

	tracker.DiscoverAndDiscard(file, 2048, fileevent.DiscardedBanned, "banned filename")

	counters := tracker.GetCounters()
	if counters.Discarded != 1 {
		t.Errorf("expected 1 discarded asset, got %d", counters.Discarded)
	}
	if counters.DiscardedSize != 2048 {
		t.Errorf("expected discarded size 2048, got %d", counters.DiscardedSize)
	}
	if counters.Pending != 0 {
		t.Errorf("expected 0 pending assets, got %d", counters.Pending)
	}
	if !tracker.IsComplete() {
		t.Error("tracker should be complete (asset immediately discarded)")
	}
}

func TestSetProcessed(t *testing.T) {
	tracker := New()
	file := fshelper.FSName(mockFS{}, "photo.jpg")

	// Discover asset
	tracker.DiscoverAsset(file, 1024, fileevent.DiscoveredImage)

	// Process it
	tracker.SetProcessed(file, fileevent.UploadedSuccess)

	counters := tracker.GetCounters()
	if counters.Processed != 1 {
		t.Errorf("expected 1 processed asset, got %d", counters.Processed)
	}
	if counters.Pending != 0 {
		t.Errorf("expected 0 pending assets, got %d", counters.Pending)
	}
	if counters.ProcessedSize != 1024 {
		t.Errorf("expected processed size 1024, got %d", counters.ProcessedSize)
	}
	if !tracker.IsComplete() {
		t.Error("tracker should be complete")
	}
}

func TestSetDiscarded(t *testing.T) {
	tracker := New()
	file := fshelper.FSName(mockFS{}, "duplicate.jpg")

	// Discover asset
	tracker.DiscoverAsset(file, 512, fileevent.DiscoveredImage)

	// Discard it
	tracker.SetDiscarded(file, fileevent.DiscardedLocalDuplicate, "duplicate in input")

	counters := tracker.GetCounters()
	if counters.Discarded != 1 {
		t.Errorf("expected 1 discarded asset, got %d", counters.Discarded)
	}
	if counters.Pending != 0 {
		t.Errorf("expected 0 pending assets, got %d", counters.Pending)
	}
	if counters.DiscardedSize != 512 {
		t.Errorf("expected discarded size 512, got %d", counters.DiscardedSize)
	}
}

func TestSetError(t *testing.T) {
	tracker := New()
	file := fshelper.FSName(mockFS{}, "failed.jpg")

	// Discover asset
	tracker.DiscoverAsset(file, 2048, fileevent.DiscoveredImage)

	// Error it
	tracker.SetError(file, fileevent.ErrorUploadFailed, fs.ErrPermission)

	counters := tracker.GetCounters()
	if counters.Errors != 1 {
		t.Errorf("expected 1 error asset, got %d", counters.Errors)
	}
	if counters.Pending != 0 {
		t.Errorf("expected 0 pending assets, got %d", counters.Pending)
	}
	if counters.ErrorSize != 2048 {
		t.Errorf("expected error size 2048, got %d", counters.ErrorSize)
	}
}

func TestMultipleAssets(t *testing.T) {
	tracker := New()

	// Discover multiple assets
	files := []struct {
		name string
		size int64
	}{
		{"photo1.jpg", 1024},
		{"photo2.jpg", 2048},
		{"photo3.jpg", 4096},
		{"video1.mp4", 8192},
	}

	for _, f := range files {
		file := fshelper.FSName(mockFS{}, f.name)
		tracker.DiscoverAsset(file, f.size, fileevent.DiscoveredImage)
	}

	counters := tracker.GetCounters()
	if counters.Pending != 4 {
		t.Errorf("expected 4 pending assets, got %d", counters.Pending)
	}
	if counters.AssetSize != 15360 {
		t.Errorf("expected asset size 15360, got %d", counters.AssetSize)
	}

	// Process some
	tracker.SetProcessed(fshelper.FSName(mockFS{}, "photo1.jpg"), fileevent.UploadedSuccess)
	tracker.SetProcessed(fshelper.FSName(mockFS{}, "photo2.jpg"), fileevent.UploadedSuccess)

	// Discard some
	tracker.SetDiscarded(fshelper.FSName(mockFS{}, "photo3.jpg"), fileevent.DiscardedLocalDuplicate, "duplicate")

	// Error some
	tracker.SetError(fshelper.FSName(mockFS{}, "video1.mp4"), fileevent.ErrorUploadFailed, fs.ErrPermission)

	counters = tracker.GetCounters()
	if counters.Processed != 2 {
		t.Errorf("expected 2 processed assets, got %d", counters.Processed)
	}
	if counters.Discarded != 1 {
		t.Errorf("expected 1 discarded asset, got %d", counters.Discarded)
	}
	if counters.Errors != 1 {
		t.Errorf("expected 1 error asset, got %d", counters.Errors)
	}
	if counters.Pending != 0 {
		t.Errorf("expected 0 pending assets, got %d", counters.Pending)
	}
	if !tracker.IsComplete() {
		t.Error("tracker should be complete")
	}
}

func TestGetPending(t *testing.T) {
	tracker := New()

	// Add some assets
	tracker.DiscoverAsset(fshelper.FSName(mockFS{}, "pending1.jpg"), 1024, fileevent.DiscoveredImage)
	tracker.DiscoverAsset(fshelper.FSName(mockFS{}, "pending2.jpg"), 2048, fileevent.DiscoveredImage)
	tracker.DiscoverAsset(fshelper.FSName(mockFS{}, "processed.jpg"), 4096, fileevent.DiscoveredImage)

	// Process one
	tracker.SetProcessed(fshelper.FSName(mockFS{}, "processed.jpg"), fileevent.UploadedSuccess)

	pending := tracker.GetPending()
	if len(pending) != 2 {
		t.Errorf("expected 2 pending assets, got %d", len(pending))
	}
}

func TestValidate(t *testing.T) {
	tracker := New()

	// Initially valid (no assets)
	if err := tracker.Validate(); err != nil {
		t.Errorf("empty tracker should be valid: %v", err)
	}

	// Add asset
	tracker.DiscoverAsset(fshelper.FSName(mockFS{}, "test.jpg"), 1024, fileevent.DiscoveredImage)

	// Should be invalid (pending asset)
	if err := tracker.Validate(); err == nil {
		t.Error("tracker with pending assets should be invalid")
	}

	// Process asset
	tracker.SetProcessed(fshelper.FSName(mockFS{}, "test.jpg"), fileevent.UploadedSuccess)

	// Should be valid again
	if err := tracker.Validate(); err != nil {
		t.Errorf("tracker should be valid after processing: %v", err)
	}
}

func TestDebugMode(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	tracker := NewWithLogger(log, true)

	file := fshelper.FSName(mockFS{}, "test.jpg")

	// Discover asset
	tracker.DiscoverAsset(file, 1024, fileevent.DiscoveredImage)

	// Get the record
	assets := tracker.GetAllAssets()
	if len(assets) != 1 {
		t.Fatalf("expected 1 asset, got %d", len(assets))
	}

	// Check event history exists in debug mode
	if assets[0].EventHistory == nil {
		t.Error("event history should be populated in debug mode")
	}
	if len(assets[0].EventHistory) != 1 {
		t.Errorf("expected 1 event in history, got %d", len(assets[0].EventHistory))
	}

	// Process and check history grows
	tracker.SetProcessed(file, fileevent.UploadedSuccess)
	assets = tracker.GetAllAssets()
	if len(assets[0].EventHistory) != 2 {
		t.Errorf("expected 2 events in history, got %d", len(assets[0].EventHistory))
	}
}

func TestGenerateReport(t *testing.T) {
	tracker := New()

	// Add some assets
	tracker.DiscoverAsset(fshelper.FSName(mockFS{}, "photo1.jpg"), 1024, fileevent.DiscoveredImage)
	tracker.DiscoverAsset(fshelper.FSName(mockFS{}, "photo2.jpg"), 2048, fileevent.DiscoveredImage)

	tracker.SetProcessed(fshelper.FSName(mockFS{}, "photo1.jpg"), fileevent.UploadedSuccess)
	tracker.SetDiscarded(fshelper.FSName(mockFS{}, "photo2.jpg"), fileevent.DiscardedLocalDuplicate, "duplicate")

	report := tracker.GenerateReport()
	if report == "" {
		t.Error("report should not be empty")
	}
	// Report should contain key information
	if len(report) < 100 {
		t.Errorf("report seems too short: %d characters", len(report))
	}
}

func TestGenerateDetailedReport(t *testing.T) {
	tracker := New()

	file := fshelper.FSName(mockFS{}, "photo.jpg")
	tracker.DiscoverAsset(file, 1024, fileevent.DiscoveredImage)
	tracker.SetProcessed(file, fileevent.UploadedSuccess)

	report := tracker.GenerateDetailedReport(context.Background())
	if report == "" {
		t.Error("detailed report should not be empty")
	}
	// Should be CSV format
	if report[:8] != "FilePath" {
		t.Error("detailed report should start with CSV header")
	}
}

func TestStateTransitionErrors(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	tracker := NewWithLogger(log, false)
	file := fshelper.FSName(mockFS{}, "test.jpg")

	// Try to transition non-existent asset - should log error but not fail
	tracker.SetProcessed(file, fileevent.UploadedSuccess)

	// Discover asset
	tracker.DiscoverAsset(file, 1024, fileevent.DiscoveredImage)

	// Process it
	tracker.SetProcessed(file, fileevent.UploadedSuccess)

	// Try to transition already-processed asset - should log error but not fail
	tracker.SetDiscarded(file, fileevent.DiscardedLocalDuplicate, "duplicate")
}

func TestConcurrency(t *testing.T) {
	tracker := New()
	done := make(chan bool)

	// Concurrently add assets
	for i := 0; i < 10; i++ {
		go func(n int) {
			file := fshelper.FSName(mockFS{}, "photo"+string(rune(n))+".jpg")
			tracker.DiscoverAsset(file, 1024, fileevent.DiscoveredImage)
			time.Sleep(time.Millisecond)
			tracker.SetProcessed(file, fileevent.UploadedSuccess)
			done <- true
		}(i)
	}

	// Wait for all
	for i := 0; i < 10; i++ {
		<-done
	}

	counters := tracker.GetCounters()
	if counters.Processed != 10 {
		t.Errorf("expected 10 processed assets, got %d", counters.Processed)
	}
}

func TestStatusMethods(t *testing.T) {
	tracker := New()
	file1 := fshelper.FSName(mockFS{}, "test1.jpg")
	file2 := fshelper.FSName(mockFS{}, "test2.jpg")
	file3 := fshelper.FSName(mockFS{}, "test3.jpg")
	file4 := fshelper.FSName(mockFS{}, "test4.jpg")

	// Discover assets
	tracker.DiscoverAsset(file1, 1024, fileevent.DiscoveredImage) // pending: 1, size: 1024
	tracker.DiscoverAsset(file2, 2048, fileevent.DiscoveredImage) // pending: 2, size: 3072
	tracker.DiscoverAsset(file3, 512, fileevent.DiscoveredImage)  // pending: 3, size: 3584
	tracker.DiscoverAsset(file4, 256, fileevent.DiscoveredImage)  // pending: 4, size: 3840

	// Test initial pending state
	if count := tracker.GetPendingCount(); count != 4 {
		t.Errorf("expected 4 pending assets, got %d", count)
	}
	if size := tracker.GetPendingSize(); size != 3840 {
		t.Errorf("expected pending size 3840, got %d", size)
	}
	if count := tracker.GetProcessedCount(); count != 0 {
		t.Errorf("expected 0 processed assets, got %d", count)
	}
	if size := tracker.GetProcessedSize(); size != 0 {
		t.Errorf("expected processed size 0, got %d", size)
	}

	// Discover assets
	tracker.DiscoverAsset(file1, 1024, fileevent.DiscoveredImage)
	tracker.DiscoverAsset(file2, 2048, fileevent.DiscoveredImage)
	tracker.DiscoverAsset(file3, 512, fileevent.DiscoveredImage)
	tracker.DiscoverAsset(file4, 256, fileevent.DiscoveredImage)

	// Test initial state
	if count := tracker.GetPendingCount(); count != 4 {
		t.Errorf("expected 4 pending assets, got %d", count)
	}
	if size := tracker.GetPendingSize(); size != 3840 { // 1024 + 2048 + 512 + 256
		t.Errorf("expected pending size 3840, got %d", size)
	}
	if count := tracker.GetProcessedCount(); count != 0 {
		t.Errorf("expected 0 processed assets, got %d", count)
	}
	if size := tracker.GetProcessedSize(); size != 0 {
		t.Errorf("expected processed size 0, got %d", size)
	}

	// Process some assets
	tracker.SetProcessed(file1, fileevent.UploadedSuccess)
	tracker.SetProcessed(file2, fileevent.UploadedSuccess)

	// Test after processing
	if count := tracker.GetPendingCount(); count != 2 {
		t.Errorf("expected 2 pending assets, got %d", count)
	}
	if size := tracker.GetPendingSize(); size != 768 { // 512 + 256
		t.Errorf("expected pending size 768, got %d", size)
	}
	if count := tracker.GetProcessedCount(); count != 2 {
		t.Errorf("expected 2 processed assets, got %d", count)
	}
	if size := tracker.GetProcessedSize(); size != 3072 { // 1024 + 2048
		t.Errorf("expected processed size 3072, got %d", size)
	}

	// Discard an asset
	tracker.SetDiscarded(file3, fileevent.DiscardedLocalDuplicate, "duplicate")

	// Test after discarding
	if count := tracker.GetPendingCount(); count != 1 {
		t.Errorf("expected 1 pending asset, got %d", count)
	}
	if size := tracker.GetPendingSize(); size != 256 {
		t.Errorf("expected pending size 256, got %d", size)
	}
	if count := tracker.GetDiscardedCount(); count != 1 {
		t.Errorf("expected 1 discarded asset, got %d", count)
	}
	if size := tracker.GetDiscardedSize(); size != 512 {
		t.Errorf("expected discarded size 512, got %d", size)
	}

	// Error on an asset
	tracker.SetError(file4, fileevent.ErrorUploadFailed, fmt.Errorf("read error"))

	// Test after error
	if count := tracker.GetPendingCount(); count != 0 {
		t.Errorf("expected 0 pending assets, got %d", count)
	}
	if size := tracker.GetPendingSize(); size != 0 {
		t.Errorf("expected pending size 0, got %d", size)
	}
	if count := tracker.GetErrorCount(); count != 1 {
		t.Errorf("expected 1 error asset, got %d", count)
	}
	if size := tracker.GetErrorSize(); size != 256 {
		t.Errorf("expected error size 256, got %d", size)
	}
}
