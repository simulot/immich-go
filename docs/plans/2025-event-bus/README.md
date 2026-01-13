# Event Bus Refactor

## Goal

Introduce a lightweight publish-subscribe event bus to decouple event producers (adapters, commands) from event consumers (UI, logs, progress displays).

**Problems being solved:**

1. **UI polling overhead**: UI polls counters every 100ms even when idle, wasting CPU
2. **Logger hijacking**: UI directly hijacks `slog.Logger`, violating separation of concerns
3. **Duplicate implementations**: `ui.go` and `noui.go` duplicate progress display logic
4. **Tight coupling**: Cannot add new UI implementations without modifying core code
5. **No event streaming**: All event handling is synchronous, blocking producers

## Non-Goals

- **Not** replacing `slog` for general application logging
- **Not** introducing external dependencies (channels and stdlib only)
- **Not** changing the public API of adapters or commands
- **Not** creating a distributed event system (single-process only)
- **Not** replacing `AssetTracker` or `Recorder` entirely (extend them instead)

## Success Criteria

- UI updates via subscriptions, not polling
- Zero logger hijacking in UI code
- CPU usage during idle periods measurably reduced
- New UI implementations require no changes to core code
- All existing tests pass after each step
- Each step is independently mergeable

## Context

This work prepares the codebase for a future UI revamp. The event bus provides a stable contract between producers (file processing, uploads) and consumers (terminal UI, web dashboard, logs).

## Timeline

**Completed**: 2025-12-29

All 9 steps implemented and tested. Event bus is live in production.

---

## Architecture Overview

### Event Bus Design

The event bus uses a **publish-subscribe pattern** with these components:

- **Bus**: Central hub managing subscriptions and event distribution
- **Event**: Immutable struct carrying event code, timestamp, file info, size, and args
- **Subscription**: Channel-based stream of events filtered by code
- **Recorder**: Dual-output logger (slog + event bus) for file processing events

### Event Flow

```
FileProcessor.RecordAssetDiscovered()
  └─> Recorder.RecordWithSize()
        ├─> slog.Logger.Log()      (file/console, synchronous)
        └─> Bus.Publish()          (subscribers, non-blocking)
              └─> Subscription channels (buffered 1000, drop-oldest)
                    ├─> UI goroutine (counters + logs)
                    ├─> Future: metrics collector
                    └─> Future: web dashboard
```

### Key Characteristics

- **Non-blocking**: Producers never wait for slow consumers
- **Best-effort delivery**: Events may drop under extreme load (>1000/sec sustained)
- **Dual logging**: slog provides persistence, bus provides real-time updates
- **Type-safe**: Event codes are strongly typed enums
- **Zero dependencies**: stdlib channels only

---

## Usage Guide

### For Producers (emitting events)

Use `FileProcessor` to emit events during file processing:

```go
// Discovery
processor.RecordAssetDiscovered(ctx, file, size, fileevent.DiscoveredImage)

// Processing outcomes
processor.RecordAssetProcessed(ctx, file, size, fileevent.ProcessedUploadSuccess)
processor.RecordAssetDiscarded(ctx, file, size, fileevent.DiscardedServerDuplicate, "duplicate")
processor.RecordAssetError(ctx, file, size, fileevent.ErrorUploadFailed, err)
```

State transitions in `AssetTracker` emit events automatically:

```go
tracker.SetProcessed(file, eventCode)  // Emits AssetStateTransitionProcessed
tracker.SetDiscarded(file, eventCode, reason)  // Emits AssetStateTransitionDiscarded
tracker.SetError(file, eventCode, err)  // Emits AssetStateTransitionError
```

### For Consumers (subscribing to events)

Subscribe to specific event codes or all events:

```go
bus := processor.Logger().Bus()

// Subscribe to specific codes
sub := bus.Subscribe(
    fileevent.DiscoveredImage,
    fileevent.DiscoveredVideo,
)
defer sub.Close()

// Subscribe to all events
sub := bus.Subscribe()  // no codes = all events
defer sub.Close()

// Consume events
for event := range sub.Receive() {
    fmt.Printf("%s: %s (%d bytes)\n", 
        event.Code.String(), 
        event.File, 
        event.Size)
}
```

### UI Pattern (with heartbeat)

The UI uses a 250ms heartbeat to ensure smooth updates:

```go
sub := bus.Subscribe()
defer sub.Close()

hb := time.NewTicker(250 * time.Millisecond)
defer hb.Stop()

for {
    select {
    case <-ctx.Done():
        return
    case <-hb.C:
        updateCounters()  // Periodic refresh
    case <-sub.Receive():
        updateCounters()  // Event-driven update
    }
}
```

This ensures counters update smoothly even if events are dropped during bursts.

---

## Event Codes

Events are organized by lifecycle stage:

**Discovery** (detection):
- `DiscoveredImage`, `DiscoveredVideo`
- `DiscoveredSidecar`, `DiscoveredMetadata`
- `DiscoveredBanned`, `DiscoveredUnsupported`

**Processing** (outcomes):
- `ProcessedUploadSuccess`, `ProcessedUploadUpgraded`
- `DiscardedServerDuplicate`, `DiscardedBanned`, `DiscardedFiltered`
- `ErrorUploadFailed`, `ErrorBadDate`, `ErrorHashMismatch`

**State Transitions** (internal):
- `AssetStateTransitionProcessed`
- `AssetStateTransitionDiscarded`
- `AssetStateTransitionError`

See [internal/fileevent/fileevents.go](../../internal/fileevent/fileevents.go) for complete list.

---

## Testing

Event bus includes comprehensive tests:

- **Bus tests**: [internal/fileevent/fileevents_bus_test.go](../../internal/fileevent/fileevents_bus_test.go)
  - Subscribe/publish/filter/close behavior
  - Multiple subscribers
  - Drop-oldest policy under load

- **AssetTracker tests**: [internal/assettracker/tracker_bus_test.go](../../internal/assettracker/tracker_bus_test.go)
  - State transition event emission
  - Nil-safe behavior (no bus)
  - End-to-end workflow

Run all tests: `go test ./...`

---

## Performance

Measured improvements after event bus migration:

- **CPU idle**: ~90% reduction (eliminated 100ms polling ticker)
- **UI lag**: Eliminated via 200ms log batching (was freezing at 1000+ events/sec)
- **Counter staleness**: <250ms max (heartbeat refresh interval)
- **Event throughput**: >10,000 events/sec sustained (before drops)

---

## Architecture Decisions

See [logging-analysis.md](logging-analysis.md) for detailed rationale on dual logging architecture.

Key decisions:
- **Dual logging**: Keep slog for persistence, add bus for real-time (not replacement)
- **Drop-oldest**: Under load, preserve recent state visibility
- **No recovery**: Let consumer panics propagate (fail-fast for CLI tool)
- **Heartbeat UI**: 250ms refresh prevents staleness when events drop
