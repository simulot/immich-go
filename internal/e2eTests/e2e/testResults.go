//go:build e2e

package e2e

import (
	"testing"

	"github.com/simulot/immich-go/internal/fileevent"
)

func CheckResults(t *testing.T, expectedTesults map[fileevent.Code]int64, forcedJSON bool, recorder *fileevent.Recorder) bool {
	r := true

	if recorder != nil {

		gotResults := recorder.GetCounts()
		for code, value := range expectedTesults {
			if gotResults[code] != value {
				t.Errorf("Expected %d results for code '%s', got %d", value, code.String(), gotResults[code])
				r = false
			}
		}

		if recorder.TotalAssets() != recorder.TotalProcessed(forcedJSON) {
			t.Errorf("Total assets %d does not match total processed %d", recorder.TotalAssets(), recorder.TotalProcessed(forcedJSON))
			r = false
		}
	}

	return r
}
