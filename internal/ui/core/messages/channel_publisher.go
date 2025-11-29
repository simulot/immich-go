package messages

import (
	"context"
	"sync"

	"github.com/simulot/immich-go/internal/ui/core/state"
)

// ChannelPublisher publishes UI events onto a buffered channel consumed by shells.
type ChannelPublisher struct {
	ch        chan Event
	closeOnce sync.Once
}

// NewChannelPublisher returns a publisher backed by a buffered channel and the read-only stream.
func NewChannelPublisher(buffer int) (*ChannelPublisher, Stream) {
	if buffer <= 0 {
		buffer = 1
	}
	ch := make(chan Event, buffer)
	return &ChannelPublisher{ch: ch}, Stream(ch)
}

var _ Publisher = (*ChannelPublisher)(nil)

// AssetQueued implements Publisher.
func (p *ChannelPublisher) AssetQueued(ctx context.Context, event state.AssetEvent) {
	p.send(ctx, Event{Type: EventAssetQueued, Payload: event})
}

// AssetUploaded implements Publisher.
func (p *ChannelPublisher) AssetUploaded(ctx context.Context, event state.AssetEvent) {
	p.send(ctx, Event{Type: EventAssetUploaded, Payload: event})
}

// AssetFailed implements Publisher.
func (p *ChannelPublisher) AssetFailed(ctx context.Context, event state.AssetEvent) {
	p.send(ctx, Event{Type: EventAssetFailed, Payload: event})
}

// UpdateStats implements Publisher.
func (p *ChannelPublisher) UpdateStats(ctx context.Context, stats state.RunStats) {
	p.send(ctx, Event{Type: EventStatsUpdated, Payload: stats})
}

// AppendLog implements Publisher.
func (p *ChannelPublisher) AppendLog(ctx context.Context, entry state.LogEvent) {
	p.send(ctx, Event{Type: EventLogLine, Payload: entry})
}

// UpdateJobs implements Publisher.
func (p *ChannelPublisher) UpdateJobs(ctx context.Context, jobs []state.JobSummary) {
	copyJobs := make([]state.JobSummary, len(jobs))
	copy(copyJobs, jobs)
	p.send(ctx, Event{Type: EventJobsUpdated, Payload: copyJobs})
}

// Close closes the underlying channel so consumers may exit.
func (p *ChannelPublisher) Close() {
	p.closeOnce.Do(func() {
		close(p.ch)
	})
}

func (p *ChannelPublisher) send(ctx context.Context, evt Event) {
	if p == nil {
		return
	}
	select {
	case <-ctx.Done():
	case p.ch <- evt:
	default:
	}
}
