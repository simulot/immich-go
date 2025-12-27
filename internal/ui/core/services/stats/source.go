package stats

import (
	"context"
	"sync"
	"time"

	"github.com/simulot/immich-go/internal/assettracker"
	"github.com/simulot/immich-go/internal/ui/core/messages"
	"github.com/simulot/immich-go/internal/ui/core/state"
)

const (
	statsAggregationInterval    = 200 * time.Millisecond
	throughputSampleMinInterval = 200 * time.Millisecond
	maxThroughputSamples        = 64
)

// Source maintains run statistics and emits updates as EventStatsUpdated events.
type Source struct {
	ch         chan messages.Event
	mu         sync.Mutex
	stats      state.RunStats
	prevBytes  int64
	prevSample time.Time
	clock      func() time.Time
	closed     bool
}

// Ensure Source implements messages.EventSource.
var _ messages.EventSource = (*Source)(nil)

// NewSource constructs a stats source with an initial RunStats snapshot.
func NewSource(buffer int, initial state.RunStats, clock func() time.Time) *Source {
	if buffer <= 0 {
		buffer = 1
	}
	if clock == nil {
		clock = time.Now
	}
	if initial.Stage == "" {
		initial.Stage = state.StageRunning
	}
	if initial.StartedAt.IsZero() {
		initial.StartedAt = clock()
	}
	if initial.LastUpdated.IsZero() {
		initial.LastUpdated = initial.StartedAt
	}

	return &Source{
		ch:    make(chan messages.Event, buffer),
		stats: initial,
		clock: clock,
	}
}

// Subscribe returns the stream for this stats source.
func (s *Source) Subscribe(ctx context.Context) messages.Stream {
	if ctx != nil {
		go func() {
			<-ctx.Done()
			s.Close()
		}()
	}
	return messages.Stream(s.ch)
}

// Close stops the source and closes its stream.
func (s *Source) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	close(s.ch)
	s.mu.Unlock()
	return nil
}

// AssetQueued increments queued counters and emits an update.
func (s *Source) AssetQueued() {
	s.mutate(func(st *state.RunStats) {
		st.Queued++
	})
}

// AssetUploaded increments uploaded counters, bytes, and emits an update.
func (s *Source) AssetUploaded(bytes int64) {
	s.mutate(func(st *state.RunStats) {
		st.Uploaded++
		st.BytesSent += bytes
	})
}

// AssetFailed increments failure counters and emits an update.
func (s *Source) AssetFailed() {
	s.mutate(func(st *state.RunStats) {
		st.Failed++
	})
}

// ApplyCounters updates aggregate counters from the asset tracker snapshot.
func (s *Source) ApplyCounters(counters assettracker.AssetCounters) {
	s.mutate(func(st *state.RunStats) {
		st.Pending = int(counters.Pending)
		st.PendingBytes = counters.PendingSize
		// Clamp cumulative counters to be non-decreasing to avoid resets emitted after shutdown.
		if v := int(counters.Processed); v > st.Processed {
			st.Processed = v
		}
		if v := counters.ProcessedSize; v > st.ProcessedBytes {
			st.ProcessedBytes = v
		}
		if v := int(counters.Discarded); v > st.Discarded {
			st.Discarded = v
		}
		if v := counters.DiscardedSize; v > st.DiscardedBytes {
			st.DiscardedBytes = v
		}
		if v := int(counters.Errors); v > st.ErrorCount {
			st.ErrorCount = v
		}
		if v := counters.ErrorSize; v > st.ErrorBytes {
			st.ErrorBytes = v
		}
		if v := int(counters.Total()); v > st.TotalDiscovered {
			st.TotalDiscovered = v
		}
		if v := counters.AssetSize; v > st.TotalDiscoveredBytes {
			st.TotalDiscoveredBytes = v
		}
		st.InFlight = st.Pending
	})
}

// SetStage sets the current run stage and emits an update.
func (s *Source) SetStage(stage state.RunStage) {
	s.mutate(func(st *state.RunStats) {
		st.Stage = stage
	})
}

// SetUploadPaused updates the paused flag and emits an update.
func (s *Source) SetUploadPaused(paused bool) {
	s.mutate(func(st *state.RunStats) {
		st.UploadPaused = paused
	})
}

// AddError increments error counters and marks errors present.
func (s *Source) AddError(deltaCount int, deltaBytes int64) {
	s.mutate(func(st *state.RunStats) {
		st.ErrorCount += deltaCount
		st.ErrorBytes += deltaBytes
	})
}

func (s *Source) mutate(mutate func(*state.RunStats)) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	if mutate != nil {
		mutate(&s.stats)
	}
	now := s.clock()
	if s.stats.Stage == "" {
		s.stats.Stage = state.StageRunning
	}
	s.stats.HasErrors = (s.stats.Failed > 0) || (s.stats.ErrorCount > 0)
	s.stats.LastUpdated = now
	s.updateThroughputLocked(now)
	snapshot := state.CloneRunStats(s.stats)
	s.mu.Unlock()

	s.send(messages.Event{Type: messages.EventStatsUpdated, Payload: snapshot})
}

func (s *Source) updateThroughputLocked(now time.Time) {
	if s.prevSample.IsZero() {
		s.prevSample = now
	}
	deltaBytes := s.stats.BytesSent - s.prevBytes
	if deltaBytes <= 0 {
		return
	}
	interval := now.Sub(s.prevSample)
	if interval < throughputSampleMinInterval {
		return
	}
	if interval <= 0 {
		interval = throughputSampleMinInterval
	}
	bytesPerSecond := float64(deltaBytes) / interval.Seconds()
	sample := state.ThroughputSample{Timestamp: now, BytesPerSecond: bytesPerSecond}
	s.stats.ThroughputSamples = append(s.stats.ThroughputSamples, sample)
	if len(s.stats.ThroughputSamples) > maxThroughputSamples {
		start := len(s.stats.ThroughputSamples) - maxThroughputSamples
		s.stats.ThroughputSamples = append([]state.ThroughputSample(nil), s.stats.ThroughputSamples[start:]...)
	}
	s.prevSample = now
	s.prevBytes = s.stats.BytesSent
}

func (s *Source) send(evt messages.Event) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	select {
	case s.ch <- evt:
	default:
	}
	s.mu.Unlock()
}
