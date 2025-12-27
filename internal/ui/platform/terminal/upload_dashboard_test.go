//go:build ui_terminal

package terminal

import (
	"strings"
	"testing"
	"time"

	"github.com/simulot/immich-go/internal/ui/core/messages"
	"github.com/simulot/immich-go/internal/ui/core/state"
)

func TestUploadDashboardModelConsumesStatsEvents(t *testing.T) {
	model := newUploadDashboardModel(Config{ProfileLabel: "prod"}, nil)
	stats := state.RunStats{Uploaded: 5, Pending: 3, Stage: state.StageRunning}

	updated, _ := model.Update(eventMsg{event: messages.Event{Type: messages.EventStatsUpdated, Payload: stats}})
	dm, ok := updated.(*uploadDashboardModel)
	if !ok {
		t.Fatalf("expected uploadDashboardModel, got %T", updated)
	}
	if dm.stats.Uploaded != 5 {
		t.Fatalf("expected uploaded count 5, got %d", dm.stats.Uploaded)
	}
	if dm.stats.Pending != 3 {
		t.Fatalf("expected pending count 3, got %d", dm.stats.Pending)
	}
}

func TestUploadDashboardModelConsumesInventoryEvents(t *testing.T) {
	model := newUploadDashboardModel(Config{ProfileLabel: "prod"}, nil)
	inv := state.ServerInventory{Photos: 42, Videos: 7, Total: 100, UpdatedAt: time.Now(), Latency: 32 * time.Millisecond}

	updated, _ := model.Update(eventMsg{event: messages.Event{Type: messages.EventInventoryUpdated, Payload: inv}})
	dm, ok := updated.(*uploadDashboardModel)
	if !ok {
		t.Fatalf("expected uploadDashboardModel, got %T", updated)
	}
	if dm.inventory.Photos != 42 || dm.inventory.Videos != 7 || dm.inventory.Total != 100 {
		t.Fatalf("inventory not updated: %+v", dm.inventory)
	}
}

func TestUploadDashboardViewRendersPanels(t *testing.T) {
	model := newUploadDashboardModel(Config{
		ProfileLabel: "demo",
		ServerURL:    "http://localhost:2283",
		UserEmail:    "test@example.com",
	}, nil)
	model.stats = state.RunStats{
		TotalDiscovered:      3214,
		TotalDiscoveredBytes: 89 << 30,
		Pending:              1082,
		PendingBytes:         31 << 30,
		Uploaded:             944,
		BytesSent:            27 << 30,
		Stage:                state.StageRunning,
		LastUpdated:          time.Now(),
	}
	model.inventory = state.ServerInventory{Photos: 4242, Videos: 512, Total: 4754, UpdatedAt: time.Now()}
	model.logs = []state.LogEvent{
		{Level: "INF", Message: "test log message", Timestamp: time.Now()},
	}
	view := model.View()
	checks := []string{"immich-go", "http://localhost:2283", "test@example.com", "test log message"}
	for _, want := range checks {
		if !strings.Contains(view, want) {
			t.Fatalf("view missing %q\n%s", want, view)
		}
	}
}

func TestListenToStreamSignalsClosure(t *testing.T) {
	ch := make(chan messages.Event)
	cmd := listenToStream(ch)
	go close(ch)
	msg := cmd()
	if _, ok := msg.(streamClosedMsg); !ok {
		t.Fatalf("expected streamClosedMsg, got %T", msg)
	}
}
