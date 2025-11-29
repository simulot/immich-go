package upload

import (
	"context"
	"errors"
	"strconv"
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

	uiCtx, cancel := context.WithCancel(ctx)
	uc.uiRunnerCancel = cancel
	legacyStreamNeeded := !uc.NoUI
	var sinks []chan messages.Event
	if legacyStreamNeeded {
		legacyChan := make(chan messages.Event, buffer)
		uc.uiStream = legacyChan
		sinks = append(sinks, legacyChan)
	}
	var runnerStream chan messages.Event
	if uc.app.UIMode != runner.ModeOff {
		runnerStream = make(chan messages.Event, buffer)
		sinks = append(sinks, runnerStream)
	}
	if len(sinks) > 0 {
		go fanOutEventStream(uiCtx, stream, sinks...)
	} else {
		go drainEventStream(uiCtx, stream)
	}

	if processor := uc.app.FileProcessor(); processor != nil {
		processor.SetCountersHook(func(c assettracker.AssetCounters) {
			uc.recordCountersSnapshot(c)
		})
		processor.SetEventHook(func(evtCtx context.Context, code fileevent.Code, file fshelper.FSAndName, size int64, attrs map[string]string) {
			uc.forwardProcessingEvent(evtCtx, code, file, size, attrs)
		})
		uc.startStatsAggregator(uiCtx)
		uc.flushStatsFromCounters(ctx)
	}
	uc.startJobsWatcher(uiCtx)

	if runnerStream != nil {
		go func() {
			cfg := runner.Config{
				Mode:          uc.app.UIMode,
				Experimental:  uc.app.UIExperimental,
				LegacyEnabled: uc.app.UILegacy,
			}
			if err := runner.Run(uiCtx, cfg, runnerStream); err != nil && !errors.Is(err, runner.ErrNoShellSelected) && !errors.Is(err, context.Canceled) {
				uc.app.Log().Debug("ui runner exited", "err", err)
			}
		}()
	}

	return nil
}

func (uc *UpCmd) shutdownUIPipeline() {
	if processor := uc.app.FileProcessor(); processor != nil {
		processor.SetCountersHook(nil)
		processor.SetEventHook(nil)
	}
	uc.stopStatsAggregator()
	uc.stopJobsWatcher()
	if uc.uiPublisher != nil {
		uc.uiPublisher.Close()
	}
	if uc.uiRunnerCancel != nil {
		uc.uiRunnerCancel()
		uc.uiRunnerCancel = nil
	}
	uc.uiStream = nil
}

func fanOutEventStream(ctx context.Context, source messages.Stream, sinks ...chan messages.Event) {
	defer func() {
		for _, sink := range sinks {
			if sink != nil {
				close(sink)
			}
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-source:
			if !ok {
				return
			}
			for _, sink := range sinks {
				if sink == nil {
					continue
				}
				select {
				case sink <- evt:
				case <-ctx.Done():
					return
				}
			}
		}
	}
}

func drainEventStream(ctx context.Context, source messages.Stream) {
	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-source:
			if !ok {
				return
			}
		}
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

func (uc *UpCmd) applyCountersSnapshot(ctx context.Context, counters assettracker.AssetCounters) {
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

const statsAggregationInterval = 200 * time.Millisecond

func (uc *UpCmd) recordCountersSnapshot(counters assettracker.AssetCounters) {
	uc.uiStatsCountersMu.Lock()
	uc.uiStatsCounters = counters
	uc.uiStatsDirty = true
	uc.uiStatsCountersMu.Unlock()
}

func (uc *UpCmd) startStatsAggregator(ctx context.Context) {
	uc.stopStatsAggregator()
	aggCtx, cancel := context.WithCancel(ctx)
	uc.uiStatsCancel = cancel
	go uc.runStatsAggregator(aggCtx)
}

func (uc *UpCmd) stopStatsAggregator() {
	if uc.uiStatsCancel != nil {
		uc.uiStatsCancel()
		uc.uiStatsCancel = nil
	}
}

func (uc *UpCmd) runStatsAggregator(ctx context.Context) {
	ticker := time.NewTicker(statsAggregationInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			uc.flushStatsFromCounters(ctx)
			return
		case <-ticker.C:
			uc.flushStatsFromCounters(ctx)
		}
	}
}

func (uc *UpCmd) flushStatsFromCounters(ctx context.Context) {
	if counters, ok := uc.consumeCountersSnapshot(); ok {
		uc.applyCountersSnapshot(ctx, counters)
	}
}

func (uc *UpCmd) startJobsWatcher(ctx context.Context) {
	uc.stopJobsWatcher()
	if uc.client.AdminImmich == nil || uc.uiPublisher == nil {
		return
	}
	interval := uc.app.UIJobsPollInterval
	cfg := runner.JobsWatcherConfig{
		Client:    uc.client.AdminImmich,
		Publisher: uc.uiPublisher,
		Interval:  interval,
	}
	if log := uc.app.Log(); log != nil {
		cfg.Logger = log.Logger
	}
	uc.uiJobsCancel = runner.StartJobsWatcher(ctx, cfg)
}

func (uc *UpCmd) stopJobsWatcher() {
	if uc.uiJobsCancel != nil {
		uc.uiJobsCancel()
		uc.uiJobsCancel = nil
	}
}

func (uc *UpCmd) consumeCountersSnapshot() (assettracker.AssetCounters, bool) {
	uc.uiStatsCountersMu.Lock()
	defer uc.uiStatsCountersMu.Unlock()
	if !uc.uiStatsDirty {
		return assettracker.AssetCounters{}, false
	}
	counters := uc.uiStatsCounters
	uc.uiStatsDirty = false
	return counters, true
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

type processingLogConfig struct {
	level   string
	message string
}

var processingEventLogConfig = map[fileevent.Code]processingLogConfig{
	fileevent.ProcessedAssociatedMetadata: {level: "info", message: "associated metadata"},
	fileevent.ProcessedMissingMetadata:    {level: "warn", message: "missing metadata"},
	fileevent.ProcessedAlbumAdded:         {level: "info", message: "added to album"},
	fileevent.ProcessedTagged:             {level: "info", message: "tag applied"},
	fileevent.ProcessedStacked:            {level: "info", message: "stack updated"},
	fileevent.ProcessedLivePhoto:          {level: "info", message: "live photo processed"},
}

func (uc *UpCmd) forwardProcessingEvent(ctx context.Context, code fileevent.Code, file fshelper.FSAndName, size int64, attrs map[string]string) {
	cfg, ok := processingEventLogConfig[code]
	if !ok {
		return
	}
	details := make(map[string]string, len(attrs)+3)
	for k, v := range attrs {
		details[k] = v
	}
	if path := safeFullName(file); path != "" {
		details["file"] = path
	}
	if size > 0 {
		details["size_bytes"] = strconv.FormatInt(size, 10)
	}
	details["event_code"] = code.String()
	uc.publishLog(ctx, cfg.level, cfg.message, details)
}
