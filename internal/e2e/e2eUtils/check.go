package e2eutils

import (
	"testing"

	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/fileprocessor"
)

func CheckResults(t *testing.T, expectedResults map[fileevent.Code]int64, forcedJSON bool, processor *fileprocessor.FileProcessor) bool {
	r := true

	if processor != nil {
		gotResults := processor.GetEventCounts()
		for code, value := range expectedResults {
			if gotResults[code] != value {
				t.Errorf("Expected %d results for code '%s', got %d", value, code.String(), gotResults[code])
				r = false
			}
		}

		// Check asset tracking completeness
		counters := processor.GetAssetCounters()
		if counters.Pending > 0 {
			t.Errorf("Found %d pending assets that never reached final state", counters.Pending)
			r = false
		}
	}
	return r
}
