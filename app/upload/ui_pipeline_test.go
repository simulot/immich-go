package upload

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/simulot/immich-go/internal/ui/core/messages"
	"github.com/simulot/immich-go/internal/ui/core/state"
	uitesting "github.com/simulot/immich-go/internal/ui/testing"
)

func TestPublishAssetQueuedUpdatesStats(t *testing.T) {
	mem := uitesting.NewMemPublisher()
	uc := &UpCmd{uiPublisher: mem}
	asset := sampleAsset(42)

	uc.publishAssetQueued(context.Background(), asset)

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

	uc.publishAssetUploaded(context.Background(), asset)

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

	uc.publishAssetFailed(context.Background(), asset, context.Canceled)

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
}

func sampleAsset(size int) *assets.Asset {
	return &assets.Asset{
		File:     fshelper.FSName(fstest.MapFS{}, "photo.jpg"),
		FileSize: size,
	}
}
