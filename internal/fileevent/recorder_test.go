package fileevent

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"
)

func TestRecorderSizeTracking(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	recorder := NewRecorder(logger)

	ctx := context.Background()

	// Record events with sizes
	recorder.RecordWithSize(ctx, DiscoveredImage, nil, 1024, "test", "image1")
	recorder.RecordWithSize(ctx, DiscoveredImage, nil, 2048, "test", "image2")
	recorder.RecordWithSize(ctx, DiscoveredVideo, nil, 5120, "test", "video1")
	recorder.RecordWithSize(ctx, DiscoveredSidecar, nil, 512, "test", "sidecar1")

	// Check counts
	eventCounts := recorder.GetEventCounts()
	if eventCounts[DiscoveredImage] != 2 {
		t.Errorf("Expected 2 images, got %d", eventCounts[DiscoveredImage])
	}
	if eventCounts[DiscoveredVideo] != 1 {
		t.Errorf("Expected 1 video, got %d", eventCounts[DiscoveredVideo])
	}
	if eventCounts[DiscoveredSidecar] != 1 {
		t.Errorf("Expected 1 sidecar, got %d", eventCounts[DiscoveredSidecar])
	}

	// Check sizes
	eventSizes := recorder.GetEventSizes()
	if eventSizes[DiscoveredImage] != 3072 {
		t.Errorf("Expected 3072 bytes for images, got %d", eventSizes[DiscoveredImage])
	}
	if eventSizes[DiscoveredVideo] != 5120 {
		t.Errorf("Expected 5120 bytes for videos, got %d", eventSizes[DiscoveredVideo])
	}
	if eventSizes[DiscoveredSidecar] != 512 {
		t.Errorf("Expected 512 bytes for sidecars, got %d", eventSizes[DiscoveredSidecar])
	}
}

func TestRecordBackwardCompatibility(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	recorder := NewRecorder(logger)

	ctx := context.Background()

	// Old Record method should still work (size = 0)
	recorder.Record(ctx, DiscoveredImage, nil, "test", "image1")
	recorder.Record(ctx, DiscoveredImage, nil, "test", "image2")

	// Check counts
	eventCounts := recorder.GetEventCounts()
	if eventCounts[DiscoveredImage] != 2 {
		t.Errorf("Expected 2 images, got %d", eventCounts[DiscoveredImage])
	}

	// Sizes should be 0
	eventSizes := recorder.GetEventSizes()
	if eventSizes[DiscoveredImage] != 0 {
		t.Errorf("Expected 0 bytes for images (old API), got %d", eventSizes[DiscoveredImage])
	}
}

func TestGenerateEventReport(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	recorder := NewRecorder(logger)

	ctx := context.Background()

	// Record various events
	recorder.RecordWithSize(ctx, DiscoveredImage, nil, 1024000, "test", "image")
	recorder.RecordWithSize(ctx, DiscoveredVideo, nil, 5120000, "test", "video")
	recorder.RecordWithSize(ctx, DiscoveredSidecar, nil, 512, "test", "sidecar")
	recorder.RecordWithSize(ctx, DiscoveredBanned, nil, 100, "test", "banned")
	recorder.Record(ctx, UploadedSuccess, nil)
	recorder.Record(ctx, UploadedServerDuplicate, nil)
	recorder.Record(ctx, ErrorUploadFailed, nil)

	// Generate report
	report := recorder.GenerateEventReport()

	// Check that report contains expected sections
	if !strings.Contains(report, "Event Report:") {
		t.Error("Report should contain 'Event Report:' header")
	}
	if !strings.Contains(report, "Discovery (Assets):") {
		t.Error("Report should contain 'Discovery (Assets):' section")
	}
	if !strings.Contains(report, "Discovery (Non-Assets):") {
		t.Error("Report should contain 'Discovery (Non-Assets):' section")
	}
	if !strings.Contains(report, "Asset Lifecycle (PROCESSED):") {
		t.Error("Report should contain 'Asset Lifecycle (PROCESSED):' section")
	}
	if !strings.Contains(report, "Asset Lifecycle (DISCARDED):") {
		t.Error("Report should contain 'Asset Lifecycle (DISCARDED):' section")
	}
	if !strings.Contains(report, "Asset Lifecycle (ERROR):") {
		t.Error("Report should contain 'Asset Lifecycle (ERROR):' section")
	}

	// Check specific event mentions
	if !strings.Contains(report, "discovered image") {
		t.Error("Report should mention 'discovered image'")
	}
	if !strings.Contains(report, "discovered video") {
		t.Error("Report should mention 'discovered video'")
	}
	if !strings.Contains(report, "uploaded successfully") {
		t.Error("Report should mention 'uploaded successfully'")
	}
}

func TestEmptyRecorder(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	recorder := NewRecorder(logger)

	// Check empty recorder
	eventCounts := recorder.GetEventCounts()
	if len(eventCounts) != 0 {
		t.Errorf("Expected empty event counts, got %d entries", len(eventCounts))
	}

	eventSizes := recorder.GetEventSizes()
	if len(eventSizes) != 0 {
		t.Errorf("Expected empty event sizes, got %d entries", len(eventSizes))
	}

	report := recorder.GenerateEventReport()
	if !strings.Contains(report, "No events recorded") {
		t.Error("Empty recorder should report 'No events recorded'")
	}
}

func TestGetEventCountsMap(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	recorder := NewRecorder(logger)

	ctx := context.Background()

	// Record some events
	recorder.Record(ctx, DiscoveredImage, nil)
	recorder.Record(ctx, DiscoveredImage, nil)
	recorder.Record(ctx, DiscoveredImage, nil)
	recorder.Record(ctx, DiscoveredVideo, nil)
	recorder.Record(ctx, UploadedSuccess, nil)

	// Get map
	eventCounts := recorder.GetEventCounts()

	// Should only have entries for recorded events
	if len(eventCounts) != 3 {
		t.Errorf("Expected 3 event types, got %d", len(eventCounts))
	}

	if eventCounts[DiscoveredImage] != 3 {
		t.Errorf("Expected 3 images, got %d", eventCounts[DiscoveredImage])
	}
	if eventCounts[DiscoveredVideo] != 1 {
		t.Errorf("Expected 1 video, got %d", eventCounts[DiscoveredVideo])
	}
	if eventCounts[UploadedSuccess] != 1 {
		t.Errorf("Expected 1 upload success, got %d", eventCounts[UploadedSuccess])
	}

	// Should not have entries for events that weren't recorded
	if _, exists := eventCounts[DiscoveredSidecar]; exists {
		t.Error("Should not have entry for DiscoveredSidecar")
	}
}
