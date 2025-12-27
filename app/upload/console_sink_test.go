package upload

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/simulot/immich-go/internal/fileevent"
)

func TestConsoleSinkProgressLine(t *testing.T) {
	sink := newConsoleSink(false)
	ctx := context.Background()

	sink.HandleEvent(ctx, fileevent.DiscoveredImage, "", 0, nil)
	sink.HandleEvent(ctx, fileevent.DiscoveredVideo, "", 0, nil)
	sink.HandleEvent(ctx, fileevent.ErrorServerError, "", 0, nil)
	sink.HandleEvent(ctx, fileevent.ProcessedUploadSuccess, "", 0, nil)
	sink.SetImmichProgress(5, 10)

	line := sink.progressLine()
	wants := []string{"Immich read 50%", "Assets found: 2", "Upload errors: 1", "Uploaded 1"}
	for _, want := range wants {
		if !strings.Contains(line, want) {
			t.Fatalf("progress line missing %q: %s", want, line)
		}
	}
}

func TestConsoleSinkStartStop(t *testing.T) {
	sink := newConsoleSink(true)
	ctx, cancel := context.WithCancel(context.Background())

	sink.Start(ctx)
	cancel()

	done := make(chan struct{})
	go func() {
		sink.Stop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("console sink Stop did not return")
	}
}
