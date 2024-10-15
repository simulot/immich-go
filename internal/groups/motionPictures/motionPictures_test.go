package motionpictures

import (
	"context"
	"testing"
	"time"

	"github.com/simulot/immich-go/internal/groups"
	"github.com/simulot/immich-go/internal/metadata"
)

func TestMotionPictureGrouper_Run(t *testing.T) {
	supported := metadata.SupportedMedia{
		Images: []string{".jpg", ".jpeg", ".png"},
		Videos: []string{".mp4", ".mov"},
	}
	grouper := New(supported)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	in := make(chan groups.Asset)
	out := grouper.Run(ctx, in)

	// Create test assets
	asset1 := groups.NewMockAsset("IMG_001.jpg", metadata.TypeImage, time.Now())
	asset2 := groups.NewMockAsset("IMG_001.mp4", metadata.TypeVideo, time.Now().Add(1*time.Second))
	asset3 := groups.NewMockAsset("IMG_002.jpg", metadata.TypeImage, time.Now().Add(3*time.Second))

	// Send assets to the grouper
	go func() {
		in <- asset1
		in <- asset2
		in <- asset3
		close(in)
	}()

	// Collect results
	var groups []*groups.AssetGroup
	for g := range out {
		groups = append(groups, g)
	}

	// Validate results
	if len(groups) != 2 {
		t.Errorf("expected 2 groups, got %d", len(groups))
	}

	// Check first group
	if groups[0].Kind != groups.KindMotionPhoto {
		t.Errorf("expected first group to be KindMotionPhoto, got %v", groups[0].Kind)
	}
	if len(groups[0].Assets) != 2 {
		t.Errorf("expected first group to have 2 assets, got %d", len(groups[0].Assets))
	}

	// Check second group
	if groups[1].Kind != groups.KindNone {
		t.Errorf("expected second group to be KindNone, got %v", groups[1].Kind)
	}
	if len(groups[1].Assets) != 1 {
		t.Errorf("expected second group to have 1 asset, got %d", len(groups[1].Assets))
	}
}
