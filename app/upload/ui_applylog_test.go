package upload

import (
	"testing"

	"github.com/rivo/tview"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/ui/core/state"
)

func TestApplyLogEventCounters(t *testing.T) {
	ui := &uiPage{
		counts:    map[fileevent.Code]*tview.TextView{},
		sizes:     map[fileevent.Code]*tview.TextView{},
		logCounts: map[fileevent.Code]int64{},
		logSizes:  map[fileevent.Code]int64{},
	}
	ui.counts[fileevent.DiscoveredImage] = tview.NewTextView()
	ui.sizes[fileevent.DiscoveredImage] = tview.NewTextView()

	entry := state.LogEvent{
		Details: map[string]string{
			"event_code_id": "1", // DiscoveredImage
			"size_bytes":    "2048",
		},
	}

	ui.applyLogEventCounters(entry)

	if got := ui.counts[fileevent.DiscoveredImage].GetText(false); got != "     1" {
		t.Fatalf("want count 1, got %q", got)
	}
	if got := ui.sizes[fileevent.DiscoveredImage].GetText(false); got != ui.formatBytes(2048) {
		t.Fatalf("want size %q, got %q", ui.formatBytes(2048), got)
	}
}
