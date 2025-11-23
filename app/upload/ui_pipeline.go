package upload

import (
	"context"
	"errors"
	"time"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/assettracker"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/filetypes"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/simulot/immich-go/internal/ui/core/messages"
	"github.com/simulot/immich-go/internal/ui/core/state"
	"github.com/simulot/immich-go/internal/ui/runner"
)

func (uc *UpCmd) initUIPipeline(ctx context.Context) error {
	if uc.uiPublisher == nil {
		uc.uiPublisher = messages.NoopPublisher{}
	}

	uc.uiStatsMu.Lock()
	uc.uiStats = state.RunStats{StartedAt: time.Now()}
	statsSnapshot := uc.uiStats
	uc.uiStatsMu.Unlock()
	uc.uiPublisher.UpdateStats(ctx, statsSnapshot)

	if uc.NoUI || !uc.app.UIExperimental {
		return nil
	}

	buffer := uc.app.UIEventBuffer
	if buffer <= 0 {
		buffer = 1
	}
	publisher, stream := messages.NewChannelPublisher(buffer)
	uc.uiPublisher = publisher
	uc.uiPublisher.UpdateStats(ctx, uc.snapshotStats())

	if processor := uc.app.FileProcessor(); processor != nil {
		processor.SetCountersHook(func(c assettracker.AssetCounters) {
			uc.updateStatsFromCounters(ctx, c)
		})
	}

	uiCtx, cancel := context.WithCancel(ctx)
	uc.uiRunnerCancel = cancel

	go func() {
		cfg := runner.Config{
			Mode:          uc.app.UIMode,
			Experimental:  uc.app.UIExperimental,
			LegacyEnabled: uc.app.UILegacy,
		}
		if err := runner.Run(uiCtx, cfg, stream); err != nil && !errors.Is(err, runner.ErrNoShellSelected) && !errors.Is(err, context.Canceled) {
			uc.app.Log().Debug("ui runner exited", "err", err)
		}
	}()

	return nil
}

func (uc *UpCmd) shutdownUIPipeline() {
	if processor := uc.app.FileProcessor(); processor != nil {
		processor.SetCountersHook(nil)
	}
	if uc.uiPublisher != nil {
		uc.uiPublisher.Close()
	}
	if uc.uiRunnerCancel != nil {
		uc.uiRunnerCancel()
		uc.uiRunnerCancel = nil
	}
}

func (uc *UpCmd) publishAssetQueued(ctx context.Context, a *assets.Asset, code fileevent.Code) {
	event := uc.buildAssetEvent(a, state.AssetStageQueued, code, 0, "", nil)
	uc.uiPublisher.AssetQueued(ctx, event)
	uc.updateStats(ctx, func(stats *state.RunStats) {
		stats.Queued++
	})
}

func (uc *UpCmd) publishAssetUploaded(ctx context.Context, a *assets.Asset, code fileevent.Code, bytes int64, details map[string]string) {
	event := uc.buildAssetEvent(a, state.AssetStageUploaded, code, bytes, "", details)
	uc.uiPublisher.AssetUploaded(ctx, event)
	uc.updateStats(ctx, func(stats *state.RunStats) {
		stats.Uploaded++
		stats.BytesSent += bytes
	})
}

func (uc *UpCmd) publishAssetFailed(ctx context.Context, a *assets.Asset, code fileevent.Code, reason error, details map[string]string) {
	msg := ""
	if reason != nil {
		msg = reason.Error()
	}
	event := uc.buildAssetEvent(a, state.AssetStageFailed, code, 0, msg, details)
	uc.uiPublisher.AssetFailed(ctx, event)
	uc.updateStats(ctx, func(stats *state.RunStats) {
		stats.Failed++
	})
}

func (uc *UpCmd) publishLog(ctx context.Context, level, message string, details map[string]string) {
	if details == nil {
		details = map[string]string{}
	}
	uc.uiPublisher.AppendLog(ctx, state.LogEvent{
		Level:     level,
		Message:   message,
		Timestamp: time.Now(),
		Details:   details,
	})
}

func (uc *UpCmd) snapshotStats() state.RunStats {
	uc.uiStatsMu.Lock()
	defer uc.uiStatsMu.Unlock()
	return uc.uiStats
}

func (uc *UpCmd) updateStats(ctx context.Context, mutate func(*state.RunStats)) {
	if mutate == nil {
		return
	}
	uc.uiStatsMu.Lock()
	mutate(&uc.uiStats)
	uc.uiStats.HasErrors = (uc.uiStats.Failed > 0) || (uc.uiStats.ErrorCount > 0)
	snapshot := uc.uiStats
	uc.uiStatsMu.Unlock()
	uc.uiPublisher.UpdateStats(ctx, snapshot)
}

func (uc *UpCmd) updateStatsFromCounters(ctx context.Context, counters assettracker.AssetCounters) {
	uc.updateStats(ctx, func(stats *state.RunStats) {
		stats.Pending = int(counters.Pending)
		stats.PendingBytes = counters.PendingSize
		stats.Processed = int(counters.Processed)
		stats.ProcessedBytes = counters.ProcessedSize
		stats.Discarded = int(counters.Discarded)
		stats.DiscardedBytes = counters.DiscardedSize
		stats.ErrorCount = int(counters.Errors)
		stats.ErrorBytes = counters.ErrorSize
		stats.TotalDiscovered = int(counters.Total())
		stats.TotalDiscoveredBytes = counters.AssetSize
	})
}

func (uc *UpCmd) buildAssetEvent(a *assets.Asset, stage state.AssetStage, code fileevent.Code, bytes int64, reason string, details map[string]string) state.AssetEvent {
	evt := state.AssetEvent{
		Asset:     assetRefFromAsset(a),
		Stage:     stage,
		Code:      state.AssetEventCode(code),
		CodeLabel: code.String(),
		Bytes:     bytes,
		Reason:    reason,
	}
	if len(details) > 0 {
		detailCopy := make(map[string]string, len(details))
		for k, v := range details {
			detailCopy[k] = v
		}
		evt.Details = detailCopy
	}
	return evt
}

func assetRefFromAsset(a *assets.Asset) state.AssetRef {
	if a == nil {
		return state.AssetRef{}
	}
	ref := state.AssetRef{ID: a.ID}
	if fullname := safeFullName(a.File); fullname != "" {
		ref.Path = fullname
	} else if a.OriginalFileName != "" {
		ref.Path = a.OriginalFileName
	}
	return ref
}

func safeFullName(fn fshelper.FSAndName) string {
	if fn.Name() == "" {
		return ""
	}
	return fn.FullName()
}

func assetDiscoveryCode(a *assets.Asset) fileevent.Code {
	if a == nil {
		return fileevent.NotHandled
	}
	if a.NameInfo.Type == filetypes.TypeVideo {
		return fileevent.DiscoveredVideo
	}
	return fileevent.DiscoveredImage
}
