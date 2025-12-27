package upload

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/assettracker"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/filetypes"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/simulot/immich-go/internal/ui/core/messages"
	statssvc "github.com/simulot/immich-go/internal/ui/core/services/stats"
	"github.com/simulot/immich-go/internal/ui/core/services/watchers"
	"github.com/simulot/immich-go/internal/ui/core/state"
	"github.com/simulot/immich-go/internal/ui/runner"
)

func (uc *UpCmd) initUIPipeline(ctx context.Context) {
	if uc.uiPublisher == nil {
		uc.uiPublisher = messages.NoopPublisher{}
	}

	// Allow tests to capture stats even when UI is disabled.
	testCapture := ctx.Value("test-stats-capture")
	forceTestCapture := testCapture != nil

	now := time.Now()
	uc.uiStatsMu.Lock()
	uc.uiStats = state.RunStats{
		StartedAt:   now,
		Stage:       state.StageRunning,
		Workers:     uc.app.ConcurrentTask,
		LastUpdated: now,
	}
	uc.uiStatsPrevBytes = 0
	uc.uiStatsPrevSample = now
	statsSnapshot := state.CloneRunStats(uc.uiStats)
	uc.uiStatsMu.Unlock()

	if !forceTestCapture && (uc.NoUI || !uc.app.UIExperimental) {
		uc.uiPublisher.UpdateStats(ctx, statsSnapshot)
		return
	}

	buffer := uc.app.UIEventBuffer
	if buffer <= 0 {
		buffer = 1
	}

	// Create stats source as the single producer of stats events
	uc.uiStatsSource = statssvc.NewSource(buffer, statsSnapshot, nil)
	uc.uiBus = messages.NewEventBus(buffer)
	_ = uc.uiBus.AddSource("stats", uc.uiStatsSource)
	// Primary pipeline publisher + stream
	publisher, stream := messages.NewChannelPublisher(buffer)
	uc.uiPublisher = publisher

	// Set up slog handler to forward logs to UI
	uc.setupUILogHandler(ctx)

	uiCtx, cancel := context.WithCancel(ctx)
	uc.uiRunnerCancel = cancel

	// Add pipeline stream to bus
	if err := uc.uiBus.AddSource("pipeline", messages.NewChannelSource(stream, publisher.Close)); err != nil {
		uc.app.Log().Debug("ui bus: add pipeline source failed", "err", err)
	}

	// Add watcher sources when available (jobs / inventory)
	if uc.client.AdminImmich != nil {
		js := watchers.NewJobsSource(uc.client.AdminImmich, uc.app.UIJobsPollInterval, uc.app.Log().Logger)
		if err := uc.uiBus.AddSource("jobs", js); err != nil {
			uc.app.Log().Debug("ui bus: add jobs source failed", "err", err)
		}
	}
	if uc.client.Immich != nil {
		is := watchers.NewInventorySource(uc.client.Immich, uc.app.UIInventoryPollInterval, uc.app.Log().Logger)
		if err := uc.uiBus.AddSource("inventory", is); err != nil {
			uc.app.Log().Debug("ui bus: add inventory source failed", "err", err)
		}
	}

	// Fan-out from the merged bus stream
	legacyStreamNeeded := !uc.NoUI && (!uc.app.UIExperimental || uc.app.UILegacy)
	var sinks []chan messages.Event
	var dumpStream chan messages.Event
	if legacyStreamNeeded {
		legacyChan := make(chan messages.Event, buffer)
		uc.uiStream = legacyChan
		sinks = append(sinks, legacyChan)

		// Legacy adapter: keep old TUI working off the bus
		if uc.app.UIExperimental {
			adapter := messages.NewLegacyAdapter(uc.uiBus, legacyChan, func() { close(legacyChan) })
			go adapter.Run(uiCtx)
			// Adapter owns closing; avoid double-close in fan-out
			sinks = sinks[:len(sinks)-1]
		}
	}
	var runnerStream chan messages.Event
	if uc.app.UIMode != runner.ModeOff {
		runnerStream = make(chan messages.Event, buffer)
		sinks = append(sinks, runnerStream)
	}
	if uc.app.UIDumpEvents && uc.app.Log() != nil && uc.app.Log().Logger != nil {
		dumpStream = make(chan messages.Event, buffer)
		sinks = append(sinks, dumpStream)
	}
	// Optional test sink: capture stats events for E2E assertions
	if testCapture != nil {
		testSink := make(chan messages.Event, buffer)
		sinks = append(sinks, testSink)
		go captureTestStats(uiCtx, testSink, testCapture)
	}
	if len(sinks) > 0 {
		go fanOutEventStream(uiCtx, uc.uiBus.Events(), sinks...)
	} else {
		go drainEventStream(uiCtx, uc.uiBus.Events())
	}
	if dumpStream != nil {
		go logUIEvents(uiCtx, dumpStream, uc.app.Log().Logger)
	}

	if processor := uc.app.FileProcessor(); processor != nil {
		if uc.uiStatsSource != nil {
			processor.SetCountersHook(func(c assettracker.AssetCounters) {
				uc.uiStatsSource.ApplyCounters(c)
			})
		} else {
			processor.SetCountersHook(func(c assettracker.AssetCounters) {
				uc.recordCountersSnapshot(c)
			})
		}

		processor.SetEventHook(func(evtCtx context.Context, code fileevent.Code, file fshelper.FSAndName, size int64, attrs map[string]string) {
			uc.forwardProcessingEvent(evtCtx, code, file, size, attrs)
		})

		if uc.uiStatsSource != nil {
			uc.uiStatsSource.ApplyCounters(processor.Tracker().GetCounters())
		} else {
			uc.startStatsAggregator(uiCtx)
			uc.flushStatsFromCounters(ctx)
		}
	}
	if uc.uiBus == nil {
		uc.startJobsWatcher(uiCtx)
		uc.startInventoryWatcher(uiCtx)
	}

	if runnerStream != nil {
		uc.uiWaitForUser = uc.app.UIExperimental && !uc.NoUI
		uc.uiRunnerDone = make(chan struct{})
		go func(done chan struct{}) {
			defer close(done)
			cfg := runner.Config{
				Mode:          uc.app.UIMode,
				Experimental:  uc.app.UIExperimental,
				LegacyEnabled: uc.app.UILegacy,
				ServerURL:     uc.client.Server,
				UserEmail:     uc.client.User.Email,
			}
			if err := runner.Run(uiCtx, cfg, runnerStream); err != nil && !errors.Is(err, runner.ErrNoShellSelected) && !errors.Is(err, context.Canceled) {
				uc.app.Log().Debug("ui runner exited", "err", err)
			}
		}(uc.uiRunnerDone)
	}
}

func (uc *UpCmd) setupUILogHandler(ctx context.Context) {
	// Register UI sink with FileProcessor to capture file events
	if processor := uc.app.FileProcessor(); processor != nil {
		uc.uiSink = newUISink(ctx, uc.uiPublisher)
		processor.Logger().RegisterSink(uc.uiSink)
	}

	// Also capture general app logs via the old writer approach
	// (Could be migrated to sink pattern later if needed)
	if uc.app.Log() != nil && uc.app.Log().Logger != nil {
		logWriter := &uiLogWriter{
			ctx:       ctx,
			publisher: uc.uiPublisher,
		}
		uc.app.Log().SetLogWriter(logWriter)
	}
}

func (uc *UpCmd) cleanupUILogHandler() {
	// Unregister UI sink
	if processor := uc.app.FileProcessor(); processor != nil && uc.uiSink != nil {
		processor.Logger().UnregisterSink(uc.uiSink)
		uc.uiSink = nil
	}

	// Clean up the main logger writer
	if uc.app.Log() != nil {
		uc.app.Log().SetLogWriter(nil)
	}
}

func (uc *UpCmd) shutdownUIPipeline(ctx context.Context) {
	if processor := uc.app.FileProcessor(); processor != nil {
		processor.SetCountersHook(nil)
		processor.SetEventHook(nil)
	}
	uc.stopStatsAggregator()
	uc.stopJobsWatcher()
	uc.stopInventoryWatcher()
	uc.cleanupUILogHandler()
	if uc.uiBus != nil {
		uc.uiBus.Close()
		uc.uiBus = nil
	} else if uc.uiPublisher != nil {
		uc.uiPublisher.Close()
	}
	if uc.uiRunnerCancel != nil {
		if !uc.uiWaitForUser || ctx == nil || ctx.Err() != nil {
			uc.uiRunnerCancel()
		}
		uc.uiRunnerCancel = nil
	}
	if uc.uiRunnerDone != nil {
		<-uc.uiRunnerDone
		uc.uiRunnerDone = nil
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

// captureTestStats reads stats events from a channel and records them in a test StatsCapture.
func captureTestStats(ctx context.Context, events <-chan messages.Event, capture interface{}) {
	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-events:
			if !ok {
				return
			}
			if evt.Type != messages.EventStatsUpdated {
				continue
			}
			if stats, ok := evt.Payload.(state.RunStats); ok {
				// Capture implements Record(state.RunStats)
				capture.(interface{ Record(state.RunStats) }).Record(stats)
			}
		}
	}
}

func (uc *UpCmd) publishAssetQueued(ctx context.Context, a *assets.Asset, code fileevent.Code) {
	event := uc.buildAssetEvent(a, state.AssetStageQueued, code, 0, "", nil)
	uc.uiPublisher.AssetQueued(ctx, event)
	if uc.uiStatsSource != nil {
		uc.uiStatsSource.AssetQueued()
		return
	}
	uc.updateStats(ctx, func(stats *state.RunStats) {
		stats.Queued++
	})
}

func (uc *UpCmd) publishAssetUploaded(ctx context.Context, a *assets.Asset, code fileevent.Code, bytes int64, details map[string]string) {
	event := uc.buildAssetEvent(a, state.AssetStageUploaded, code, bytes, "", details)
	uc.uiPublisher.AssetUploaded(ctx, event)
	if uc.uiStatsSource != nil {
		uc.uiStatsSource.AssetUploaded(bytes)
		return
	}
	uc.updateStats(ctx, func(stats *state.RunStats) {
		stats.Uploaded++
		stats.BytesSent += bytes
	})
}

func (uc *UpCmd) publishAssetFailed(ctx context.Context, a *assets.Asset, code fileevent.Code, reason error, details map[string]string) { //nolint:unparam
	msg := ""
	if reason != nil {
		msg = reason.Error()
	}
	event := uc.buildAssetEvent(a, state.AssetStageFailed, code, 0, msg, details)
	uc.uiPublisher.AssetFailed(ctx, event)
	if uc.uiStatsSource != nil {
		uc.uiStatsSource.AssetFailed()
		return
	}
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
	return state.CloneRunStats(uc.uiStats)
}

func (uc *UpCmd) updateStats(ctx context.Context, mutate func(*state.RunStats)) {
	if mutate == nil {
		return
	}
	uc.uiStatsMu.Lock()
	mutate(&uc.uiStats)
	now := time.Now()
	if uc.uiStats.Stage == "" {
		uc.uiStats.Stage = state.StageRunning
	}
	uc.uiStats.HasErrors = (uc.uiStats.Failed > 0) || (uc.uiStats.ErrorCount > 0)
	uc.uiStats.LastUpdated = now
	uc.updateThroughputLocked(now)
	snapshot := state.CloneRunStats(uc.uiStats)
	uc.uiStatsMu.Unlock()
	uc.uiPublisher.UpdateStats(ctx, snapshot)
}

func (uc *UpCmd) applyCountersSnapshot(ctx context.Context, counters assettracker.AssetCounters) {
	if uc.uiStatsSource != nil {
		uc.uiStatsSource.ApplyCounters(counters)
		return
	}
	uc.updateStats(ctx, func(stats *state.RunStats) {
		stats.Pending = int(counters.Pending)
		stats.PendingBytes = counters.PendingSize
		if v := int(counters.Processed); v > stats.Processed {
			stats.Processed = v
		}
		if v := counters.ProcessedSize; v > stats.ProcessedBytes {
			stats.ProcessedBytes = v
		}
		if v := int(counters.Discarded); v > stats.Discarded {
			stats.Discarded = v
		}
		if v := counters.DiscardedSize; v > stats.DiscardedBytes {
			stats.DiscardedBytes = v
		}
		if v := int(counters.Errors); v > stats.ErrorCount {
			stats.ErrorCount = v
		}
		if v := counters.ErrorSize; v > stats.ErrorBytes {
			stats.ErrorBytes = v
		}
		if v := int(counters.Total()); v > stats.TotalDiscovered {
			stats.TotalDiscovered = v
		}
		if v := counters.AssetSize; v > stats.TotalDiscoveredBytes {
			stats.TotalDiscoveredBytes = v
		}
		stats.InFlight = stats.Pending
	})
}

const (
	statsAggregationInterval    = 200 * time.Millisecond
	throughputSampleMinInterval = 200 * time.Millisecond
	maxThroughputSamples        = 64
)

func (uc *UpCmd) updateThroughputLocked(now time.Time) {
	if uc.uiStatsPrevSample.IsZero() {
		uc.uiStatsPrevSample = now
	}
	deltaBytes := uc.uiStats.BytesSent - uc.uiStatsPrevBytes
	if deltaBytes <= 0 {
		return
	}
	interval := now.Sub(uc.uiStatsPrevSample)
	if interval < throughputSampleMinInterval {
		return
	}
	if interval <= 0 {
		interval = throughputSampleMinInterval
	}
	bytesPerSecond := float64(deltaBytes) / interval.Seconds()
	sample := state.ThroughputSample{
		Timestamp:      now,
		BytesPerSecond: bytesPerSecond,
	}
	uc.uiStats.ThroughputSamples = append(uc.uiStats.ThroughputSamples, sample)
	if len(uc.uiStats.ThroughputSamples) > maxThroughputSamples {
		start := len(uc.uiStats.ThroughputSamples) - maxThroughputSamples
		uc.uiStats.ThroughputSamples = append([]state.ThroughputSample(nil), uc.uiStats.ThroughputSamples[start:]...)
	}
	uc.uiStatsPrevSample = now
	uc.uiStatsPrevBytes = uc.uiStats.BytesSent
}

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

func (uc *UpCmd) startInventoryWatcher(ctx context.Context) {
	uc.stopInventoryWatcher()
	if uc.client.Immich == nil || uc.uiPublisher == nil {
		return
	}
	interval := uc.app.UIInventoryPollInterval
	cfg := runner.InventoryWatcherConfig{
		Client:    uc.client.Immich,
		Publisher: uc.uiPublisher,
		Interval:  interval,
	}
	if log := uc.app.Log(); log != nil {
		cfg.Logger = log.Logger
	}
	uc.uiInventoryCancel = runner.StartInventoryWatcher(ctx, cfg)
}

func (uc *UpCmd) stopInventoryWatcher() {
	if uc.uiInventoryCancel != nil {
		uc.uiInventoryCancel()
		uc.uiInventoryCancel = nil
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
	if a.Type == filetypes.TypeVideo {
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

func logUIEvents(ctx context.Context, events <-chan messages.Event, logger *slog.Logger) {
	if logger == nil {
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-events:
			if !ok {
				return
			}
			logger.DebugContext(ctx, "ui-event", "type", evt.Type, "summary", summarizeEventPayload(evt.Payload))
		}
	}
}

func summarizeEventPayload(payload any) string {
	switch v := payload.(type) {
	case state.RunStats:
		return fmt.Sprintf("stats queued=%d uploaded=%d failed=%d", v.Queued, v.Uploaded, v.Failed)
	case state.LogEvent:
		return fmt.Sprintf("log[%s]: %s", v.Level, v.Message)
	case []state.JobSummary:
		if len(v) == 0 {
			return "jobs=0"
		}
		return fmt.Sprintf("jobs=%d first=%s", len(v), v[0].Name)
	default:
		return fmt.Sprintf("payload=%T", payload)
	}
}

// uiLogWriter implements io.Writer to forward slog output to UI events
type uiLogWriter struct {
	ctx       context.Context
	publisher messages.Publisher
}

func (w *uiLogWriter) Write(p []byte) (n int, err error) {
	if w.publisher == nil {
		return len(p), nil
	}
	// Parse the log line - format is typically: "YYYY-MM-DD HH:MM:SS LEVEL message"
	line := strings.TrimSpace(string(p))
	if line == "" {
		return len(p), nil
	}

	// Extract level and message
	parts := strings.SplitN(line, " ", 4)
	level := "info"
	message := line

	if len(parts) >= 4 {
		// Format: "DATE TIME LEVEL message"
		levelStr := strings.ToLower(strings.TrimSpace(parts[2]))
		switch {
		case strings.HasPrefix(levelStr, "err"):
			level = "error"
		case strings.HasPrefix(levelStr, "warn"):
			level = "warn"
		case strings.HasPrefix(levelStr, "info"):
			level = "info"
		case strings.HasPrefix(levelStr, "debug"):
			level = "debug"
		}
		message = parts[3]
	}

	w.publisher.AppendLog(w.ctx, state.LogEvent{
		Level:     level,
		Message:   message,
		Timestamp: time.Now(),
		Details:   nil,
	})

	return len(p), nil
}
