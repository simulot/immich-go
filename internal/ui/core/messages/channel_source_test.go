package messages

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestChannelSourceClosesOnContextCancel(t *testing.T) {
	ch := make(chan Event, 1)
	var closed int32
	cs := NewChannelSource(Stream(ch), func() { atomic.StoreInt32(&closed, 1) })

	ctx, cancel := context.WithCancel(context.Background())
	_ = cs.Subscribe(ctx)
	cancel()

	select {
	case <-time.After(50 * time.Millisecond):
		if atomic.LoadInt32(&closed) == 0 {
			t.Fatalf("expected close function to be called on cancel")
		}
	}
}

func TestChannelSourceSubscribeReturnsStream(t *testing.T) {
	ch := make(chan Event, 1)
	cs := NewChannelSource(Stream(ch), nil)

	s := cs.Subscribe(context.Background())
	ch <- Event{Type: EventLogLine}

	if _, ok := <-s; !ok {
		t.Fatalf("expected stream to be readable")
	}
}
