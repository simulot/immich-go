package messages

import (
	"context"
	"testing"
	"time"
)

type stubBus struct{ ch chan Event }

func (s stubBus) Events() Stream { return Stream(s.ch) }

func TestLegacyAdapterForwardsEvents(t *testing.T) {
	srcCh := make(chan Event, 1)
	srcCh <- Event{Type: EventLogLine}
	close(srcCh)

	sink := make(chan Event, 1)
	adapter := NewLegacyAdapter(stubBus{ch: srcCh}, sink, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		adapter.Run(ctx)
		close(done)
	}()

	select {
	case evt := <-sink:
		if evt.Type != EventLogLine {
			t.Fatalf("expected EventLogLine, got %s", evt.Type)
		}
	case <-time.After(time.Second):
		t.Fatalf("timeout waiting for forwarded event")
	}
	<-done
}

func TestLegacyAdapterStopsOnContextCancel(t *testing.T) {
	srcCh := make(chan Event)
	sink := make(chan Event, 1)
	closed := false
	adapter := NewLegacyAdapter(stubBus{ch: srcCh}, sink, func() { closed = true })

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		adapter.Run(ctx)
		close(done)
	}()

	cancel()
	<-done

	if !closed {
		t.Fatalf("expected close callback to be invoked")
	}
}
