package messages

import (
	"context"
	"testing"
	"time"

	"github.com/simulot/immich-go/internal/ui/core/state"
)

func TestChannelPublisherEmitsEvents(t *testing.T) {
	pub, stream := NewChannelPublisher(4)
	ctx := context.Background()

	pub.AssetQueued(ctx, state.AssetEvent{Stage: state.AssetStageQueued})
	pub.UpdateStats(ctx, state.RunStats{Queued: 1})
	pub.AppendLog(ctx, state.LogEvent{Message: "hello"})

	want := []EventType{EventAssetQueued, EventStatsUpdated, EventLogLine}
	for _, typ := range want {
		select {
		case evt := <-stream:
			if evt.Type != typ {
				t.Fatalf("expected event %s, got %s", typ, evt.Type)
			}
		case <-time.After(time.Second):
			t.Fatalf("timeout waiting for event %s", typ)
		}
	}

	pub.Close()
	if _, ok := <-stream; ok {
		t.Fatalf("expected stream to be closed after publisher.Close")
	}
}

func TestChannelPublisherCopiesJobUpdates(t *testing.T) {
	pub, stream := NewChannelPublisher(1)
	ctx := context.Background()

	jobs := []state.JobSummary{{Name: "original"}}
	pub.UpdateJobs(ctx, jobs)
	jobs[0].Name = "mutated"

	select {
	case evt := <-stream:
		gotJobs, ok := evt.Payload.([]state.JobSummary)
		if !ok {
			t.Fatalf("expected job slice payload, got %T", evt.Payload)
		}
		if gotJobs[0].Name != "original" {
			t.Fatalf("expected immutable payload, got %q", gotJobs[0].Name)
		}
	case <-time.After(time.Second):
		t.Fatalf("timeout waiting for jobs event")
	}
}

func TestChannelPublisherRespectsContextCancellation(t *testing.T) {
	pub, stream := NewChannelPublisher(1)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	pub.AssetQueued(ctx, state.AssetEvent{Stage: state.AssetStageQueued})

	select {
	case evt := <-stream:
		t.Fatalf("expected no event, received %v", evt.Type)
	default:
	}
}

func TestChannelPublisherCloseIdempotent(t *testing.T) {
	pub, stream := NewChannelPublisher(1)
	pub.Close()
	pub.Close()

	if _, ok := <-stream; ok {
		t.Fatalf("stream should be closed")
	}
}
