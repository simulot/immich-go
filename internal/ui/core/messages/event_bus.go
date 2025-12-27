package messages

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// ErrBusClosed is returned when adding sources after the bus is closed.
var ErrBusClosed = errors.New("event bus closed")

// EventSource produces UI events and should respect context cancellation.
type EventSource interface {
	Subscribe(ctx context.Context) Stream
	Close() error
}

// EventBus merges events from multiple sources into a single stream.
type EventBus struct {
	ctx     context.Context
	cancel  context.CancelFunc
	merged  chan Event
	mu      sync.Mutex
	sources map[string]sourceEntry
	closed  bool
	wg      sync.WaitGroup
}

type sourceEntry struct {
	src    EventSource
	cancel context.CancelFunc
}

// NewEventBus constructs an EventBus with a buffered merged stream.
func NewEventBus(buffer int) *EventBus {
	if buffer <= 0 {
		buffer = 1
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &EventBus{
		ctx:     ctx,
		cancel:  cancel,
		merged:  make(chan Event, buffer),
		sources: make(map[string]sourceEntry),
	}
}

// Events returns the merged stream of all registered sources.
func (b *EventBus) Events() Stream {
	return Stream(b.merged)
}

// AddSource registers a named source and begins forwarding its events.
func (b *EventBus) AddSource(name string, src EventSource) error {
	if src == nil {
		return fmt.Errorf("event source %q is nil", name)
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return ErrBusClosed
	}
	if _, exists := b.sources[name]; exists {
		return fmt.Errorf("event source %q already registered", name)
	}

	srcCtx, cancel := context.WithCancel(b.ctx)
	stream := src.Subscribe(srcCtx)
	b.sources[name] = sourceEntry{src: src, cancel: cancel}

	b.wg.Add(1)
	go b.forward(srcCtx, stream)

	return nil
}

// RemoveSource stops forwarding events from the named source.
func (b *EventBus) RemoveSource(name string) {
	b.mu.Lock()
	entry, ok := b.sources[name]
	if ok {
		delete(b.sources, name)
	}
	b.mu.Unlock()

	if ok {
		entry.cancel()
		_ = entry.src.Close()
	}
}

func (b *EventBus) forward(ctx context.Context, stream Stream) {
	defer b.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-stream:
			if !ok {
				return
			}
			select {
			case <-ctx.Done():
				return
			case b.merged <- evt:
			}
		}
	}
}

// Close stops all sources, waits for forwarders to exit, and closes the merged stream.
func (b *EventBus) Close() {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return
	}
	b.closed = true
	b.cancel()

	entries := make([]sourceEntry, 0, len(b.sources))
	for _, entry := range b.sources {
		entries = append(entries, entry)
	}
	b.sources = make(map[string]sourceEntry)
	b.mu.Unlock()

	for _, entry := range entries {
		entry.cancel()
		_ = entry.src.Close()
	}

	b.wg.Wait()
	close(b.merged)
}
