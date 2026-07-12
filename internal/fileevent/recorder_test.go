package fileevent

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"
)

type namedLogValue string

func (n namedLogValue) Name() string         { return string(n) }
func (n namedLogValue) LogValue() slog.Value { return slog.StringValue(string(n)) }

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
	recorder.Record(ctx, ProcessedUploadSuccess, nil)
	recorder.Record(ctx, DiscardedServerDuplicate, nil)
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

func TestGenerateEventReportGroupsUnknownFilesByExtension(t *testing.T) {
	recorder := NewRecorder(nil)
	ctx := context.Background()

	recorder.RecordWithSize(ctx, DiscoveredUnknown, namedLogValue("takeout/clip.MP"), 2048)
	recorder.RecordWithSize(ctx, DiscoveredUnknown, namedLogValue("takeout/another.mp"), 1024)
	recorder.RecordWithSize(ctx, DiscoveredUnknown, namedLogValue("takeout/video.3g2"), 4096)
	recorder.RecordWithSize(ctx, DiscoveredUnknown, namedLogValue("takeout/MVIMG_123"), 512)
	recorder.RecordWithSize(ctx, DiscoveredUnknown, namedLogValue("takeout/archive.asf"), 3072)

	report := recorder.GenerateEventReport()
	for _, want := range []string{
		"discovered unknown file            :       5  (10.5 KB)",
		".3g2                             :       1  (4.0 KB)",
		".asf                             :       1  (3.0 KB)",
		".mp                              :       2  (3.0 KB)",
		"[no extension]                   :       1  (512 B)",
	} {
		if !strings.Contains(report, want) {
			t.Errorf("report does not contain %q:\n%s", want, report)
		}
	}

	last := -1
	for _, extension := range []string{".3g2", ".asf", ".mp", "[no extension]"} {
		index := strings.Index(report, fmt.Sprintf("    %-33s:", extension))
		if index <= last {
			t.Fatalf("extension %q is not in deterministic size/name order:\n%s", extension, report)
		}
		last = index
	}
}

func TestUnknownExtensionGroupingIgnoresUnnamedLogValues(t *testing.T) {
	recorder := NewRecorder(nil)
	recorder.RecordWithSize(context.Background(), DiscoveredUnknown, unnamedLogValue{}, 42)

	report := recorder.GenerateEventReport()
	if strings.Contains(report, "[no extension]") {
		t.Fatalf("unnamed log value should not be grouped as a filename:\n%s", report)
	}
}

type unnamedLogValue struct{}

func (unnamedLogValue) LogValue() slog.Value { return slog.StringValue("unknown") }

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
	recorder.Record(ctx, ProcessedUploadSuccess, nil)

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
	if eventCounts[ProcessedUploadSuccess] != 1 {
		t.Errorf("Expected 1 upload success, got %d", eventCounts[ProcessedUploadSuccess])
	}

	// Should not have entries for events that weren't recorded
	if _, exists := eventCounts[DiscoveredSidecar]; exists {
		t.Error("Should not have entry for DiscoveredSidecar")
	}
}
