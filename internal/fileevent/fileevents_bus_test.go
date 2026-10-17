package fileevent

import (
	"log/slog"
	"sync"
	"testing"
	"time"
)

func TestBusSubscribeAndPublish_All(t *testing.T) {
	bus := NewBus()
	sub := bus.Subscribe() // all codes
	defer sub.Close()

	ev1 := Event{Code: DiscoveredImage, Time: time.Now()}
	ev2 := Event{Code: ProcessedUploadSuccess, Time: time.Now()}

	bus.Publish(ev1)
	bus.Publish(ev2)

	recv := make([]Event, 0, 2)
	for i := 0; i < 2; i++ {
		select {
		case e := <-sub.Receive():
			recv = append(recv, e)
		case <-time.After(500 * time.Millisecond):
			t.Fatalf("timeout waiting for events; got %d", len(recv))
		}
	}
	if recv[0].Code != ev1.Code || recv[1].Code != ev2.Code {
		t.Fatalf("unexpected event sequence: %+v", recv)
	}
}

func TestBusSubscribe_Filter(t *testing.T) {
	bus := NewBus()
	sub := bus.Subscribe(DiscoveredImage)
	defer sub.Close()

	bus.Publish(Event{Code: DiscoveredImage, Time: time.Now()})
	bus.Publish(Event{Code: DiscoveredVideo, Time: time.Now()})

	select {
	case e := <-sub.Receive():
		if e.Code != DiscoveredImage {
			t.Fatalf("expected DiscoveredImage, got %v", e.Code)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for filtered event")
	}

	// Ensure no second event (video) arrives
	select {
	case <-sub.Receive():
		t.Fatal("unexpected event received for non-subscribed code")
	case <-time.After(200 * time.Millisecond):
		// OK
	}
}

func TestRecorderPublishesToBus(t *testing.T) {
	bus := NewBus()
	sub := bus.Subscribe(DiscoveredImage)
	defer sub.Close()

	r := NewRecorderWithBus(slog.Default(), bus)
	r.RecordWithSize(nil, DiscoveredImage, nil, 1234)

	select {
	case e := <-sub.Receive():
		if e.Code != DiscoveredImage || e.Size != 1234 {
			t.Fatalf("unexpected event: %+v", e)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for recorder event")
	}
}

func TestSubscriptionClose(t *testing.T) {
	bus := NewBus()
	sub := bus.Subscribe(DiscoveredImage)
	ch := sub.Receive()
	sub.Close()

	// Channel must be closed
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("expected closed channel")
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for closed channel")
	}
}

// Close must be safe when called twice, and when called from several goroutines.
func TestSubscriptionDoubleClose(t *testing.T) {
	bus := NewBus()
	sub := bus.Subscribe()
	sub.Close()
	sub.Close()
}

func TestSubscriptionConcurrentClose(t *testing.T) {
	bus := NewBus()
	sub := bus.Subscribe()

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sub.Close()
		}()
	}
	wg.Wait()
}

// A closed subscription is removed from the bus. Publishing after Close delivers
// nothing and must not send on a closed channel.
func TestPublishAfterCloseDeliversNothing(t *testing.T) {
	bus := NewBus()
	sub := bus.Subscribe()
	sub.Close()

	bus.Publish(Event{Code: DiscoveredImage, Time: time.Now()})

	select {
	case _, ok := <-sub.Receive():
		if ok {
			t.Fatal("event delivered after Close")
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for the closed channel")
	}
}
