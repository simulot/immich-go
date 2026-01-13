# Event Bus Implementation Plan

## Step 1: Add Event Bus Core Types

**File**: `internal/fileevent/fileevents.go`

**Changes**:
- Add `Event` struct with fields: `Code`, `Timestamp`, `File`, `Size`, `Metadata map[string]any`
- Add `Subscription` interface with `Receive() <-chan Event` and `Close()`
- Add `Bus` struct with:
  - `Subscribe(codes ...Code) Subscription`
  - `Publish(event Event)` (non-blocking)
  - Buffered channel delivery (capacity 1000)
  - Subscriber management (add/remove)

**Testable**:
- Create bus, subscribe, publish event, verify receipt
- Multiple subscribers receive independent copies
- Filter by event code works correctly
- Non-blocking publish (use `select` with `default`)
- Graceful shutdown closes all subscriber channels

**Mergeable**: Yes, no breaking changes. Bus is optional.

---

## Step 2: Extend Recorder with Optional Bus

**File**: `internal/fileevent/fileevents.go`

**Changes**:
- Add `bus *Bus` field to `Recorder`
- Add `NewRecorderWithBus(logger *slog.Logger, bus *Bus) *Recorder`
- Update `RecordWithSize()`:
  - After atomic counter increment
  - If `bus != nil`, call `bus.Publish()`
- Add `Bus() *Bus` accessor

**Testable**:
- Recorder without bus works unchanged
- Recorder with bus publishes events correctly
- Backward compatibility: existing code unaffected

**Mergeable**: Yes, existing API unchanged, bus is opt-in.

---

## Step 3: Test Event Bus Under Load

**File**: `internal/fileevent/fileevents_test.go`

**Changes**:
- Test burst handling: 10,000 events published in tight loop
- Verify no dropped events when subscriber keeps up
- Verify oldest events dropped when buffer overflows
- Test concurrent subscribers (3+ goroutines)
- Test subscribe/unsubscribe during event stream
- Benchmark: `BenchmarkBusPublish`, `BenchmarkBusSubscribe`

**Testable**: Tests are the deliverable.

**Mergeable**: Yes, tests only.

---

## Step 4: Create FileProcessor with Event Bus

**File**: `internal/fileprocessor/fileprocessor.go`

**Changes**:
- Add `NewFileProcessorWithBus(bus *fileevent.Bus) *FileProcessor`
- Constructor creates `Recorder` with bus attached
- Keep existing `NewFileProcessor()` for compatibility

**File**: `internal/fileprocessor/fileprocessor_test.go`

**Changes**:
- Test events emitted when `RecordAssetDiscovered()` called
- Test events emitted when `RecordAssetProcessed()` called
- Verify `AssetTracker` and event bus stay in sync

**Testable**: Unit tests verify events emitted correctly.

**Mergeable**: Yes, opt-in via constructor.

---

## Step 5: Replace UI Polling with Event Subscription

**File**: `app/upload/ui.go`

**Changes**:
- Remove ticker-based polling loop (lines 145-174)
- Subscribe to discovery events: `DiscoveredImage`, `DiscoveredVideo`, etc.
- Subscribe to processing events: `ProcessedUploadSuccess`, etc.
- Update counters on event receipt in goroutine
- Use `GetCounts()` only for initial UI render
- Remove `highJackLogger()` method (lines 60-67)

**Testable**:
- Manual: Run upload with UI, verify counts update in real-time
- Manual: Monitor CPU usage during idle periods (should be near zero)
- Compare old vs new: `time ./immich-go upload ...` (expect slight improvement)

**Mergeable**: Yes, UI code only. No adapter changes.

---

## Step 6: Introduce Batched Log Consumer

**File**: `app/log.go`

**Changes**:
- Add `BatchedLogConsumer` struct
- Subscribe to all event codes
- Buffer events for 250ms or 100 events (whichever first)
- Flush batch to file/console in single write
- Replace direct `slog.Handler` writes

**File**: `app/log_test.go`

**Changes**:
- Test batch flushing on timer
- Test batch flushing on count threshold
- Test graceful shutdown flushes pending events

**Testable**:
- Measure log I/O syscalls before/after (use `strace -c`)
- Verify log output identical to current behavior

**Mergeable**: Yes, logging implementation detail.

---

## Step 7: Emit AssetTracker State Changes as Events

**File**: `internal/assettracker/tracker.go`

**Changes**:
- Add `bus *fileevent.Bus` field
- Add `NewAssetTrackerWithBus(bus *Bus) *AssetTracker`
- Update `SetProcessed()`, `SetDiscarded()`, `SetError()`:
  - After state change, publish `AssetStateChanged` event
- Keep existing `NewAssetTracker()` for compatibility

**File**: `internal/fileevent/fileevents.go`

**Changes**:
- Add event codes:
  - `AssetStateChangedProcessed`
  - `AssetStateChangedDiscarded`
  - `AssetStateChangedError`

**File**: `app/upload/ui.go`

**Changes**:
- Subscribe to `AssetStateChanged*` events
- Update status zone on event receipt
- Remove polling of `GetPendingCount()`, `GetProcessedCount()`

**Testable**:
- Unit test: state change emits correct event
- Integration test: UI status zone updates correctly

**Mergeable**: Yes, opt-in via constructor.

---

## Step 8: Clean Up Legacy Polling Code

**Files**: `app/upload/ui.go`, `app/upload/noui.go`

**Changes**:
- Remove unused ticker code
- Remove `GetCounts()` polling fallbacks
- Consolidate duplicate progress logic between `ui.go` and `noui.go`
- Consider extracting shared progress formatter

**Testable**:
- All upload commands still work
- No regressions in output formatting

**Mergeable**: Yes, cleanup only.

---

## Step 9: Document Event Bus API

**File**: `docs/technical.md`

**Changes**:
- Add "Event Bus Architecture" section
- Document event types, subscription API, filtering
- Document guarantees (eventual delivery, no ordering across subscribers)
- Document performance characteristics (buffering, drop policy)
- Provide code example: subscribing to file discovery events

**File**: `internal/fileevent/fileevents.go`

**Changes**:
- Add GoDoc comments for `Event`, `Subscription`, `Bus`
- Document non-blocking publish behavior
- Document buffer overflow policy

**Mergeable**: Yes, documentation only.

---

## Notes

- Each step is independently testable
- Each step can be merged to `develop` without breaking existing features
- Steps 1-4 add infrastructure without changing behavior
- Steps 5-7 migrate consumers to event bus
- Step 8 removes dead code
- Step 9 documents the new architecture

Total estimated PRs: **9 small PRs** vs 1 large refactor.
