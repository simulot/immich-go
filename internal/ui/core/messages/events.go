package messages

// EventType identifies the semantic meaning of an event on the UI bus.
type EventType string

const (
	EventAssetQueued   EventType = "asset_queued"
	EventAssetUploaded EventType = "asset_uploaded"
	EventAssetFailed   EventType = "asset_failed"
	EventLogLine       EventType = "log_line"
	EventStatsUpdated  EventType = "stats_updated"
	EventJobsUpdated   EventType = "jobs_updated"
)

// Event is a renderer-agnostic payload consumed by UI shells.
type Event struct {
	Type    EventType
	Payload any
}

// Stream represents a read-only channel of events.
type Stream <-chan Event
