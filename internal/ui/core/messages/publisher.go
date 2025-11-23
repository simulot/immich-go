package messages

import (
	"context"

	"github.com/simulot/immich-go/internal/ui/core/state"
)

// Publisher exposes strongly-typed methods for emitting UI-friendly events.
type Publisher interface {
	AssetQueued(ctx context.Context, event state.AssetEvent)
	AssetUploaded(ctx context.Context, event state.AssetEvent)
	AssetFailed(ctx context.Context, event state.AssetEvent)
	UpdateStats(ctx context.Context, stats state.RunStats)
	AppendLog(ctx context.Context, entry state.LogEvent)
	UpdateJobs(ctx context.Context, jobs []state.JobSummary)
	Close()
}

// NoopPublisher is the default implementation when the UI subsystem is disabled.
type NoopPublisher struct{}

// Ensure NoopPublisher satisfies Publisher.
var _ Publisher = (*NoopPublisher)(nil)

// AssetQueued implements Publisher.
func (NoopPublisher) AssetQueued(context.Context, state.AssetEvent) {}

// AssetUploaded implements Publisher.
func (NoopPublisher) AssetUploaded(context.Context, state.AssetEvent) {}

// AssetFailed implements Publisher.
func (NoopPublisher) AssetFailed(context.Context, state.AssetEvent) {}

// UpdateStats implements Publisher.
func (NoopPublisher) UpdateStats(context.Context, state.RunStats) {}

// AppendLog implements Publisher.
func (NoopPublisher) AppendLog(context.Context, state.LogEvent) {}

// UpdateJobs implements Publisher.
func (NoopPublisher) UpdateJobs(context.Context, []state.JobSummary) {}

// Close implements Publisher.
func (NoopPublisher) Close() {}
