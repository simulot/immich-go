package e2eutils

import (
	"bufio"
	"encoding/json"
	"os"
	"sync"
	"testing"

	"github.com/simulot/immich-go/internal/ui/core/messages"
	"github.com/simulot/immich-go/internal/ui/core/state"
)

// StatsCapture collects RunStats snapshots from the event bus (for testing).
type StatsCapture struct {
	mu    sync.Mutex
	stats []state.RunStats
}

// NewStatsCapture creates a new in-memory stats collector.
func NewStatsCapture() *StatsCapture {
	return &StatsCapture{stats: []state.RunStats{}}
}

// Record adds a RunStats snapshot to the collection.
func (sc *StatsCapture) Record(s state.RunStats) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.stats = append(sc.stats, s)
}

// All returns all collected RunStats snapshots in order.
func (sc *StatsCapture) All() []state.RunStats {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	result := make([]state.RunStats, len(sc.stats))
	copy(result, sc.stats)
	return result
}

// Last returns the last meaningful (non-empty) RunStats snapshot, or fails if none captured.
func (sc *StatsCapture) Last(t testing.TB) state.RunStats {
	t.Helper()
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if len(sc.stats) == 0 {
		t.Fatalf("no RunStats captured")
	}
	// Prefer the last meaningful snapshot (counters, bytes, pending, or terminal stage)
	isMeaningful := func(s state.RunStats) bool {
		// if s.Stage != "" /*&& s.Stage != state.StageRunning*/ {
		// 	return true
		// }
		return (s.Uploaded+s.Discarded+s.Failed+s.TotalDiscovered+s.Processed+s.ErrorCount+s.Pending+s.Retries) > 0 ||
			s.BytesSent > 0 || s.PendingBytes > 0 || s.ProcessedBytes > 0
	}
	for i := len(sc.stats) - 1; i >= 0; i-- {
		s := sc.stats[i]
		if isMeaningful(s) {
			return s
		}
	}
	// Fallback: return the last entry even if zeroed
	return sc.stats[len(sc.stats)-1]
}

// LoadRunStatsEvents reads the NDJSON file produced by IMMICH_GO_UI_EVENTS_FILE and
// returns all RunStats payloads (one per stats event), preserving order.
func LoadRunStatsEvents(t testing.TB, path string) []state.RunStats {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("cannot open UI events file %s: %v", path, err)
	}
	defer f.Close()

	type eventRecord struct {
		Type    messages.EventType `json:"type"`
		Payload state.RunStats     `json:"payload"`
	}

	var stats []state.RunStats
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		var rec eventRecord
		if err := json.Unmarshal(line, &rec); err != nil {
			t.Fatalf("cannot decode UI event line: %v", err)
		}
		if rec.Type == messages.EventStatsUpdated {
			stats = append(stats, rec.Payload)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("error reading UI events file: %v", err)
	}
	return stats
}

// LastRunStats is a convenience to get the final RunStats from a list.
func LastRunStats(t testing.TB, stats []state.RunStats) state.RunStats {
	t.Helper()
	if len(stats) == 0 {
		t.Fatalf("no RunStats events captured")
	}
	// Prefer the last non-empty snapshot to avoid initial zero snapshots.
	for i := len(stats) - 1; i >= 0; i-- {
		s := stats[i]
		if (s.Uploaded+s.Discarded+s.Failed+s.TotalDiscovered) > 0 || (s.Processed+s.ErrorCount) > 0 {
			return s
		}
	}
	// Fallback: return the last entry even if zeroed
	return stats[len(stats)-1]
}
