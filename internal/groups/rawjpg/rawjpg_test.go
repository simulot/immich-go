package rawjpg_test

import (
	"context"
	"testing"
	"time"

	"github.com/simulot/immich-go/internal/groups"
	"github.com/simulot/immich-go/internal/groups/rawjpg"
	"github.com/simulot/immich-go/internal/metadata"
)

type mockAsset struct {
	radical string
	ext     string
	typ     string
	taken   time.Time
}

func (m *mockAsset) Radical() string {
	return m.radical
}

func (m *mockAsset) Ext() string {
	return m.ext
}

func (m *mockAsset) Type() string {
	return m.typ
}

func (m *mockAsset) Base() string {
	return m.radical + m.ext
}

func (m *mockAsset) DateTaken() time.Time {
	return m.taken
}

func (m *mockAsset) String() string {
	return m.Base() + m.DateTaken().String()
}

func TestRawJpgGrouper_Group(t *testing.T) {
	now := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	input := []groups.Asset{
		&mockAsset{radical: "img1", ext: ".raw", typ: metadata.TypeImage, taken: now},                        // group 1
		&mockAsset{radical: "img1", ext: ".jpg", typ: metadata.TypeImage, taken: now.Add(1 * time.Second)},   // group 1
		&mockAsset{radical: "img1", ext: ".jpg", typ: metadata.TypeImage, taken: now.Add(20 * time.Second)},  // asset 1
		&mockAsset{radical: "img2", ext: ".jpg", typ: metadata.TypeImage, taken: now},                        // group 2
		&mockAsset{radical: "img2", ext: ".raw", typ: metadata.TypeImage, taken: now.Add(-1 * time.Second)},  // group 2
		&mockAsset{radical: "img2", ext: ".raw", typ: metadata.TypeImage, taken: now.Add(-20 * time.Second)}, // asset 2
		&mockAsset{radical: "img3", ext: ".jpg", typ: metadata.TypeImage, taken: now.Add(1 * time.Second)},   // asset 3
		&mockAsset{radical: "img4", ext: ".raw", typ: metadata.TypeImage, taken: now.Add(1 * time.Second)},   // asset 4
		&mockAsset{radical: "img4", ext: ".mp4", typ: metadata.TypeVideo, taken: now.Add(1 * time.Second)},   // asset 5
	}
	tests := []struct {
		name           string
		mode           rawjpg.RawJpgMode
		expectedGroups []*groups.AssetGroup
		expectedAssets []groups.Asset
	}{
		{
			name: "ModeNone",
			mode: rawjpg.ModeNone,
			expectedAssets: []groups.Asset{
				&mockAsset{radical: "img1", ext: ".raw", typ: metadata.TypeImage, taken: now},
				&mockAsset{radical: "img1", ext: ".jpg", typ: metadata.TypeImage, taken: now.Add(1 * time.Second)},
				&mockAsset{radical: "img1", ext: ".jpg", typ: metadata.TypeImage, taken: now.Add(20 * time.Second)},
				&mockAsset{radical: "img2", ext: ".jpg", typ: metadata.TypeImage, taken: now},
				&mockAsset{radical: "img2", ext: ".raw", typ: metadata.TypeImage, taken: now.Add(1 * time.Second)},
				&mockAsset{radical: "img2", ext: ".raw", typ: metadata.TypeImage, taken: now.Add(20 * time.Second)},
				&mockAsset{radical: "img3", ext: ".jpg", typ: metadata.TypeImage, taken: now.Add(1 * time.Second)},
				&mockAsset{radical: "img4", ext: ".raw", typ: metadata.TypeImage, taken: now.Add(1 * time.Second)},
				&mockAsset{radical: "img4", ext: ".mp4", typ: metadata.TypeVideo, taken: now.Add(1 * time.Second)},
			},
		},
		{
			name: "ModeKeepRaw",
			mode: rawjpg.ModeKeepRaw,
			expectedAssets: []groups.Asset{
				&mockAsset{radical: "img1", ext: ".raw", typ: metadata.TypeImage, taken: now},
				&mockAsset{radical: "img1", ext: ".jpg", typ: metadata.TypeImage, taken: now.Add(20 * time.Second)},
				&mockAsset{radical: "img2", ext: ".raw", typ: metadata.TypeImage, taken: now.Add(1 * time.Second)},
				&mockAsset{radical: "img2", ext: ".raw", typ: metadata.TypeImage, taken: now.Add(20 * time.Second)},
				&mockAsset{radical: "img3", ext: ".jpg", typ: metadata.TypeImage, taken: now.Add(1 * time.Second)},
				&mockAsset{radical: "img4", ext: ".raw", typ: metadata.TypeImage, taken: now.Add(1 * time.Second)},
				&mockAsset{radical: "img4", ext: ".mp4", typ: metadata.TypeVideo, taken: now.Add(1 * time.Second)},
			},
		},
		{
			name: "ModeKeepJpg",
			mode: rawjpg.ModeKeepJpg,
			expectedAssets: []groups.Asset{
				&mockAsset{radical: "img1", ext: ".jpg", typ: metadata.TypeImage, taken: now.Add(1 * time.Second)},
				&mockAsset{radical: "img1", ext: ".jpg", typ: metadata.TypeImage, taken: now.Add(20 * time.Second)},
				&mockAsset{radical: "img2", ext: ".jpg", typ: metadata.TypeImage, taken: now},
				&mockAsset{radical: "img2", ext: ".raw", typ: metadata.TypeImage, taken: now.Add(20 * time.Second)},
				&mockAsset{radical: "img3", ext: ".jpg", typ: metadata.TypeImage, taken: now.Add(1 * time.Second)},
				&mockAsset{radical: "img4", ext: ".raw", typ: metadata.TypeImage, taken: now.Add(1 * time.Second)},
				&mockAsset{radical: "img4", ext: ".mp4", typ: metadata.TypeVideo, taken: now.Add(1 * time.Second)},
			},
		},
		{
			name: "ModeRawCover",
			mode: rawjpg.ModeRawCover,
			expectedGroups: []*groups.AssetGroup{
				groups.NewAssetGroup(groups.KindRawJpg,
					&mockAsset{radical: "img1", ext: ".raw", typ: metadata.TypeImage, taken: now},
					&mockAsset{radical: "img1", ext: ".jpg", typ: metadata.TypeImage, taken: now.Add(1 * time.Second)}).SetCover(0),
				groups.NewAssetGroup(groups.KindRawJpg,
					&mockAsset{radical: "img2", ext: ".jpg", typ: metadata.TypeImage, taken: now},
					&mockAsset{radical: "img2", ext: ".raw", typ: metadata.TypeImage, taken: now.Add(1 * time.Second)}).SetCover(1),
			},
			expectedAssets: []groups.Asset{
				&mockAsset{radical: "img1", ext: ".jpg", typ: metadata.TypeImage, taken: now.Add(20 * time.Second)},  // asset 1
				&mockAsset{radical: "img2", ext: ".raw", typ: metadata.TypeImage, taken: now.Add(-20 * time.Second)}, // asset 2
				&mockAsset{radical: "img3", ext: ".jpg", typ: metadata.TypeImage, taken: now.Add(1 * time.Second)},   // asset 3
				&mockAsset{radical: "img4", ext: ".raw", typ: metadata.TypeImage, taken: now.Add(1 * time.Second)},   // asset 4
				&mockAsset{radical: "img4", ext: ".mp4", typ: metadata.TypeVideo, taken: now.Add(1 * time.Second)},   // asset 5
			},
		},
		{
			name: "ModeJpgCover",
			mode: rawjpg.ModeJpgCover,
			expectedGroups: []*groups.AssetGroup{
				groups.NewAssetGroup(groups.KindRawJpg,
					&mockAsset{radical: "img1", ext: ".raw", typ: metadata.TypeImage, taken: now},
					&mockAsset{radical: "img1", ext: ".jpg", typ: metadata.TypeImage, taken: now.Add(1 * time.Second)}).SetCover(1),
				groups.NewAssetGroup(groups.KindRawJpg,
					&mockAsset{radical: "img2", ext: ".jpg", typ: metadata.TypeImage, taken: now},
					&mockAsset{radical: "img2", ext: ".raw", typ: metadata.TypeImage, taken: now.Add(1 * time.Second)}).SetCover(0),
			},
			expectedAssets: []groups.Asset{
				&mockAsset{radical: "img1", ext: ".jpg", typ: metadata.TypeImage, taken: now.Add(20 * time.Second)},  // asset 1
				&mockAsset{radical: "img2", ext: ".raw", typ: metadata.TypeImage, taken: now.Add(-20 * time.Second)}, // asset 2
				&mockAsset{radical: "img3", ext: ".jpg", typ: metadata.TypeImage, taken: now.Add(1 * time.Second)},   // asset 3
				&mockAsset{radical: "img4", ext: ".raw", typ: metadata.TypeImage, taken: now.Add(1 * time.Second)},   // asset 4
				&mockAsset{radical: "img4", ext: ".mp4", typ: metadata.TypeVideo, taken: now.Add(1 * time.Second)},   // asset 5
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			in := make(chan groups.Asset, len(input))

			gr := &rawjpg.RawJpgGrouper{Mode: tt.mode}

			for _, asset := range input {
				in <- asset
			}
			close(in)

			outg, outa := gr.Group(ctx, in)

			var gotGroups []*groups.AssetGroup
			var gotAssets []groups.Asset
			running := true
			for running {
				select {
				case <-ctx.Done():
					// t.Fatal("timeout")
				case g, ok := <-outg:
					if ok {
						gotGroups = append(gotGroups, g)
					}
				case a, ok := <-outa:
					if !ok {
						running = false
					} else {
						gotAssets = append(gotAssets, a)
					}
				}
			}
			if len(gotAssets) != len(tt.expectedAssets) {
				t.Fatalf("expected %d assets, got %d", len(tt.expectedAssets), len(gotAssets))
			} else {
				for i, asset := range gotAssets {
					if asset.Radical() != tt.expectedAssets[i].Radical() || asset.Ext() != tt.expectedAssets[i].Ext() {
						t.Errorf("expected asset %s, got %s", tt.expectedAssets[i], asset)
					}
				}
			}
			if len(gotGroups) != len(tt.expectedGroups) {
				t.Fatalf("expected %d groups, got %d", len(tt.expectedGroups), len(gotGroups))
			} else {
				for i, group := range gotGroups {
					for j, asset := range group.Assets {
						if asset.Radical() != tt.expectedGroups[i].Assets[j].Radical() || asset.Ext() != tt.expectedGroups[i].Assets[j].Ext() {
							t.Errorf("expected group %s, got %s", tt.expectedGroups[i].Assets[j], asset)
						}
					}
				}
			}
		})
	}
}
