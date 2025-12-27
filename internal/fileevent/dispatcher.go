package fileevent

import (
	"context"
	"sync"
)

// EventSink receives structured file events and processes them.
// Implementations might log to files, send to UI, collect metrics, etc.
type EventSink interface {
	// HandleEvent receives a file event with structured data
	HandleEvent(ctx context.Context, code Code, file string, size int64, args map[string]any)
}

// Dispatcher manages multiple EventSinks and broadcasts events to all of them.
type Dispatcher struct {
	sinks []EventSink
	mu    sync.RWMutex
}

// NewDispatcher creates a new event dispatcher.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		sinks: []EventSink{},
	}
}

// RegisterSink adds a new event sink to receive events.
func (d *Dispatcher) RegisterSink(sink EventSink) {
	if sink == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.sinks = append(d.sinks, sink)
}

// UnregisterSink removes an event sink.
func (d *Dispatcher) UnregisterSink(sink EventSink) {
	if sink == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	for i, s := range d.sinks {
		if s == sink {
			d.sinks = append(d.sinks[:i], d.sinks[i+1:]...)
			break
		}
	}
}

// Dispatch sends an event to all registered sinks.
// The variadic args are converted to a map of key-value pairs.
func (d *Dispatcher) Dispatch(ctx context.Context, code Code, file string, size int64, args ...any) {
	// Convert variadic args to map
	argsMap := make(map[string]any)
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			if key, ok := args[i].(string); ok {
				argsMap[key] = args[i+1]
			}
		}
	}

	d.mu.RLock()
	sinksCopy := make([]EventSink, len(d.sinks))
	copy(sinksCopy, d.sinks)
	d.mu.RUnlock()

	// Dispatch to all sinks without holding the lock
	for _, sink := range sinksCopy {
		sink.HandleEvent(ctx, code, file, size, argsMap)
	}
}
