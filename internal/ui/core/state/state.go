package state

import "time"

// RunStage indicates the current phase of the CLI session.
type RunStage string

const (
	StageUnknown   RunStage = "unknown"
	StagePending   RunStage = "pending"
	StageRunning   RunStage = "running"
	StagePaused    RunStage = "paused"
	StageCompleted RunStage = "completed"
	StageFailed    RunStage = "failed"
)

// ThroughputSample captures the computed upload throughput at a point in time.
type ThroughputSample struct {
	Timestamp      time.Time
	BytesPerSecond float64
}

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
	Retries              int
	Workers              int
	InFlight             int
	UploadPaused         bool
	Stage                RunStage
	ETA                  time.Duration
	ThroughputSamples    []ThroughputSample
	LastUpdated          time.Time
	HasErrors            bool
	StartedAt            time.Time
}

// JobSummary describes a background job running on the Immich server.
type JobSummary struct {
	Name      string
	Kind      string
	Active    int
	Waiting   int
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

// ServerInventory captures Immich library statistics for the authenticated user.
type ServerInventory struct {
	Photos    int
	Videos    int
	Total     int
	UpdatedAt time.Time
	Latency   time.Duration
}

// NewRunStats returns a zeroed RunStats with the provided start time.
func NewRunStats(start time.Time) RunStats {
	return RunStats{StartedAt: start, Stage: StageRunning}
}

// NewJobSummary returns an empty JobSummary for the provided job name.
func NewJobSummary(name string) JobSummary {
	return JobSummary{Name: name, Kind: name}
}

// CloneRunStats deep copies slices inside RunStats to avoid sharing mutable backing arrays across goroutines.
func CloneRunStats(stats RunStats) RunStats {
	if len(stats.ThroughputSamples) > 0 {
		samples := make([]ThroughputSample, len(stats.ThroughputSamples))
		copy(samples, stats.ThroughputSamples)
		stats.ThroughputSamples = samples
	}
	return stats
}
