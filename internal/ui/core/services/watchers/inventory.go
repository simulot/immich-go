package watchers

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/ui/core/messages"
	"github.com/simulot/immich-go/internal/ui/core/state"
)

const DefaultInventoryPollInterval = 2 * time.Second

// inventoryClient captures the subset of methods needed for polling asset statistics.
type inventoryClient interface {
	GetAssetStatistics(ctx context.Context) (immich.UserStatistics, error)
}

// InventorySource polls Immich for asset statistics and emits EventInventoryUpdated.
type InventorySource struct {
	client   inventoryClient
	interval time.Duration
	logger   *slog.Logger
	clock    func() time.Time
	ch       chan messages.Event
	mu       sync.Mutex
	closed   bool
}

// Ensure InventorySource implements messages.EventSource.
var _ messages.EventSource = (*InventorySource)(nil)

// NewInventorySource constructs an inventory event source.
func NewInventorySource(client inventoryClient, interval time.Duration, logger *slog.Logger) *InventorySource {
	if interval <= 0 {
		interval = DefaultInventoryPollInterval
	}
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(nil, nil))
	}
	return &InventorySource{
		client:   client,
		interval: interval,
		logger:   logger,
		clock:    time.Now,
		ch:       make(chan messages.Event, 1),
	}
}

// Subscribe returns the stream for this inventory source.
func (is *InventorySource) Subscribe(ctx context.Context) messages.Stream {
	go is.poll(ctx)
	return messages.Stream(is.ch)
}

// Close stops the source and closes its stream.
func (is *InventorySource) Close() error {
	is.mu.Lock()
	if is.closed {
		is.mu.Unlock()
		return nil
	}
	is.closed = true
	close(is.ch)
	is.mu.Unlock()
	return nil
}

func (is *InventorySource) poll(ctx context.Context) {
	defer func() {
		is.mu.Lock()
		if !is.closed {
			is.closed = true
			close(is.ch)
		}
		is.mu.Unlock()
	}()

	if is.client == nil {
		return
	}

	ticker := time.NewTicker(is.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			inv := is.fetchInventorySnapshot(ctx)
			if inv == nil {
				continue
			}
			is.send(messages.Event{Type: messages.EventInventoryUpdated, Payload: *inv})
		}
	}
}

func (is *InventorySource) fetchInventorySnapshot(ctx context.Context) *state.ServerInventory {
	start := is.clock()
	stats, err := is.client.GetAssetStatistics(ctx)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			is.logger.Debug("inventory watcher: fetch failed", "error", err)
		}
		return nil
	}
	now := is.clock()
	return &state.ServerInventory{
		Photos:    stats.Images,
		Videos:    stats.Videos,
		Total:     stats.Total,
		UpdatedAt: now,
		Latency:   now.Sub(start),
	}
}

func (is *InventorySource) send(evt messages.Event) {
	is.mu.Lock()
	if is.closed {
		is.mu.Unlock()
		return
	}
	select {
	case is.ch <- evt:
	default:
	}
	is.mu.Unlock()
}
