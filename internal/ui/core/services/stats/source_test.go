package stats

import (
	"context"
	"testing"
	"time"

	"github.com/simulot/immich-go/internal/assettracker"
	"github.com/simulot/immich-go/internal/ui/core/state"
)

type fakeClock struct {
	times []time.Time
	idx   int
}

func (f *fakeClock) now() time.Time {
	t := f.times[f.idx]
	f.idx++
	return t
}

func TestStatsSourceEmitsOnAssetQueued(t *testing.T) {
	clock := &fakeClock{times: []time.Time{time.Unix(0, 0), time.Unix(1, 0)}}
	s := NewSource(2, state.RunStats{}, clock.now)
	stream := s.Subscribe(context.Background())

	s.AssetQueued()

	select {
	case evt := <-stream:
		stats, ok := evt.Payload.(state.RunStats)
		if !ok {
			t.Fatalf("expected RunStats payload, got %T", evt.Payload)
		}
		if stats.Queued != 1 {
			t.Fatalf("expected queued=1, got %d", stats.Queued)
		}
	case <-time.After(time.Second):
		t.Fatalf("timeout waiting for stats event")
	}
}

func TestStatsSourceApplyCounters(t *testing.T) {
	clock := &fakeClock{times: []time.Time{time.Unix(0, 0), time.Unix(1, 0)}}
	s := NewSource(1, state.RunStats{}, clock.now)
	stream := s.Subscribe(context.Background())

	counters := assettracker.AssetCounters{
		Pending:       2,
		PendingSize:   20,
		Processed:     3,
		ProcessedSize: 30,
		Discarded:     1,
		DiscardedSize: 10,
		Errors:        4,
		ErrorSize:     40,
		AssetSize:     100,
	}

	s.ApplyCounters(counters)

	evt := <-stream
	stats := evt.Payload.(state.RunStats)
	if stats.Pending != 2 || stats.Processed != 3 || stats.Discarded != 1 || stats.ErrorCount != 4 {
		t.Fatalf("unexpected counters: pending=%d processed=%d discarded=%d errorcount=%d", stats.Pending, stats.Processed, stats.Discarded, stats.ErrorCount)
	}
	if stats.TotalDiscovered != 10 { // Total() = 2+3+1+4
		t.Fatalf("expected TotalDiscovered=10, got %d", stats.TotalDiscovered)
	}
	if stats.TotalDiscoveredBytes != 100 {
		t.Fatalf("unexpected TotalDiscoveredBytes: %d", stats.TotalDiscoveredBytes)
	}
}

func TestStatsSourceThroughputSampling(t *testing.T) {
	clock := &fakeClock{times: []time.Time{
		time.Unix(0, 0),                  // init
		time.Unix(0, 0),                  // first mutate
		time.Unix(0, int64(time.Second)), // second mutate after 1s
	}}
	s := NewSource(1, state.RunStats{}, clock.now)
	stream := s.Subscribe(context.Background())

	s.AssetUploaded(1000)
	<-stream // drain first update (no throughput yet)

	s.AssetUploaded(1000)
	evt := <-stream
	stats := evt.Payload.(state.RunStats)
	if len(stats.ThroughputSamples) != 1 {
		t.Fatalf("expected 1 throughput sample, got %d", len(stats.ThroughputSamples))
	}
	bps := stats.ThroughputSamples[0].BytesPerSecond
	if bps <= 0 {
		t.Fatalf("expected bytes per second > 0, got %f", bps)
	}
}

func TestStatsSourceClosesOnContextCancel(t *testing.T) {
	clock := &fakeClock{times: []time.Time{time.Unix(0, 0)}}
	s := NewSource(1, state.RunStats{}, clock.now)
	ctx, cancel := context.WithCancel(context.Background())
	stream := s.Subscribe(ctx)
	cancel()

	select {
	case _, ok := <-stream:
		if ok {
			t.Fatalf("expected stream to be closed after cancel")
		}
	case <-time.After(time.Second):
		t.Fatalf("timeout waiting for stream close")
	}
}
