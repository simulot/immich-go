package upload

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/assettracker"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/simulot/immich-go/internal/ui/core/messages"
	"github.com/simulot/immich-go/internal/ui/core/state"
	uitesting "github.com/simulot/immich-go/internal/ui/testing"
)

func TestPublishAssetQueuedUpdatesStats(t *testing.T) {
	mem := uitesting.NewMemPublisher()
	uc := &UpCmd{uiPublisher: mem}
	asset := sampleAsset(42)

	uc.publishAssetQueued(context.Background(), asset, fileevent.DiscoveredImage)

	events := mem.Events()
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Type != messages.EventAssetQueued {
		t.Fatalf("expected first event to be AssetQueued, got %s", events[0].Type)
	}
	stats, ok := events[1].Payload.(state.RunStats)
	if !ok {
		t.Fatalf("expected stats payload, got %T", events[1].Payload)
	}
	if stats.Queued != 1 {
		t.Fatalf("expected queued count to be 1, got %d", stats.Queued)
	}
}

func TestPublishAssetUploadedUpdatesBytes(t *testing.T) {
	mem := uitesting.NewMemPublisher()
	uc := &UpCmd{uiPublisher: mem}
	asset := sampleAsset(2048)

	uc.publishAssetUploaded(context.Background(), asset, fileevent.ProcessedUploadSuccess, int64(asset.FileSize), nil)

	events := mem.Events()
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Type != messages.EventAssetUploaded {
		t.Fatalf("expected first event to be AssetUploaded, got %s", events[0].Type)
	}
	stats, ok := events[1].Payload.(state.RunStats)
	if !ok {
		t.Fatalf("expected stats payload, got %T", events[1].Payload)
	}
	if stats.Uploaded != 1 {
		t.Fatalf("expected uploaded count to be 1, got %d", stats.Uploaded)
	}
	if stats.BytesSent != 2048 {
		t.Fatalf("expected bytes to be 2048, got %d", stats.BytesSent)
	}
}

func TestPublishAssetFailedUpdatesCounter(t *testing.T) {
	mem := uitesting.NewMemPublisher()
	uc := &UpCmd{uiPublisher: mem}
	asset := sampleAsset(10)

	uc.publishAssetFailed(context.Background(), asset, fileevent.ErrorServerError, context.Canceled, nil)

	events := mem.Events()
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Type != messages.EventAssetFailed {
		t.Fatalf("expected first event to be AssetFailed, got %s", events[0].Type)
	}
	stats, ok := events[1].Payload.(state.RunStats)
	if !ok {
		t.Fatalf("expected stats payload, got %T", events[1].Payload)
	}
	if stats.Failed != 1 {
		t.Fatalf("expected failed count to be 1, got %d", stats.Failed)
	}
	if !stats.HasErrors {
		t.Fatalf("expected HasErrors to be true when failures occur")
	}
}

func TestForwardProcessingEventAppendsLog(t *testing.T) {
	mem := uitesting.NewMemPublisher()
	uc := &UpCmd{uiPublisher: mem}
	asset := sampleAsset(0)
	attrs := map[string]string{"album": "Family"}

	uc.forwardProcessingEvent(context.Background(), fileevent.ProcessedAlbumAdded, asset.File, 0, attrs)

	events := mem.Events()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != messages.EventLogLine {
		t.Fatalf("expected log event, got %s", events[0].Type)
	}
	logEvent, ok := events[0].Payload.(state.LogEvent)
	if !ok {
		t.Fatalf("expected LogEvent payload, got %T", events[0].Payload)
	}
	if got := logEvent.Details["album"]; got != "Family" {
		t.Fatalf("expected album detail, got %q", got)
	}
	if logEvent.Details["event_code"] != fileevent.ProcessedAlbumAdded.String() {
		t.Fatalf("expected event_code detail, got %q", logEvent.Details["event_code"])
	}
}

func TestForwardProcessingEventIgnoresUntrackedCodes(t *testing.T) {
	mem := uitesting.NewMemPublisher()
	uc := &UpCmd{uiPublisher: mem}
	asset := sampleAsset(0)

	uc.forwardProcessingEvent(context.Background(), fileevent.DiscoveredImage, asset.File, 0, nil)

	if len(mem.Events()) != 0 {
		t.Fatalf("expected no events for non-processing code, got %d", len(mem.Events()))
	}
}

func TestApplyCountersSnapshotUpdatesStats(t *testing.T) {
	mem := uitesting.NewMemPublisher()
	uc := &UpCmd{uiPublisher: mem}
	counters := assettracker.AssetCounters{
		Pending:       3,
		PendingSize:   300,
		Processed:     2,
		ProcessedSize: 200,
		Discarded:     1,
		DiscardedSize: 50,
		Errors:        1,
		ErrorSize:     10,
		AssetSize:     360,
	}
	uc.applyCountersSnapshot(context.Background(), counters)

	events := mem.Events()
	if len(events) != 1 {
		t.Fatalf("expected stats event, got %d", len(events))
	}
	stats, ok := events[0].Payload.(state.RunStats)
	if !ok {
		t.Fatalf("expected RunStats payload, got %T", events[0].Payload)
	}
	if stats.Pending != 3 || stats.Processed != 2 || stats.Discarded != 1 {
		t.Fatalf("unexpected stats snapshot: %+v", stats)
	}
	if stats.TotalDiscoveredBytes != 360 {
		t.Fatalf("expected total bytes 360, got %d", stats.TotalDiscoveredBytes)
	}
}

func TestRecordAndFlushCountersSnapshot(t *testing.T) {
	mem := uitesting.NewMemPublisher()
	uc := &UpCmd{uiPublisher: mem}
	counters := assettracker.AssetCounters{Pending: 5}
	uc.recordCountersSnapshot(counters)
	uc.flushStatsFromCounters(context.Background())

	events := mem.Events()
	if len(events) != 1 {
		t.Fatalf("expected 1 stats event, got %d", len(events))
	}
	stats := events[0].Payload.(state.RunStats)
	if stats.Pending != 5 {
		t.Fatalf("expected pending 5, got %d", stats.Pending)
	}
	// Ensure second flush without new data emits nothing
	initialCount := len(mem.Events())
	uc.flushStatsFromCounters(context.Background())
	if len(mem.Events()) != initialCount {
		t.Fatalf("expected no new events without dirty snapshot")
	}
}

func TestFanOutEventStreamCopiesEvents(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	source := make(chan messages.Event, 1)
	sinkA := make(chan messages.Event, 1)
	sinkB := make(chan messages.Event, 1)
	go fanOutEventStream(ctx, messages.Stream(source), sinkA, sinkB)
	want := messages.Event{Type: messages.EventLogLine}
	source <- want
	close(source)
	for _, sink := range []<-chan messages.Event{sinkA, sinkB} {
		select {
		case got, ok := <-sink:
			if !ok {
				t.Fatalf("sink closed before receiving event")
			}
			if got.Type != want.Type {
				t.Fatalf("expected event %v, got %v", want.Type, got.Type)
			}
		case <-time.After(time.Second):
			t.Fatalf("timeout waiting for fan-out delivery")
		}
	}
}

func TestDrainEventStreamStopsOnClose(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	source := make(chan messages.Event)
	done := make(chan struct{})
	go func() {
		drainEventStream(ctx, messages.Stream(source))
		close(done)
	}()
	close(source)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatalf("drainEventStream did not exit after source closed")
	}
}

func TestLogUIEventsWritesToLogger(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream := make(chan messages.Event, 1)
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	go logUIEvents(ctx, stream, logger)
	stream <- messages.Event{Type: messages.EventLogLine, Payload: state.LogEvent{Level: "info", Message: "hello"}}
	close(stream)
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if strings.Contains(buf.String(), "hello") {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected dump log to contain message, got %q", buf.String())
}

func TestLogUIEventsSummarizesJobs(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream := make(chan messages.Event, 1)
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	go logUIEvents(ctx, stream, logger)
	stream <- messages.Event{Type: messages.EventJobsUpdated, Payload: []state.JobSummary{{Name: "background-uploader"}}}
	close(stream)
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		logline := buf.String()
		if strings.Contains(logline, "jobs=1") && strings.Contains(logline, "background-uploader") {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected dump log to summarize jobs, got %q", buf.String())
}

func sampleAsset(size int) *assets.Asset {
	return &assets.Asset{
		File:     fshelper.FSName(fstest.MapFS{}, "photo.jpg"),
		FileSize: size,
	}
}
