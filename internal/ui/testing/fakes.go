package testing

import (
	"context"
	"sync"

	"github.com/simulot/immich-go/internal/ui/core/messages"
	"github.com/simulot/immich-go/internal/ui/core/state"
)

// MemPublisher captures events for assertions in unit tests.
type MemPublisher struct {
	mu     sync.Mutex
	events []messages.Event
}

// NewMemPublisher constructs an empty in-memory publisher.
func NewMemPublisher() *MemPublisher {
	return &MemPublisher{}
}

// Ensure MemPublisher implements messages.Publisher.
var _ messages.Publisher = (*MemPublisher)(nil)

func (m *MemPublisher) AssetQueued(_ context.Context, ref state.AssetRef) {
	m.append(messages.Event{Type: messages.EventAssetQueued, Payload: ref})
}

func (m *MemPublisher) AssetUploaded(_ context.Context, ref state.AssetRef, bytes int64) {
	payload := struct {
		Ref   state.AssetRef
		Bytes int64
	}{Ref: ref, Bytes: bytes}
	m.append(messages.Event{Type: messages.EventAssetUploaded, Payload: payload})
}

func (m *MemPublisher) AssetFailed(_ context.Context, ref state.AssetRef, reason string) {
	payload := struct {
		Ref    state.AssetRef
		Reason string
	}{Ref: ref, Reason: reason}
	m.append(messages.Event{Type: messages.EventAssetFailed, Payload: payload})
}

func (m *MemPublisher) UpdateStats(_ context.Context, stats state.RunStats) {
	m.append(messages.Event{Type: messages.EventStatsUpdated, Payload: stats})
}

func (m *MemPublisher) AppendLog(_ context.Context, entry state.LogEvent) {
	m.append(messages.Event{Type: messages.EventLogLine, Payload: entry})
}

func (m *MemPublisher) UpdateJobs(_ context.Context, jobs []state.JobSummary) {
	jobsCopy := make([]state.JobSummary, len(jobs))
	copy(jobsCopy, jobs)
	m.append(messages.Event{Type: messages.EventJobsUpdated, Payload: jobsCopy})
}

func (m *MemPublisher) Close() {}

// Events returns a snapshot of the recorded events.
func (m *MemPublisher) Events() []messages.Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]messages.Event, len(m.events))
	copy(out, m.events)
	return out
}

func (m *MemPublisher) append(evt messages.Event) {
	m.mu.Lock()
	m.events = append(m.events, evt)
	m.mu.Unlock()
}
