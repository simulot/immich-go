package state

import "time"

// AssetRef identifies an asset being processed.
type AssetRef struct {
	ID   string
	Path string
}

// AssetStage describes the lifecycle phase of an asset event.
type AssetStage string

const (
	AssetStageQueued   AssetStage = "queued"
	AssetStageUploaded AssetStage = "uploaded"
	AssetStageFailed   AssetStage = "failed"
)

// AssetEventCode is a renderer-agnostic identifier for lifecycle events.
type AssetEventCode int

// AssetEvent carries structured information about an asset lifecycle update.
type AssetEvent struct {
	Asset     AssetRef
	Stage     AssetStage
	Code      AssetEventCode
	CodeLabel string
	Bytes     int64
	Reason    string
	Details   map[string]string
}

// RunStats aggregates high-level counters for the current CLI session.
type RunStats struct {
	Queued               int
	Uploaded             int
	Failed               int
	BytesSent            int64
	Pending              int
	PendingBytes         int64
	Processed            int
	ProcessedBytes       int64
	Discarded            int
	DiscardedBytes       int64
	ErrorCount           int
	ErrorBytes           int64
	TotalDiscovered      int
	TotalDiscoveredBytes int64
	HasErrors            bool
	StartedAt            time.Time
}

// JobSummary describes a background job running on the Immich server.
type JobSummary struct {
	Name      string
	Pending   int
	Completed int
	Failed    int
	UpdatedAt time.Time
}

// LogEvent captures user-facing log data that may need highlighting in the UI.
type LogEvent struct {
	Level     string
	Message   string
	Timestamp time.Time
	Details   map[string]string
}

// NewRunStats returns a zeroed RunStats with the provided start time.
func NewRunStats(start time.Time) RunStats {
	return RunStats{StartedAt: start}
}

// NewJobSummary returns an empty JobSummary for the provided job name.
func NewJobSummary(name string) JobSummary {
	return JobSummary{Name: name}
}
