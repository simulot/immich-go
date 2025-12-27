package messages

import (
	"context"
	"sync"
)

// ChannelSource adapts an existing Stream plus a close function into an EventSource.
type ChannelSource struct {
	stream Stream
	close  func()
	once   sync.Once
}

// Ensure ChannelSource implements EventSource.
var _ EventSource = (*ChannelSource)(nil)

// NewChannelSource wraps a Stream and optional close function.
func NewChannelSource(stream Stream, closeFn func()) *ChannelSource {
	if stream == nil {
		stream = make(chan Event)
	}
	if closeFn == nil {
		closeFn = func() {}
	}
	return &ChannelSource{stream: stream, close: closeFn}
}

// Subscribe returns the wrapped stream. Cancellation triggers Close.
func (cs *ChannelSource) Subscribe(ctx context.Context) Stream {
	if ctx != nil {
		go func() {
			<-ctx.Done()
			cs.Close()
		}()
	}
	return cs.stream
}

// Close invokes the provided close function once.
func (cs *ChannelSource) Close() error {
	cs.once.Do(func() {
		cs.close()
	})
	return nil
}
