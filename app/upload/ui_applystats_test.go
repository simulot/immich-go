package upload

import (
	"fmt"
	"testing"

	"github.com/rivo/tview"
	"github.com/simulot/immich-go/internal/ui/core/state"
)

func TestApplyStatsUpdatesCounters(t *testing.T) {
	ui := &uiPage{
		statusViews:    map[string]*tview.TextView{},
		discoveryViews: map[string]*tview.TextView{},
	}
	keys := []string{"pendingCount", "pendingSize", "uploadedCount", "uploadedSize", "discardedCount", "discardedSize", "errorCount", "errorSize", "totalCount", "totalSize"}
	for _, k := range keys {
		ui.statusViews[k] = tview.NewTextView()
	}
	ui.discoveryViews["discoveredCount"] = tview.NewTextView()
	ui.discoveryViews["discoveredSize"] = tview.NewTextView()

	stats := state.RunStats{
		Pending:              3,
		PendingBytes:         300,
		Processed:            7,
		ProcessedBytes:       700,
		Discarded:            2,
		DiscardedBytes:       200,
		ErrorCount:           1,
		ErrorBytes:           100,
		TotalDiscovered:      13,
		TotalDiscoveredBytes: 1300,
	}

	ui.applyStats(stats)

	assertText := func(v *tview.TextView, want string) {
		got := v.GetText(false)
		if got != want {
			t.Fatalf("want %q, got %q", want, got)
		}
	}

	assertText(ui.statusViews["pendingCount"], fmt.Sprintf("%6d", 3))
	assertText(ui.statusViews["pendingSize"], ui.formatBytes(300))
	assertText(ui.statusViews["uploadedCount"], fmt.Sprintf("%6d", 7))
	assertText(ui.statusViews["uploadedSize"], ui.formatBytes(700))
	assertText(ui.statusViews["discardedCount"], fmt.Sprintf("%6d", 2))
	assertText(ui.statusViews["discardedSize"], ui.formatBytes(200))
	assertText(ui.statusViews["errorCount"], fmt.Sprintf("%6d", 1))
	assertText(ui.statusViews["errorSize"], ui.formatBytes(100))
	assertText(ui.statusViews["totalCount"], fmt.Sprintf("%6d", 13))
	assertText(ui.statusViews["totalSize"], ui.formatBytes(1300))
	assertText(ui.discoveryViews["discoveredCount"], fmt.Sprintf("%6d", 13))
	assertText(ui.discoveryViews["discoveredSize"], ui.formatBytes(1300))
}
