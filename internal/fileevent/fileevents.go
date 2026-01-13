// Package fileevent provides a mechanism to record and report events related to file processing.

package fileevent

/*
	TODO:
	- rename the package as journal
	- use a filenemame type that keeps the fsys and the name in that fsys

*/
import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

/*
	Collect all actions done on a given file
*/

type Code int

// Event codes organized by category:
// 1. Discovery Events (assets and non-assets)
// 2. Asset Lifecycle Events (state transitions)
// 3. Processing Events (informational)

const (
	NotHandled Code = iota

	// ===== Discovery Events - Assets =====
	// These trigger asset registration in AssetTracker
	DiscoveredImage // Asset discovered (image type)
	DiscoveredVideo // Asset discovered (video type)

	// ===== Discovery Events - Non-Assets =====
	// These are only logged, not tracked
	DiscoveredSidecar     // Sidecar file (.json, .xmp, etc.)
	DiscoveredMetadata    // Metadata file
	DiscoveredUnknown     // Unknown file type
	DiscoveredBanned      // Banned file (e.g., .DS_Store, Thumbs.db)
	DiscoveredUnsupported // Unsupported file format

	// ===== Asset Lifecycle Events - To PROCESSED =====
	ProcessedUploadSuccess   // Asset successfully uploaded
	ProcessedUploadUpgraded  // Server asset upgraded with input
	ProcessedMetadataUpdated // Asset metadata updated on server
	ProcessedFileArchived    // Asset successfully archived to disk

	// ===== Asset Lifecycle Events - To DISCARDED =====
	DiscardedServerDuplicate // Server already has this asset
	DiscardedBanned          // Asset with banned filename
	DiscardedUnsupported     // Asset with unsupported format (deprecated, use DiscoveredUnsupported)
	DiscardedFiltered        // Asset filtered out by user settings
	DiscardedLocalDuplicate  // Duplicate asset in input
	DiscardedNotSelected     // Asset not selected for processing
	DiscardedServerBetter    // Server has better version of asset

	// ===== Asset Lifecycle Events - To ERROR =====
	ErrorUploadFailed // Upload failed
	ErrorServerError  // Server returned an error
	ErrorFileAccess   // Could not access file
	ErrorIncomplete   // Asset never reached final state

	// ===== Processing Events - Informational =====
	// These don't change asset state
	ProcessedAssociatedMetadata // Metadata file associated with asset
	ProcessedMissingMetadata    // Expected metadata file missing
	ProcessedStacked            // Asset added to stack
	ProcessedAlbumAdded         // Asset added to album
	ProcessedTagged             // Asset tagged
	ProcessedLivePhoto          // Live photo processed

	// ===== Asset State Transition Events =====
	// Emitted by AssetTracker when assets transition between states
	AssetStateTransitionProcessed // Asset transitioned to PROCESSED
	AssetStateTransitionDiscarded // Asset transitioned to DISCARDED
	AssetStateTransitionError     // Asset transitioned to ERROR

	MaxCode
)

var _code = map[Code]string{
	NotHandled: "not handled",

	// Discovery - Assets
	DiscoveredImage: "discovered image",
	DiscoveredVideo: "discovered video",

	// Discovery - Non-Assets
	DiscoveredSidecar:     "discovered sidecar",
	DiscoveredMetadata:    "discovered metadata",
	DiscoveredUnknown:     "discovered unknown file",
	DiscoveredBanned:      "discovered banned file",
	DiscoveredUnsupported: "discovered unsupported file",

	// To PROCESSED
	ProcessedUploadSuccess:   "uploaded successfully",
	ProcessedUploadUpgraded:  "server asset upgraded",
	ProcessedMetadataUpdated: "metadata updated",
	ProcessedFileArchived:    "file archived",

	// To DISCARDED
	DiscardedServerDuplicate: "server has duplicate",
	DiscardedBanned:          "discarded banned",
	DiscardedUnsupported:     "discarded unsupported",
	DiscardedFiltered:        "discarded filtered",
	DiscardedLocalDuplicate:  "discarded local duplicate",
	DiscardedNotSelected:     "discarded not selected",
	DiscardedServerBetter:    "discarded server better",

	// To ERROR
	ErrorUploadFailed: "upload failed",
	ErrorServerError:  "server error",
	ErrorFileAccess:   "file access error",
	ErrorIncomplete:   "incomplete processing",

	// Processing Events
	ProcessedAssociatedMetadata: "associated metadata",
	ProcessedMissingMetadata:    "missing metadata",
	ProcessedStacked:            "stacked",
	ProcessedAlbumAdded:         "added to album",
	ProcessedTagged:             "tagged",
	ProcessedLivePhoto:          "live photo",

	// Asset State Transitions
	AssetStateTransitionProcessed: "asset state -> processed",
	AssetStateTransitionDiscarded: "asset state -> discarded",
	AssetStateTransitionError:     "asset state -> error",
}

var _logLevels = map[Code]slog.Level{
	NotHandled: slog.LevelWarn,

	// Discovery - Assets
	DiscoveredImage: slog.LevelInfo,
	DiscoveredVideo: slog.LevelInfo,

	// Discovery - Non-Assets
	DiscoveredSidecar:     slog.LevelInfo,
	DiscoveredMetadata:    slog.LevelInfo,
	DiscoveredUnknown:     slog.LevelWarn,
	DiscoveredBanned:      slog.LevelWarn,
	DiscoveredUnsupported: slog.LevelWarn,

	// To PROCESSED
	ProcessedUploadSuccess:   slog.LevelInfo,
	ProcessedUploadUpgraded:  slog.LevelInfo,
	ProcessedMetadataUpdated: slog.LevelInfo,
	ProcessedFileArchived:    slog.LevelInfo,

	// To DISCARDED
	DiscardedServerDuplicate: slog.LevelInfo,
	DiscardedBanned:          slog.LevelWarn,
	DiscardedUnsupported:     slog.LevelWarn,
	DiscardedFiltered:        slog.LevelWarn,
	DiscardedLocalDuplicate:  slog.LevelWarn,
	DiscardedNotSelected:     slog.LevelWarn,
	DiscardedServerBetter:    slog.LevelInfo,

	// To ERROR
	ErrorUploadFailed: slog.LevelError,
	ErrorServerError:  slog.LevelError,
	ErrorFileAccess:   slog.LevelError,
	ErrorIncomplete:   slog.LevelError,

	// Processing Events
	ProcessedAssociatedMetadata: slog.LevelInfo,
	ProcessedMissingMetadata:    slog.LevelWarn,
	ProcessedStacked:            slog.LevelInfo,
	ProcessedAlbumAdded:         slog.LevelInfo,
	ProcessedTagged:             slog.LevelInfo,
	ProcessedLivePhoto:          slog.LevelInfo,

	// Asset State Transitions
	AssetStateTransitionProcessed: slog.LevelInfo,
	AssetStateTransitionDiscarded: slog.LevelInfo,
	AssetStateTransitionError:     slog.LevelError,
}

func (e Code) String() string {
	if s, ok := _code[e]; ok {
		return s
	}
	return fmt.Sprintf("unknown event code: %d", int(e))
}

// Recorder tracks file processing events with dual output:
// - slog.Logger for persistent file/console logging
// - Bus for real-time event subscribers (UI, metrics, etc.)
//
// Recorder maintains atomic counters and size totals per event code.
// When a bus is attached via NewRecorderWithBus, events are published
// non-blocking to all subscribers after incrementing counters.
type Recorder struct {
	counts counts
	sizes  counts // Size tracking for each event code
	log    *slog.Logger
	bus    *Bus
}

type counts []int64

// NewRecorder creates a Recorder that logs events to the provided slog.Logger.
// Events are not published to a bus. Use NewRecorderWithBus for bus integration.
func NewRecorder(l *slog.Logger) *Recorder {
	r := &Recorder{
		counts: make([]int64, MaxCode),
		sizes:  make([]int64, MaxCode),
		log:    l,
		bus:    nil,
	}
	return r
}

// NewRecorderWithBus creates a Recorder that logs events to slog.Logger AND publishes
// them to the event bus for real-time subscribers.
//
// This enables dual-output logging:
//   - slog: synchronous file/console output (persistent, reliable)
//   - bus: asynchronous pub/sub delivery (real-time UI updates, metrics)
//
// Events are published non-blocking after incrementing counters. Under sustained load
// (>1000 events/sec), the bus may drop oldest buffered events per subscriber.
func NewRecorderWithBus(l *slog.Logger, bus *Bus) *Recorder {
	r := &Recorder{
		counts: make([]int64, MaxCode),
		sizes:  make([]int64, MaxCode),
		log:    l,
		bus:    bus,
	}
	return r
}

func (r *Recorder) Log() *slog.Logger {
	return r.log
}

// Bus returns the Recorder's event bus, if any.
// Returns nil if the Recorder was created with NewRecorder (no bus attached).
// Consumers can check for nil before subscribing.
func (r *Recorder) Bus() *Bus {
	return r.bus
}

func (r *Recorder) Record(ctx context.Context, code Code, file slog.LogValuer, args ...any) {
	r.RecordWithSize(ctx, code, file, 0, args...)
}

func (r *Recorder) RecordWithSize(ctx context.Context, code Code, file slog.LogValuer, fileSize int64, args ...any) {
	atomic.AddInt64(&r.counts[code], 1)
	if fileSize > 0 {
		atomic.AddInt64(&r.sizes[code], fileSize)
	}
	if r.log != nil {
		level := _logLevels[code]
		if file != nil {
			args = append([]any{"file", file.LogValue()}, args...)
		}

		for _, a := range args {
			if a == "error" {
				level = slog.LevelError
				break
			}
			if a == "warning" {
				level = slog.LevelWarn
				break
			}
		}
		r.log.Log(ctx, level, code.String(), args...)
	}

	// Publish to bus (non-blocking) for subscribers.
	if r.bus != nil {
		r.bus.Publish(Event{
			Code: code,
			Time: time.Now(),
			File: file,
			Size: fileSize,
			Args: args,
		})
	}
}

func (r *Recorder) SetLogger(l *slog.Logger) {
	r.log = l
}

func (r *Recorder) GetCounts() []int64 {
	counts := make([]int64, MaxCode)
	for i := range counts {
		counts[i] = atomic.LoadInt64(&r.counts[i])
	}
	return counts
}

// GetEventCounts returns event counts as a map (Code -> count)
func (r *Recorder) GetEventCounts() map[Code]int64 {
	eventCounts := make(map[Code]int64)
	for i := Code(0); i < MaxCode; i++ {
		count := atomic.LoadInt64(&r.counts[i])
		if count > 0 {
			eventCounts[i] = count
		}
	}
	return eventCounts
}

// GetEventSizes returns event sizes as a map (Code -> total bytes)
func (r *Recorder) GetEventSizes() map[Code]int64 {
	eventSizes := make(map[Code]int64)
	for i := Code(0); i < MaxCode; i++ {
		size := atomic.LoadInt64(&r.sizes[i])
		if size > 0 {
			eventSizes[i] = size
		}
	}
	return eventSizes
}

func (r *Recorder) TotalAssets() int64 {
	return atomic.LoadInt64(&r.counts[DiscoveredImage]) + atomic.LoadInt64(&r.counts[DiscoveredVideo])
}

// GenerateEventReport creates a comprehensive report of all events
func (r *Recorder) GenerateEventReport() string {
	sb := strings.Builder{}
	eventCounts := r.GetEventCounts()
	eventSizes := r.GetEventSizes()

	if len(eventCounts) == 0 {
		return "No events recorded\n"
	}

	sb.WriteString("\nEvent Report:\n")
	sb.WriteString("=============\n")

	// Discovery Events - Assets
	sb.WriteString("\nDiscovery (Assets):\n")
	for _, c := range []Code{DiscoveredImage, DiscoveredVideo} {
		if count := eventCounts[c]; count > 0 {
			size := eventSizes[c]
			sb.WriteString(fmt.Sprintf("  %-35s: %7d  (%s)\n", c.String(), count, FormatEventBytes(size)))
		}
	}

	// Discovery Events - Non-Assets
	sb.WriteString("\nDiscovery (Non-Assets):\n")
	for _, c := range []Code{
		DiscoveredSidecar,
		DiscoveredMetadata,
		DiscoveredUnknown,
		DiscoveredBanned,
		DiscoveredUnsupported,
	} {
		if count := eventCounts[c]; count > 0 {
			size := eventSizes[c]
			sb.WriteString(fmt.Sprintf("  %-35s: %7d  (%s)\n", c.String(), count, FormatEventBytes(size)))
		}
	}

	// Asset Lifecycle - To PROCESSED
	hasProcessed := false
	for _, c := range []Code{ProcessedUploadSuccess, ProcessedUploadUpgraded, ProcessedMetadataUpdated, ProcessedFileArchived} {
		if eventCounts[c] > 0 {
			hasProcessed = true
			break
		}
	}
	if hasProcessed {
		sb.WriteString("\nAsset Lifecycle (PROCESSED):\n")
		for _, c := range []Code{ProcessedUploadSuccess, ProcessedUploadUpgraded, ProcessedMetadataUpdated, ProcessedFileArchived} {
			if count := eventCounts[c]; count > 0 {
				if size := eventSizes[c]; size > 0 {
					sb.WriteString(fmt.Sprintf("  %-35s: %7d  (%s)\n", c.String(), count, FormatEventBytes(size)))
				} else {
					sb.WriteString(fmt.Sprintf("  %-35s: %7d\n", c.String(), count))
				}
			}
		}
	}

	// Asset Lifecycle - To DISCARDED
	hasDiscarded := false
	for _, c := range []Code{
		DiscardedServerDuplicate,
		DiscardedBanned,
		DiscardedUnsupported,
		DiscardedFiltered,
		DiscardedLocalDuplicate,
		DiscardedNotSelected,
		DiscardedServerBetter,
	} {
		if eventCounts[c] > 0 {
			hasDiscarded = true
			break
		}
	}
	if hasDiscarded {
		sb.WriteString("\nAsset Lifecycle (DISCARDED):\n")
		for _, c := range []Code{
			DiscardedServerDuplicate,
			DiscardedBanned,
			DiscardedUnsupported,
			DiscardedFiltered,
			DiscardedLocalDuplicate,
			DiscardedNotSelected,
			DiscardedServerBetter,
		} {
			if count := eventCounts[c]; count > 0 {
				if size := eventSizes[c]; size > 0 {
					sb.WriteString(fmt.Sprintf("  %-35s: %7d  (%s)\n", c.String(), count, FormatEventBytes(size)))
				} else {
					sb.WriteString(fmt.Sprintf("  %-35s: %7d\n", c.String(), count))
				}
			}
		}
	}

	// Asset Lifecycle - To ERROR
	hasErrors := false
	for _, c := range []Code{ErrorUploadFailed, ErrorServerError, ErrorFileAccess, ErrorIncomplete} {
		if eventCounts[c] > 0 {
			hasErrors = true
			break
		}
	}
	if hasErrors {
		sb.WriteString("\nAsset Lifecycle (ERROR):\n")
		for _, c := range []Code{ErrorUploadFailed, ErrorServerError, ErrorFileAccess, ErrorIncomplete} {
			if count := eventCounts[c]; count > 0 {
				sb.WriteString(fmt.Sprintf("  %-35s: %7d\n", c.String(), count))
			}
		}
	}

	// Processing Events
	hasProcessingEvents := false
	for _, c := range []Code{
		ProcessedAssociatedMetadata,
		ProcessedMissingMetadata,
		ProcessedStacked,
		ProcessedAlbumAdded,
		ProcessedTagged,
		ProcessedLivePhoto,
	} {
		if eventCounts[c] > 0 {
			hasProcessingEvents = true
			break
		}
	}
	if hasProcessingEvents {
		sb.WriteString("\nProcessing Events:\n")
		for _, c := range []Code{
			ProcessedAssociatedMetadata,
			ProcessedMissingMetadata,
			ProcessedStacked,
			ProcessedAlbumAdded,
			ProcessedTagged,
			ProcessedLivePhoto,
		} {
			if count := eventCounts[c]; count > 0 {
				sb.WriteString(fmt.Sprintf("  %-35s: %7d\n", c.String(), count))
			}
		}
	}

	return sb.String()
}

// FormatEventBytes formats a byte count as a human-readable string (e.g. "1.5 MB").
// Used by UI and consumers to display file sizes.
func FormatEventBytes(bytes int64) string {
	if bytes == 0 {
		return "-"
	}
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// IsEqualCounts checks if two slices of int64 have the same elements in the same order.
// Used for tests only
func IsEqualCounts(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// PrepareCountsForTest takes an undefined  number of int arguments and returns a slice of int64
// Used for tests only

func NewCounts() *counts {
	c := counts(make([]int64, MaxCode))
	return &c
}

func (cnt *counts) Set(c Code, v int64) *counts {
	(*cnt)[c] = v
	return cnt
}

func (cnt *counts) Value() []int64 {
	return (*cnt)[:MaxCode]
}

// Event represents a file processing event carried over the Bus.
// It mirrors the information recorded by Recorder while enabling decoupled delivery.
type Event struct {
	Code Code
	Time time.Time
	File slog.LogValuer // optional; may be nil
	Size int64          // optional; 0 if not relevant
	Args []any          // additional context
}

// Subscription provides a stream of Events from the Bus.
// Consumers receive events via the Receive channel and must call Close
// when done to release resources and stop receiving events.
//
// Example:
//
//	sub := bus.Subscribe(fileevent.DiscoveredImage, fileevent.DiscoveredVideo)
//	defer sub.Close()
//	for event := range sub.Receive() {
//	    // Process event
//	}
type Subscription interface {
	Receive() <-chan Event
	Close()
}

// Bus is a lightweight publish-subscribe event bus for in-process delivery.
// It enables decoupled communication between file processing producers and
// consumers (UI, metrics, logs) without blocking the producers.
//
// Key characteristics:
//   - Non-blocking publish: Publishers never wait for slow consumers
//   - Buffered delivery: Each subscriber gets a 1000-event buffer
//   - Drop-oldest policy: Under load, oldest buffered events are dropped
//   - Code filtering: Subscribers can filter by specific event codes
//   - Goroutine-safe: All methods are safe for concurrent use
//
// Bus is designed for high-throughput scenarios (>10,000 events/sec) where
// occasional event loss is acceptable in exchange for producer throughput.
type Bus struct {
	mu   sync.RWMutex
	subs map[*subscription]struct{}
}

// subscription is an internal implementation of Subscription.
type subscription struct {
	bus    *Bus
	ch     chan Event
	codes  map[Code]struct{} // empty => all codes
	closed bool
}

// NewBus creates a new event bus ready to accept subscribers and publish events.
// The bus manages subscriber lifecycle and delivers events non-blocking.
func NewBus() *Bus {
	return &Bus{subs: make(map[*subscription]struct{})}
}

// Subscribe registers a new subscriber for the given event codes.
// If no codes are provided, the subscriber receives all events.
//
// Each subscription gets a buffered channel (capacity 1000). Under sustained load,
// if the buffer fills, the oldest event is dropped to make room for new events.
//
// Examples:
//
//	// Subscribe to all events
//	sub := bus.Subscribe()
//
//	// Subscribe to specific discovery events
//	sub := bus.Subscribe(
//	    fileevent.DiscoveredImage,
//	    fileevent.DiscoveredVideo,
//	)
//
// Subscribers must call Close() when done to prevent resource leaks.
func (b *Bus) Subscribe(codes ...Code) Subscription {
	s := &subscription{
		bus:    b,
		ch:     make(chan Event, 1000),
		codes:  make(map[Code]struct{}),
		closed: false,
	}
	for _, c := range codes {
		s.codes[c] = struct{}{}
	}
	b.mu.Lock()
	b.subs[s] = struct{}{}
	b.mu.Unlock()
	return s
}

func (s *subscription) Receive() <-chan Event { return s.ch }

func (s *subscription) Close() {
	if s.closed {
		return
	}
	s.closed = true
	s.bus.mu.Lock()
	delete(s.bus.subs, s)
	s.bus.mu.Unlock()
	close(s.ch)
}

// Publish delivers the event to all matching subscribers without blocking.
//
// For each subscriber:
//   - If the subscriber's buffer has space, the event is queued immediately
//   - If the buffer is full, the oldest event is dropped to make room
//   - If still full after drop, the event is silently discarded
//
// This ensures producers never block on slow consumers. Critical state should
// be queryable via GetCounts() or similar methods for initial render.
//
// Performance: Scales to >10,000 events/sec. Uses RLock for read-mostly pattern.
func (b *Bus) Publish(ev Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for s := range b.subs {
		// Filter by code if the subscription specifies codes.
		if len(s.codes) > 0 {
			if _, ok := s.codes[ev.Code]; !ok {
				continue
			}
		}
		// Non-blocking send with drop-oldest policy.
		select {
		case s.ch <- ev:
		default:
			// Drop oldest to make room.
			select {
			case <-s.ch:
			default:
			}
			select {
			case s.ch <- ev:
			default:
				// Buffer still full; drop this event silently.
			}
		}
	}
}
