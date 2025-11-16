package assettracker

import (
	"time"

	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/fshelper"
)

// AssetState represents the lifecycle state of an asset
type AssetState int

const (
	StatePending   AssetState = iota // Asset found, entering pipeline
	StateProcessed                   // Asset successfully handled
	StateDiscarded                   // Asset rejected/skipped
	StateError                       // Asset failed to process
)

// String returns the string representation of the asset state
func (s AssetState) String() string {
	switch s {
	case StatePending:
		return "PENDING"
	case StateProcessed:
		return "PROCESSED"
	case StateDiscarded:
		return "DISCARDED"
	case StateError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// AssetRecord tracks an individual asset through its lifecycle
type AssetRecord struct {
	File         fshelper.FSAndName // File identity
	FileSize     int64              // File size in bytes
	State        AssetState         // Current state
	EventCode    fileevent.Code     // Most recent event code
	Reason       string             // Why discarded/errored
	EventHistory []EventRecord      // Complete timeline (only in debug mode)
	DiscoveredAt time.Time          // When asset was discovered
	FinalizedAt  time.Time          // When asset reached final state
}

// EventRecord represents a single event in an asset's history
type EventRecord struct {
	Code      fileevent.Code // Event code
	Timestamp time.Time      // When event occurred
	Message   string         // Event message
	Args      map[string]any // Additional arguments
}

// AssetCounters provides summary statistics for tracked assets
type AssetCounters struct {
	// Asset counts (images/videos tracked through lifecycle)
	Pending   int64 // Assets not yet finalized
	Processed int64 // Assets successfully handled
	Discarded int64 // Assets skipped (immediate or during processing)
	Errors    int64 // Assets that failed

	// Asset size tracking
	AssetSize     int64 // Total asset bytes (all states)
	ProcessedSize int64 // Processed asset bytes
	DiscardedSize int64 // Discarded asset bytes
	ErrorSize     int64 // Errored asset bytes
	PendingSize   int64 // Bytes pending
}

// Total returns the total number of assets tracked
func (ac *AssetCounters) Total() int64 {
	return ac.Pending + ac.Processed + ac.Discarded + ac.Errors
}

// IsComplete returns true if all assets have reached a final state
func (ac *AssetCounters) IsComplete() bool {
	return ac.Pending == 0
}
