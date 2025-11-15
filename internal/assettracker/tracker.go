package assettracker

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/fshelper"
)

// AssetTracker tracks the complete lifecycle of assets (images/videos) from
// discovery through final state (processed/discarded/error).
// Non-asset files are ignored by this tracker.
type AssetTracker struct {
	// Per-asset tracking (ONLY images/videos)
	assets map[string]*AssetRecord // key: file.FullName()
	mu     sync.RWMutex

	// Asset counters (derived from tracked assets)
	pending   int64 // Assets in PENDING state
	processed int64 // Assets in PROCESSED state
	discarded int64 // Assets in DISCARDED state (includes immediate discards)
	errors    int64 // Assets in ERROR state

	// Size tracking (assets only)
	assetSize     int64 // Total bytes of assets (all states)
	processedSize int64 // Bytes of processed assets
	discardedSize int64 // Bytes of discarded assets
	errorSize     int64 // Bytes of errored assets

	// Debug mode
	debugMode bool
	log       *slog.Logger
}

// New creates a new AssetTracker
func New() *AssetTracker {
	return &AssetTracker{
		assets:    make(map[string]*AssetRecord),
		debugMode: false,
	}
}

// NewWithLogger creates a new AssetTracker with debug logging
func NewWithLogger(log *slog.Logger, debugMode bool) *AssetTracker {
	return &AssetTracker{
		assets:    make(map[string]*AssetRecord),
		debugMode: debugMode,
		log:       log,
	}
}

// DiscoverAsset registers an asset (image/video) entering the pipeline
func (at *AssetTracker) DiscoverAsset(file fshelper.FSAndName, fileSize int64, eventCode fileevent.Code) {
	at.mu.Lock()
	defer at.mu.Unlock()

	key := file.FullName()
	if _, exists := at.assets[key]; exists {
		// Asset already tracked - this shouldn't happen
		if at.log != nil {
			at.log.Warn("Asset already tracked", "file", file, "code", eventCode)
		}
		return
	}

	record := &AssetRecord{
		File:         file,
		FileSize:     fileSize,
		State:        StatePending,
		EventCode:    eventCode,
		DiscoveredAt: time.Now(),
	}

	if at.debugMode {
		record.EventHistory = []EventRecord{
			{
				Code:      eventCode,
				Timestamp: record.DiscoveredAt,
				Message:   "Asset discovered",
			},
		}
	}

	at.assets[key] = record
	at.pending++
	at.assetSize += fileSize
}

// DiscoverAndDiscard registers an asset that is immediately discarded
// (e.g., banned filename that's still an image type)
func (at *AssetTracker) DiscoverAndDiscard(file fshelper.FSAndName, fileSize int64, eventCode fileevent.Code, reason string) {
	at.mu.Lock()
	defer at.mu.Unlock()

	key := file.FullName()
	if _, exists := at.assets[key]; exists {
		// Asset already tracked
		if at.log != nil {
			at.log.Warn("Asset already tracked", "file", file, "code", eventCode)
		}
		return
	}

	now := time.Now()
	record := &AssetRecord{
		File:         file,
		FileSize:     fileSize,
		State:        StateDiscarded,
		EventCode:    eventCode,
		Reason:       reason,
		DiscoveredAt: now,
		FinalizedAt:  now,
	}

	if at.debugMode {
		record.EventHistory = []EventRecord{
			{
				Code:      eventCode,
				Timestamp: now,
				Message:   "Asset discovered and immediately discarded",
				Args:      map[string]any{"reason": reason},
			},
		}
	}

	at.assets[key] = record
	at.discarded++
	at.assetSize += fileSize
	at.discardedSize += fileSize
}

// SetProcessed transitions an asset to the PROCESSED state
func (at *AssetTracker) SetProcessed(file fshelper.FSAndName, eventCode fileevent.Code) {
	at.mu.Lock()
	defer at.mu.Unlock()

	key := file.FullName()
	record, exists := at.assets[key]
	if !exists {
		if at.log != nil {
			at.log.Error("SetProcessed: asset not found", "file", key, "code", eventCode)
		}
		return
	}

	if record.State != StatePending {
		if at.log != nil {
			at.log.Error("SetProcessed: asset not in pending state", "file", key, "current_state", record.State, "code", eventCode)
		}
		return
	}

	record.State = StateProcessed
	record.EventCode = eventCode
	record.FinalizedAt = time.Now()

	if at.debugMode && record.EventHistory != nil {
		record.EventHistory = append(record.EventHistory, EventRecord{
			Code:      eventCode,
			Timestamp: record.FinalizedAt,
			Message:   "Asset processed",
		})
	}

	at.pending--
	at.processed++
	at.processedSize += record.FileSize
}

// SetDiscarded transitions an asset to the DISCARDED state
func (at *AssetTracker) SetDiscarded(file fshelper.FSAndName, eventCode fileevent.Code, reason string) {
	at.mu.Lock()
	defer at.mu.Unlock()

	key := file.FullName()
	record, exists := at.assets[key]
	if !exists {
		if at.log != nil {
			at.log.Error("SetDiscarded: asset not found", "file", key, "code", eventCode, "reason", reason)
		}
		return
	}

	if record.State != StatePending {
		if at.log != nil {
			at.log.Error("SetDiscarded: asset not in pending state", "file", key, "current_state", record.State, "code", eventCode, "reason", reason)
		}
		return
	}

	record.State = StateDiscarded
	record.EventCode = eventCode
	record.Reason = reason
	record.FinalizedAt = time.Now()

	if at.debugMode && record.EventHistory != nil {
		record.EventHistory = append(record.EventHistory, EventRecord{
			Code:      eventCode,
			Timestamp: record.FinalizedAt,
			Message:   "Asset discarded",
			Args:      map[string]any{"reason": reason},
		})
	}

	at.pending--
	at.discarded++
	at.discardedSize += record.FileSize
}

// SetError transitions an asset to the ERROR state
func (at *AssetTracker) SetError(file fshelper.FSAndName, eventCode fileevent.Code, err error) {
	at.mu.Lock()
	defer at.mu.Unlock()

	key := file.FullName()
	record, exists := at.assets[key]
	if !exists {
		if at.log != nil {
			at.log.Error("SetError: asset not found", "file", key, "code", eventCode, "error", err.Error())
		}
		return
	}

	if record.State != StatePending {
		if at.log != nil {
			at.log.Error("SetError: asset not in pending state", "file", key, "current_state", record.State, "code", eventCode, "error", err.Error())
		}
		return
	}

	record.State = StateError
	record.EventCode = eventCode
	record.Reason = err.Error()
	record.FinalizedAt = time.Now()

	if at.debugMode && record.EventHistory != nil {
		record.EventHistory = append(record.EventHistory, EventRecord{
			Code:      eventCode,
			Timestamp: record.FinalizedAt,
			Message:   "Asset error",
			Args:      map[string]any{"error": err.Error()},
		})
	}

	at.pending--
	at.errors++
	at.errorSize += record.FileSize
}

// GetCounters returns current asset counters
func (at *AssetTracker) GetCounters() AssetCounters {
	at.mu.RLock()
	defer at.mu.RUnlock()

	pendingSize := at.assetSize - at.processedSize - at.discardedSize - at.errorSize

	return AssetCounters{
		Pending:       at.pending,
		Processed:     at.processed,
		Discarded:     at.discarded,
		Errors:        at.errors,
		AssetSize:     at.assetSize,
		ProcessedSize: at.processedSize,
		DiscardedSize: at.discardedSize,
		ErrorSize:     at.errorSize,
		PendingSize:   pendingSize,
	}
}

// GetPending returns all assets that haven't reached a final state
func (at *AssetTracker) GetPending() []AssetRecord {
	at.mu.RLock()
	defer at.mu.RUnlock()

	var pending []AssetRecord
	for _, record := range at.assets {
		if record.State == StatePending {
			pending = append(pending, *record)
		}
	}
	return pending
}

// GetAllAssets returns all tracked assets
func (at *AssetTracker) GetAllAssets() []AssetRecord {
	at.mu.RLock()
	defer at.mu.RUnlock()

	assets := make([]AssetRecord, 0, len(at.assets))
	for _, record := range at.assets {
		assets = append(assets, *record)
	}
	return assets
}

// IsComplete returns true if all assets have reached a final state
func (at *AssetTracker) IsComplete() bool {
	at.mu.RLock()
	defer at.mu.RUnlock()
	return at.pending == 0
}

// Validate checks that all assets reached a final state
func (at *AssetTracker) Validate() error {
	pending := at.GetPending()
	if len(pending) > 0 {
		return fmt.Errorf("%d assets never reached final state", len(pending))
	}
	return nil
}

// GenerateReport creates a summary report of asset processing
func (at *AssetTracker) GenerateReport() string {
	counters := at.GetCounters()

	report := fmt.Sprintf("\nAsset Tracking Report:\n")
	report += fmt.Sprintf("=====================\n")
	report += fmt.Sprintf("Total Assets:    %7d  (%s)\n", counters.Total(), formatBytes(counters.AssetSize))
	report += fmt.Sprintf("  Processed:     %7d  (%s)\n", counters.Processed, formatBytes(counters.ProcessedSize))
	report += fmt.Sprintf("  Discarded:     %7d  (%s)\n", counters.Discarded, formatBytes(counters.DiscardedSize))
	report += fmt.Sprintf("  Errors:        %7d  (%s)\n", counters.Errors, formatBytes(counters.ErrorSize))
	report += fmt.Sprintf("  Pending:       %7d  (%s)\n", counters.Pending, formatBytes(counters.PendingSize))

	if !counters.IsComplete() {
		report += fmt.Sprintf("\n⚠️  WARNING: %d assets did not reach a final state!\n", counters.Pending)
	}

	return report
}

// GenerateDetailedReport creates a detailed CSV report of all assets (debug mode)
func (at *AssetTracker) GenerateDetailedReport(ctx context.Context) string {
	at.mu.RLock()
	defer at.mu.RUnlock()

	report := "FilePath,Size,State,EventCode,Reason,DiscoveredAt,FinalizedAt\n"

	for _, record := range at.assets {
		finalizedAt := ""
		if !record.FinalizedAt.IsZero() {
			finalizedAt = record.FinalizedAt.Format(time.RFC3339)
		}

		report += fmt.Sprintf("%s,%d,%s,%s,%q,%s,%s\n",
			record.File.FullName(),
			record.FileSize,
			record.State,
			record.EventCode,
			record.Reason,
			record.DiscoveredAt.Format(time.RFC3339),
			finalizedAt,
		)
	}

	return report
}

// formatBytes formats byte count as human-readable string
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
