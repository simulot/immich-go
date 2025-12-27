package messages

import "context"

// LegacyAdapter subscribes to the merged event bus and forwards events to a legacy channel.
type LegacyAdapter struct {
	src    EventBusConsumer
	sink   chan Event
	closef func()
}

// EventBusConsumer is the minimal interface of EventBus we need.
type EventBusConsumer interface {
	Events() Stream
}

// NewLegacyAdapter wires a bus consumer to a legacy sink channel.
func NewLegacyAdapter(src EventBusConsumer, sink chan Event, closef func()) *LegacyAdapter {
	if closef == nil {
		closef = func() {}
	}
	return &LegacyAdapter{src: src, sink: sink, closef: closef}
}

// Run starts forwarding until ctx is done or source stream closes.
func (a *LegacyAdapter) Run(ctx context.Context) {
	if a.src == nil || a.sink == nil {
		return
	}
	stream := a.src.Events()
	for {
		select {
		case <-ctx.Done():
			a.closef()
			return
		case evt, ok := <-stream:
			if !ok {
				a.closef()
				return
			}
			select {
			case a.sink <- evt:
			case <-ctx.Done():
				a.closef()
				return
			}
		}
	}
}
