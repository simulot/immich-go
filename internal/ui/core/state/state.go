package state

import "time"

// AssetRef identifies an asset being processed.
type AssetRef struct {
	ID   string
	Path string
}

// RunStats aggregates high-level counters for the current CLI session.
type RunStats struct {
	Queued    int
	Uploaded  int
	Failed    int
	BytesSent int64
	StartedAt time.Time
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
