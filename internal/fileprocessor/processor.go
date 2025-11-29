package fileprocessor

import (
	"context"
	"fmt"

	"github.com/simulot/immich-go/internal/assettracker"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/fshelper"
)

// FileProcessor coordinates AssetTracker and EventLogger to provide
// a unified interface for tracking file processing lifecycle.
type FileProcessor struct {
	tracker      *assettracker.AssetTracker
	logger       *fileevent.Recorder
	countersHook func(assettracker.AssetCounters)
	eventHook    EventHook
}

// New creates a new FileProcessor with the given tracker and logger
func New(tracker *assettracker.AssetTracker, logger *fileevent.Recorder) *FileProcessor {
	return &FileProcessor{
		tracker: tracker,
		logger:  logger,
	}
}

// SetCountersHook registers a callback invoked every time asset counters change.
func (fp *FileProcessor) SetCountersHook(hook func(assettracker.AssetCounters)) {
	fp.countersHook = hook
	if hook != nil {
		hook(fp.tracker.GetCounters())
	}
}

func (fp *FileProcessor) emitCounters() {
	if fp.countersHook != nil {
		fp.countersHook(fp.tracker.GetCounters())
	}
}

// EventHook observes low-level file events after they are recorded.
type EventHook func(ctx context.Context, code fileevent.Code, file fshelper.FSAndName, size int64, attrs map[string]string)

// SetEventHook registers a callback invoked after each record operation.
func (fp *FileProcessor) SetEventHook(hook EventHook) {
	fp.eventHook = hook
}

func (fp *FileProcessor) emitEvent(ctx context.Context, code fileevent.Code, file fshelper.FSAndName, size int64, attrs map[string]string) {
	if fp.eventHook == nil {
		return
	}
	var copyAttrs map[string]string
	if len(attrs) > 0 {
		copyAttrs = make(map[string]string, len(attrs))
		for k, v := range attrs {
			copyAttrs[k] = v
		}
	}
	fp.eventHook(ctx, code, file, size, copyAttrs)
}

func attrsFromPairs(args ...any) map[string]string {
	if len(args) == 0 {
		return nil
	}
	attrs := make(map[string]string)
	for i := 0; i < len(args); i += 2 {
		key, ok := args[i].(string)
		if !ok {
			continue
		}
		var value string
		if i+1 < len(args) {
			value = fmt.Sprint(args[i+1])
		}
		attrs[key] = value
	}
	return attrs
}

// Tracker returns the underlying AssetTracker
func (fp *FileProcessor) Tracker() *assettracker.AssetTracker {
	return fp.tracker
}

// Logger returns the underlying EventLogger
func (fp *FileProcessor) Logger() *fileevent.Recorder {
	return fp.logger
}

// RecordAssetDiscovered records an asset (image/video) entering the pipeline.
// The asset is tracked and the discovery event is logged.
func (fp *FileProcessor) RecordAssetDiscovered(ctx context.Context, file fshelper.FSAndName, size int64, code fileevent.Code) {
	fp.tracker.DiscoverAsset(file, size, code)
	fp.logger.RecordWithSize(ctx, code, file, size)
	fp.emitEvent(ctx, code, file, size, nil)
	fp.emitCounters()
}

// RecordAssetDiscardedImmediately records an asset that is immediately discarded
// upon discovery (e.g., banned filename that's still an image type).
// The asset is tracked in DISCARDED state and the event is logged.
func (fp *FileProcessor) RecordAssetDiscardedImmediately(ctx context.Context, file fshelper.FSAndName, size int64, code fileevent.Code, reason string) {
	fp.tracker.DiscoverAndDiscard(file, size, code, reason)
	fp.logger.RecordWithSize(ctx, code, file, size, "reason", reason)
	fp.emitEvent(ctx, code, file, size, map[string]string{"reason": reason})
	fp.emitCounters()
}

// RecordAssetProcessed transitions an asset to PROCESSED state.
// The state change is tracked and the event is logged.
func (fp *FileProcessor) RecordAssetProcessed(ctx context.Context, file fshelper.FSAndName, size int64, code fileevent.Code) {
	fp.tracker.SetProcessed(file, code)
	fp.logger.RecordWithSize(ctx, code, file, size)
	fp.emitEvent(ctx, code, file, size, nil)
	fp.emitCounters()
}

// RecordAssetDiscarded transitions an asset to DISCARDED state.
// The state change is tracked and the event is logged with the reason.
func (fp *FileProcessor) RecordAssetDiscarded(ctx context.Context, file fshelper.FSAndName, size int64, code fileevent.Code, reason string) {
	fp.tracker.SetDiscarded(file, code, reason)
	fp.logger.RecordWithSize(ctx, code, file, size, "reason", reason)
	fp.emitEvent(ctx, code, file, size, map[string]string{"reason": reason})
	fp.emitCounters()
}

// RecordAssetError transitions an asset to ERROR state.
// The state change is tracked and the error event is logged.
func (fp *FileProcessor) RecordAssetError(ctx context.Context, file fshelper.FSAndName, size int64, code fileevent.Code, err error) {
	fp.tracker.SetError(file, code, err)
	fp.logger.RecordWithSize(ctx, code, file, size, "error", err.Error())
	fp.emitEvent(ctx, code, file, size, map[string]string{"error": err.Error()})
	fp.emitCounters()
}

// RecordNonAsset records a non-asset file (sidecar, metadata, etc.).
// Only logged, not tracked in AssetTracker.
func (fp *FileProcessor) RecordNonAsset(ctx context.Context, file fshelper.FSAndName, size int64, code fileevent.Code, args ...any) {
	fp.logger.RecordWithSize(ctx, code, file, size, args...)
	fp.emitEvent(ctx, code, file, size, attrsFromPairs(args...))
}

// Finalize validates that all assets have reached a final state.
// Returns an error if any assets are still pending.
func (fp *FileProcessor) Finalize(ctx context.Context) error {
	// Validate all assets reached final state
	if err := fp.tracker.Validate(); err != nil {
		// Log any incomplete assets
		pending := fp.tracker.GetPending()
		for _, asset := range pending {
			fp.logger.Record(ctx, fileevent.ErrorIncomplete, asset.File,
				"error", "asset never reached final state",
				"discovered_at", asset.DiscoveredAt,
			)
		}
		return err
	}
	return nil
}

// GenerateReport generates a comprehensive report combining asset tracking
// and event logging information.
func (fp *FileProcessor) GenerateReport() string {
	report := ""

	// Asset tracking report
	assetReport := fp.tracker.GenerateReport()
	if assetReport != "" {
		report += assetReport
	}

	// Event logging report
	eventReport := fp.logger.GenerateEventReport()
	if eventReport != "" {
		report += "\n" + eventReport
	}

	return report
}

// GetAssetCounters returns current asset counters from the tracker
func (fp *FileProcessor) GetAssetCounters() assettracker.AssetCounters {
	return fp.tracker.GetCounters()
}

// GetEventCounts returns event counts from the logger
func (fp *FileProcessor) GetEventCounts() map[fileevent.Code]int64 {
	return fp.logger.GetEventCounts()
}

// GetEventSizes returns event sizes from the logger
func (fp *FileProcessor) GetEventSizes() map[fileevent.Code]int64 {
	return fp.logger.GetEventSizes()
}

// IsComplete returns true if all assets have reached a final state
func (fp *FileProcessor) IsComplete() bool {
	return fp.tracker.IsComplete()
}

// GetPendingAssets returns all assets that haven't reached a final state
func (fp *FileProcessor) GetPendingAssets() []assettracker.AssetRecord {
	return fp.tracker.GetPending()
}

// GenerateDetailedReport creates a detailed CSV report of all assets (debug mode)
func (fp *FileProcessor) GenerateDetailedReport(ctx context.Context) string {
	return fp.tracker.GenerateDetailedReport(ctx)
}

// Summary provides a quick overview of processing status
func (fp *FileProcessor) Summary() string {
	counters := fp.tracker.GetCounters()
	return fmt.Sprintf("Assets: %d processed, %d discarded, %d errors, %d pending",
		counters.Processed, counters.Discarded, counters.Errors, counters.Pending)
}
