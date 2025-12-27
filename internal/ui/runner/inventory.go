package runner

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/ui/core/messages"
	"github.com/simulot/immich-go/internal/ui/core/state"
)

// DefaultInventoryPollInterval controls how often the inventory watcher queries Immich.
const DefaultInventoryPollInterval = 2 * time.Second

// inventoryClient captures the subset of methods needed for polling asset statistics.
type inventoryClient interface {
	GetAssetStatistics(ctx context.Context) (immich.UserStatistics, error)
}

// InventoryWatcherConfig configures the background inventory watcher.
type InventoryWatcherConfig struct {
	Client    inventoryClient
	Publisher messages.Publisher
	Interval  time.Duration
	Logger    *slog.Logger
	Clock     func() time.Time
}

// StartInventoryWatcher polls Immich for asset statistics and publishes inventory updates.
func StartInventoryWatcher(ctx context.Context, cfg InventoryWatcherConfig) context.CancelFunc {
	if cfg.Client == nil || cfg.Publisher == nil {
		return func() {}
	}

	interval := cfg.Interval
	if interval <= 0 {
		interval = DefaultInventoryPollInterval
	}
	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}

	watchCtx, cancel := context.WithCancel(ctx)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-watchCtx.Done():
				return
			case <-ticker.C:
				publishInventorySnapshot(watchCtx, cfg, clock)
			}
		}
	}()

	return cancel
}

func publishInventorySnapshot(ctx context.Context, cfg InventoryWatcherConfig, clock func() time.Time) {
	start := clock()
	stats, err := cfg.Client.GetAssetStatistics(ctx)
	if err != nil {
		if cfg.Logger != nil && !errors.Is(err, context.Canceled) {
			cfg.Logger.Debug("inventory watcher: fetch failed", "error", err)
		}
		return
	}
	now := clock()
	inv := state.ServerInventory{
		Photos:    stats.Images,
		Videos:    stats.Videos,
		Total:     stats.Total,
		UpdatedAt: now,
		Latency:   now.Sub(start),
	}
	cfg.Publisher.UpdateInventory(ctx, inv)
}
