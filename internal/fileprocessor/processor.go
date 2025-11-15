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
	tracker *assettracker.AssetTracker
	logger  *fileevent.Recorder
}

// New creates a new FileProcessor with the given tracker and logger
func New(tracker *assettracker.AssetTracker, logger *fileevent.Recorder) *FileProcessor {
	return &FileProcessor{
		tracker: tracker,
		logger:  logger,
	}
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
}

// RecordAssetDiscardedImmediately records an asset that is immediately discarded
// upon discovery (e.g., banned filename that's still an image type).
// The asset is tracked in DISCARDED state and the event is logged.
func (fp *FileProcessor) RecordAssetDiscardedImmediately(ctx context.Context, file fshelper.FSAndName, size int64, code fileevent.Code, reason string) {
	fp.tracker.DiscoverAndDiscard(file, size, code, reason)
	fp.logger.RecordWithSize(ctx, code, file, size, "reason", reason)
}

// RecordAssetProcessed transitions an asset to PROCESSED state.
// The state change is tracked and the event is logged.
func (fp *FileProcessor) RecordAssetProcessed(ctx context.Context, file fshelper.FSAndName, code fileevent.Code) error {
	if err := fp.tracker.SetProcessed(file, code); err != nil {
		return err
	}
	fp.logger.Record(ctx, code, file)
	return nil
}

// RecordAssetDiscarded transitions an asset to DISCARDED state.
// The state change is tracked and the event is logged with the reason.
func (fp *FileProcessor) RecordAssetDiscarded(ctx context.Context, file fshelper.FSAndName, code fileevent.Code, reason string) error {
	if err := fp.tracker.SetDiscarded(file, code, reason); err != nil {
		return err
	}
	fp.logger.Record(ctx, code, file, "reason", reason)
	return nil
}

// RecordAssetError transitions an asset to ERROR state.
// The state change is tracked and the error event is logged.
func (fp *FileProcessor) RecordAssetError(ctx context.Context, file fshelper.FSAndName, code fileevent.Code, err error) error {
	if setErr := fp.tracker.SetError(file, code, err); setErr != nil {
		return setErr
	}
	fp.logger.Record(ctx, code, file, "error", err.Error())
	return nil
}

// RecordNonAsset records a non-asset file (sidecar, metadata, etc.).
// Only logged, not tracked in AssetTracker.
func (fp *FileProcessor) RecordNonAsset(ctx context.Context, file fshelper.FSAndName, size int64, code fileevent.Code, args ...any) {
	fp.logger.RecordWithSize(ctx, code, file, size, args...)
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
