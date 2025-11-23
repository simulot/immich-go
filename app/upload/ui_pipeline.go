package upload

import (
	"context"
	"errors"
	"time"

	"github.com/simulot/immich-go/internal/assets"
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
	if uc.uiPublisher != nil {
		uc.uiPublisher.Close()
	}
	if uc.uiRunnerCancel != nil {
		uc.uiRunnerCancel()
		uc.uiRunnerCancel = nil
	}
}

func (uc *UpCmd) publishAssetQueued(ctx context.Context, a *assets.Asset) {
	uc.uiPublisher.AssetQueued(ctx, assetRefFromAsset(a))
	uc.updateStats(ctx, func(stats *state.RunStats) {
		stats.Queued++
	})
}

func (uc *UpCmd) publishAssetUploaded(ctx context.Context, a *assets.Asset) {
	bytes := int64(a.FileSize)
	uc.uiPublisher.AssetUploaded(ctx, assetRefFromAsset(a), bytes)
	uc.updateStats(ctx, func(stats *state.RunStats) {
		stats.Uploaded++
		stats.BytesSent += bytes
	})
}

func (uc *UpCmd) publishAssetFailed(ctx context.Context, a *assets.Asset, reason error) {
	msg := ""
	if reason != nil {
		msg = reason.Error()
	}
	uc.uiPublisher.AssetFailed(ctx, assetRefFromAsset(a), msg)
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
	snapshot := uc.uiStats
	uc.uiStatsMu.Unlock()
	uc.uiPublisher.UpdateStats(ctx, snapshot)
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
