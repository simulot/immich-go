package upload

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/gen/syncset"
)

type countingLockClient struct {
	albumCalls atomic.Int32
	setCalls   atomic.Int32
}

func (c *countingLockClient) GetAssetAlbums(context.Context, string) ([]immich.AlbumSimplified, error) {
	c.albumCalls.Add(1)
	return nil, nil
}

func (c *countingLockClient) SetAssetsVisibility(context.Context, []string, assets.Visibility) error {
	c.setCalls.Add(1)
	return nil
}

func TestLockServerAssetDoesNotModifyTheServerAsset(t *testing.T) {
	server := &assets.Asset{ID: "s1", Visibility: assets.VisibilityTimeline}
	got, _ := lockServerAsset(context.Background(), &countingLockClient{}, syncset.New[string](), &assets.Asset{Visibility: assets.VisibilityLocked}, server)
	if got != lockApplied {
		t.Fatalf("got %d, want lockApplied", got)
	}
	if server.Visibility != assets.VisibilityTimeline {
		t.Errorf("the shared server asset was modified: %q", server.Visibility)
	}
}

func TestLockServerAssetOnlyOncePerServerAsset(t *testing.T) {
	c := &countingLockClient{}
	lockedIDs := syncset.New[string]()
	input := &assets.Asset{Visibility: assets.VisibilityLocked}
	server := &assets.Asset{ID: "s1"}

	if got, _ := lockServerAsset(context.Background(), c, lockedIDs, input, server); got != lockApplied {
		t.Fatalf("first call: got %d, want lockApplied", got)
	}
	if got, _ := lockServerAsset(context.Background(), c, lockedIDs, input, server); got != lockNotNeeded {
		t.Fatalf("second call: got %d, want lockNotNeeded", got)
	}
	if c.albumCalls.Load() != 1 || c.setCalls.Load() != 1 {
		t.Errorf("server calls: albums=%d set=%d, want 1 and 1", c.albumCalls.Load(), c.setCalls.Load())
	}
}

// Run with -race: workers share the server asset of the index.
func TestLockServerAssetConcurrent(t *testing.T) {
	c := &countingLockClient{}
	lockedIDs := syncset.New[string]()
	server := &assets.Asset{ID: "s1"}

	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			lockServerAsset(context.Background(), c, lockedIDs, &assets.Asset{Visibility: assets.VisibilityLocked}, server)
		})
	}
	wg.Wait()

	if !lockedIDs.Contains("s1") {
		t.Error("the asset should be remembered as locked")
	}
}
