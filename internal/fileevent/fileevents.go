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
	"sync/atomic"
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
	UploadedSuccess  // Asset successfully uploaded
	UploadedUpgraded // Server asset upgraded with input

	// ===== Asset Lifecycle Events - To DISCARDED =====
	UploadedServerDuplicate // Server already has this asset
	DiscardedBanned         // Asset with banned filename
	DiscardedUnsupported    // Asset with unsupported format (deprecated, use DiscoveredUnsupported)
	DiscardedFiltered       // Asset filtered out by user settings
	DiscardedLocalDuplicate // Duplicate asset in input
	DiscardedNotSelected    // Asset not selected for processing
	DiscardedServerBetter   // Server has better version of asset

	// ===== Asset Lifecycle Events - To ERROR =====
	ErrorUploadFailed // Upload failed
	ErrorServerError  // Server returned an error
	ErrorFileAccess   // Could not access file
	ErrorIncomplete   // Asset never reached final state

	// ===== Processing Events - Informational =====
	// These don't change asset state
	AnalysisAssociatedMetadata        // Metadata file associated with asset
	AnalysisMissingAssociatedMetadata // Expected metadata file missing
	ProcessedStacked                  // Asset added to stack
	ProcessedAlbumAdded               // Asset added to album
	ProcessedTagged                   // Asset tagged
	ProcessedLivePhoto                // Live photo processed

	// ===== Legacy/Compatibility =====
	// Maintained for backward compatibility
	Uploaded               // Legacy: use UploadedSuccess
	UploadAlbumCreated     // Album created/updated
	UploadAddToAlbum       // Legacy: use ProcessedAlbumAdded
	AnalysisLocalDuplicate // Legacy: use DiscardedLocalDuplicate
	UploadNotSelected      // Legacy: use DiscardedNotSelected
	UploadServerBetter     // Legacy: use DiscardedServerBetter
	Stacked                // Legacy: use ProcessedStacked
	LivePhoto              // Legacy: use ProcessedLivePhoto
	Tagged                 // Legacy: use ProcessedTagged
	Metadata               // Legacy metadata marker
	INFO                   // Generic info message
	Written                // File written to disk
	DiscoveredDiscarded    // Legacy: use specific discard reasons
	DiscoveredUseless      // Legacy: use DiscoveredUnknown or DiscoveredBanned
	Error                  // Generic error

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
	UploadedSuccess:  "uploaded successfully",
	UploadedUpgraded: "server asset upgraded",

	// To DISCARDED
	UploadedServerDuplicate: "server has duplicate",
	DiscardedBanned:         "discarded banned",
	DiscardedUnsupported:    "discarded unsupported",
	DiscardedFiltered:       "discarded filtered",
	DiscardedLocalDuplicate: "discarded local duplicate",
	DiscardedNotSelected:    "discarded not selected",
	DiscardedServerBetter:   "discarded server better",

	// To ERROR
	ErrorUploadFailed: "upload failed",
	ErrorServerError:  "server error",
	ErrorFileAccess:   "file access error",
	ErrorIncomplete:   "incomplete processing",

	// Processing Events
	AnalysisAssociatedMetadata:        "associated metadata",
	AnalysisMissingAssociatedMetadata: "missing metadata",
	ProcessedStacked:                  "stacked",
	ProcessedAlbumAdded:               "added to album",
	ProcessedTagged:                   "tagged",
	ProcessedLivePhoto:                "live photo",

	// Legacy
	Uploaded:               "uploaded",
	UploadAlbumCreated:     "album created",
	UploadAddToAlbum:       "added to album",
	AnalysisLocalDuplicate: "local duplicate",
	UploadNotSelected:      "not selected",
	UploadServerBetter:     "server better",
	Stacked:                "stacked",
	LivePhoto:              "live photo",
	Tagged:                 "tagged",
	Metadata:               "metadata",
	INFO:                   "info",
	Written:                "written",
	DiscoveredDiscarded:    "discarded",
	DiscoveredUseless:      "useless file",
	Error:                  "error",
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
	UploadedSuccess:  slog.LevelInfo,
	UploadedUpgraded: slog.LevelInfo,

	// To DISCARDED
	UploadedServerDuplicate: slog.LevelInfo,
	DiscardedBanned:         slog.LevelWarn,
	DiscardedUnsupported:    slog.LevelWarn,
	DiscardedFiltered:       slog.LevelWarn,
	DiscardedLocalDuplicate: slog.LevelWarn,
	DiscardedNotSelected:    slog.LevelWarn,
	DiscardedServerBetter:   slog.LevelInfo,

	// To ERROR
	ErrorUploadFailed: slog.LevelError,
	ErrorServerError:  slog.LevelError,
	ErrorFileAccess:   slog.LevelError,
	ErrorIncomplete:   slog.LevelError,

	// Processing Events
	AnalysisAssociatedMetadata:        slog.LevelInfo,
	AnalysisMissingAssociatedMetadata: slog.LevelWarn,
	ProcessedStacked:                  slog.LevelInfo,
	ProcessedAlbumAdded:               slog.LevelInfo,
	ProcessedTagged:                   slog.LevelInfo,
	ProcessedLivePhoto:                slog.LevelInfo,

	// Legacy
	Uploaded:               slog.LevelInfo,
	UploadAlbumCreated:     slog.LevelInfo,
	UploadAddToAlbum:       slog.LevelInfo,
	AnalysisLocalDuplicate: slog.LevelWarn,
	UploadNotSelected:      slog.LevelWarn,
	UploadServerBetter:     slog.LevelInfo,
	Stacked:                slog.LevelInfo,
	LivePhoto:              slog.LevelInfo,
	Tagged:                 slog.LevelInfo,
	Metadata:               slog.LevelInfo,
	INFO:                   slog.LevelInfo,
	Written:                slog.LevelInfo,
	DiscoveredDiscarded:    slog.LevelWarn,
	DiscoveredUseless:      slog.LevelWarn,
	Error:                  slog.LevelError,
}

func (e Code) String() string {
	if s, ok := _code[e]; ok {
		return s
	}
	return fmt.Sprintf("unknown event code: %d", int(e))
}

type Recorder struct {
	counts counts
	sizes  counts // Size tracking for each event code
	log    *slog.Logger
}

type counts []int64

func NewRecorder(l *slog.Logger) *Recorder {
	r := &Recorder{
		counts: make([]int64, MaxCode),
		sizes:  make([]int64, MaxCode),
		log:    l,
	}
	return r
}

func (r *Recorder) Log() *slog.Logger {
	return r.log
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

func (r *Recorder) TotalProcessedGP() int64 {
	return atomic.LoadInt64(&r.counts[AnalysisAssociatedMetadata]) +
		atomic.LoadInt64(&r.counts[AnalysisMissingAssociatedMetadata]) +
		atomic.LoadInt64(&r.counts[DiscoveredDiscarded])
}

func (r *Recorder) TotalProcessed(forcedMissingJSON bool) int64 {
	v := atomic.LoadInt64(&r.counts[UploadedSuccess]) +
		atomic.LoadInt64(&r.counts[ErrorUploadFailed]) +
		atomic.LoadInt64(&r.counts[ErrorServerError]) +
		atomic.LoadInt64(&r.counts[DiscardedNotSelected]) +
		atomic.LoadInt64(&r.counts[UploadedUpgraded]) +
		atomic.LoadInt64(&r.counts[UploadedServerDuplicate]) +
		atomic.LoadInt64(&r.counts[DiscardedServerBetter]) +
		atomic.LoadInt64(&r.counts[DiscoveredDiscarded]) +
		atomic.LoadInt64(&r.counts[DiscardedLocalDuplicate])
	if !forcedMissingJSON {
		v += atomic.LoadInt64(&r.counts[AnalysisMissingAssociatedMetadata])
	}
	return v
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
			sb.WriteString(fmt.Sprintf("  %-35s: %7d  (%s)\n", c.String(), count, formatEventBytes(size)))
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
			sb.WriteString(fmt.Sprintf("  %-35s: %7d  (%s)\n", c.String(), count, formatEventBytes(size)))
		}
	}

	// Asset Lifecycle - To PROCESSED
	hasProcessed := false
	for _, c := range []Code{UploadedSuccess, UploadedUpgraded} {
		if eventCounts[c] > 0 {
			hasProcessed = true
			break
		}
	}
	if hasProcessed {
		sb.WriteString("\nAsset Lifecycle (PROCESSED):\n")
		for _, c := range []Code{UploadedSuccess, UploadedUpgraded} {
			if count := eventCounts[c]; count > 0 {
				sb.WriteString(fmt.Sprintf("  %-35s: %7d\n", c.String(), count))
			}
		}
	}

	// Asset Lifecycle - To DISCARDED
	hasDiscarded := false
	for _, c := range []Code{
		UploadedServerDuplicate,
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
			UploadedServerDuplicate,
			DiscardedBanned,
			DiscardedUnsupported,
			DiscardedFiltered,
			DiscardedLocalDuplicate,
			DiscardedNotSelected,
			DiscardedServerBetter,
		} {
			if count := eventCounts[c]; count > 0 {
				sb.WriteString(fmt.Sprintf("  %-35s: %7d\n", c.String(), count))
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
		AnalysisAssociatedMetadata,
		AnalysisMissingAssociatedMetadata,
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
			AnalysisAssociatedMetadata,
			AnalysisMissingAssociatedMetadata,
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

// formatEventBytes formats byte count as human-readable string
func formatEventBytes(bytes int64) string {
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
