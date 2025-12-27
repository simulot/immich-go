package messages

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type fakeSource struct {
	ch            chan Event
	closeOnCancel bool
	closeOnce     sync.Once
}

func newFakeSource(buffer int, closeOnCancel bool) *fakeSource {
	if buffer <= 0 {
		buffer = 1
	}
	return &fakeSource{ch: make(chan Event, buffer), closeOnCancel: closeOnCancel}
}

func (f *fakeSource) Subscribe(ctx context.Context) Stream {
	if f.closeOnCancel {
		go func() {
			<-ctx.Done()
			f.closeOnce.Do(func() {
				close(f.ch)
			})
		}()
	}
	return Stream(f.ch)
}

func (f *fakeSource) Close() error {
	if f.closeOnCancel {
		f.closeOnce.Do(func() {
			close(f.ch)
		})
	}
	return nil
}

func (f *fakeSource) emit(evt Event) {
	f.ch <- evt
}

func TestEventBusForwardsEvents(t *testing.T) {
	bus := NewEventBus(4)
	source := newFakeSource(4, true)
	if err := bus.AddSource("primary", source); err != nil {
		t.Fatalf("AddSource failed: %v", err)
	}

	source.emit(Event{Type: EventAssetQueued})

	select {
	case evt := <-bus.Events():
		if evt.Type != EventAssetQueued {
			t.Fatalf("expected %s, got %s", EventAssetQueued, evt.Type)
		}
	case <-time.After(time.Second):
		t.Fatalf("timeout waiting for forwarded event")
	}

	bus.Close()
}

func TestEventBusRejectsDuplicateSources(t *testing.T) {
	bus := NewEventBus(2)
	src := newFakeSource(1, true)

	if err := bus.AddSource("dup", src); err != nil {
		t.Fatalf("first AddSource failed: %v", err)
	}
	if err := bus.AddSource("dup", src); err == nil {
		t.Fatalf("expected error on duplicate source name")
	}

	bus.Close()
}

func TestEventBusRemoveSourceStopsForwarding(t *testing.T) {
	bus := NewEventBus(2)
	src := newFakeSource(2, false)
	if err := bus.AddSource("temp", src); err != nil {
		t.Fatalf("AddSource failed: %v", err)
	}

	src.emit(Event{Type: EventStatsUpdated})
	<-bus.Events() // drain first event to ensure wiring is active

	bus.RemoveSource("temp")
	src.emit(Event{Type: EventLogLine})

	select {
	case evt := <-bus.Events():
		t.Fatalf("expected no event after removal, got %v", evt.Type)
	case <-time.After(150 * time.Millisecond):
		// success: nothing forwarded
	}

	bus.Close()
}

func TestEventBusCloseClosesMergedStream(t *testing.T) {
	bus := NewEventBus(1)
	src := newFakeSource(1, true)
	_ = bus.AddSource("close", src)

	bus.Close()

	if _, ok := <-bus.Events(); ok {
		t.Fatalf("expected merged stream to be closed")
	}
}

func TestEventBusAddAfterCloseFails(t *testing.T) {
	bus := NewEventBus(1)
	bus.Close()

	err := bus.AddSource("late", newFakeSource(1, true))
	if err == nil {
		t.Fatalf("expected error when adding after close")
	}
	if !errors.Is(err, ErrBusClosed) {
		t.Fatalf("expected ErrBusClosed, got %v", err)
	}
}
