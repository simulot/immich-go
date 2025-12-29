package assettracker

import (
	"errors"
	"testing"
	"time"

	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/fshelper"
)

// TestAssetTrackerWithBus verifies that AssetTracker publishes state transition events
func TestAssetTrackerWithBus(t *testing.T) {
	bus := fileevent.NewBus()
	tracker := NewWithBus(nil, false, bus)

	// Subscribe to state transition events
	sub := bus.Subscribe(
		fileevent.AssetStateTransitionProcessed,
		fileevent.AssetStateTransitionDiscarded,
		fileevent.AssetStateTransitionError,
	)
	defer sub.Close()

	// Create test file
	file := fshelper.FSName(mockFS{}, "test.jpg")

	// Discover asset
	tracker.DiscoverAsset(file, 1024, fileevent.DiscoveredImage)

	// Test 1: SetProcessed should emit AssetStateTransitionProcessed
	tracker.SetProcessed(file, fileevent.ProcessedUploadSuccess)

	select {
	case event := <-sub.Receive():
		if event.Code != fileevent.AssetStateTransitionProcessed {
			t.Errorf("Expected AssetStateTransitionProcessed, got %v", event.Code)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected AssetStateTransitionProcessed event, but none received")
	}

	// Create another asset for discard test
	file2 := fshelper.FSName(mockFS{}, "test2.jpg")
	tracker.DiscoverAsset(file2, 2048, fileevent.DiscoveredImage)

	// Test 2: SetDiscarded should emit AssetStateTransitionDiscarded
	tracker.SetDiscarded(file2, fileevent.DiscardedServerDuplicate, "duplicate file")

	select {
	case event := <-sub.Receive():
		if event.Code != fileevent.AssetStateTransitionDiscarded {
			t.Errorf("Expected AssetStateTransitionDiscarded, got %v", event.Code)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected AssetStateTransitionDiscarded event, but none received")
	}

	// Create another asset for error test
	file3 := fshelper.FSName(mockFS{}, "test3.jpg")
	tracker.DiscoverAsset(file3, 3072, fileevent.DiscoveredImage)

	// Test 3: SetError should emit AssetStateTransitionError
	tracker.SetError(file3, fileevent.ErrorUploadFailed, errors.New("network error"))

	select {
	case event := <-sub.Receive():
		if event.Code != fileevent.AssetStateTransitionError {
			t.Errorf("Expected AssetStateTransitionError, got %v", event.Code)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected AssetStateTransitionError event, but none received")
	}
}

// TestAssetTrackerWithoutBus verifies that AssetTracker works correctly when no bus is provided
func TestAssetTrackerWithoutBus(t *testing.T) {
	// Create tracker without bus (nil)
	tracker := NewWithLogger(nil, false)

	file := fshelper.FSName(mockFS{}, "test.jpg")

	// Should not panic when bus is nil
	tracker.DiscoverAsset(file, 1024, fileevent.DiscoveredImage)
	tracker.SetProcessed(file, fileevent.ProcessedUploadSuccess)

	// Verify state changed correctly
	counters := tracker.GetCounters()
	if counters.Processed != 1 {
		t.Errorf("Expected 1 processed asset, got %d", counters.Processed)
	}
}

// TestAssetTrackerBusIntegration verifies complete workflow with event bus
func TestAssetTrackerBusIntegration(t *testing.T) {
	bus := fileevent.NewBus()
	tracker := NewWithBus(nil, false, bus)

	// Collect all state transition events
	events := make([]fileevent.Event, 0)
	sub := bus.Subscribe(
		fileevent.AssetStateTransitionProcessed,
		fileevent.AssetStateTransitionDiscarded,
		fileevent.AssetStateTransitionError,
	)
	defer sub.Close()

	// Drain events in background
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case event, ok := <-sub.Receive():
				if !ok {
					return
				}
				events = append(events, event)
			case <-time.After(200 * time.Millisecond):
				return
			}
		}
	}()

	// Process multiple assets with different outcomes
	file1 := fshelper.FSName(mockFS{}, "processed.jpg")
	tracker.DiscoverAsset(file1, 1024, fileevent.DiscoveredImage)
	tracker.SetProcessed(file1, fileevent.ProcessedUploadSuccess)

	file2 := fshelper.FSName(mockFS{}, "discarded.jpg")
	tracker.DiscoverAsset(file2, 2048, fileevent.DiscoveredImage)
	tracker.SetDiscarded(file2, fileevent.DiscardedServerDuplicate, "duplicate")

	file3 := fshelper.FSName(mockFS{}, "error.jpg")
	tracker.DiscoverAsset(file3, 3072, fileevent.DiscoveredImage)
	tracker.SetError(file3, fileevent.ErrorUploadFailed, errors.New("test error"))

	// Wait for event collection to complete
	<-done

	// Verify we received all 3 state transition events
	if len(events) != 3 {
		t.Errorf("Expected 3 state transition events, got %d", len(events))
	}

	// Verify event types
	expectedCodes := map[fileevent.Code]bool{
		fileevent.AssetStateTransitionProcessed: false,
		fileevent.AssetStateTransitionDiscarded: false,
		fileevent.AssetStateTransitionError:     false,
	}

	for _, event := range events {
		if _, exists := expectedCodes[event.Code]; exists {
			expectedCodes[event.Code] = true
		} else {
			t.Errorf("Unexpected event code: %v", event.Code)
		}
	}

	for code, received := range expectedCodes {
		if !received {
			t.Errorf("Did not receive expected event: %v", code)
		}
	}
}
