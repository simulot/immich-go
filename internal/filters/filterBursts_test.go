package filters

import (
	"testing"
	"time"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/filetypes"
	"github.com/simulot/immich-go/internal/fshelper"
)

var ic = filenames.NewInfoCollector(time.Local, filetypes.DefaultSupportedMedia)

func mockAsset(name string) *assets.Asset {
	a := &assets.Asset{
		File: fshelper.FSName(nil, name),
	}
	a.SetNameInfo(ic.GetInfo(name))
	return a
}

func Test_unGroupBurst(t *testing.T) {
	tests := []struct {
		name     string
		group    *assets.Group
		expected *assets.Group
	}{
		{
			name: "GroupByBurst",
			group: &assets.Group{
				Grouping: assets.GroupByBurst,
			},
			expected: &assets.Group{
				Grouping: assets.GroupByNone,
			},
		},
		{
			name: "NotGroupByBurst",
			group: &assets.Group{
				Grouping: assets.GroupByOther,
			},
			expected: &assets.Group{
				Grouping: assets.GroupByOther,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := unGroupBurst(tt.group)
			if result.Grouping != tt.expected.Grouping {
				t.Errorf("expected %v, got %v", tt.expected.Grouping, result.Grouping)
			}
		})
	}
}

func Test_stackBurstKeepJPEG(t *testing.T) {
	tests := []struct {
		name      string
		group     *assets.Group
		jpgCount  int
		rawCount  int
		heicCount int
		expected  *assets.Group
	}{
		{
			name: "GroupByBurstWithJPEG",
			group: assets.NewGroup(assets.GroupByBurst,
				mockAsset("photo1.jpg"),
				mockAsset("photo2.jpg"),
			),
			jpgCount: 2,
			expected: assets.NewGroup(assets.GroupByBurst,
				mockAsset("photo1.jpg"),
				mockAsset("photo2.jpg"),
			),
		},
		{
			name: "GroupByBurstWithMixed",
			group: assets.NewGroup(assets.GroupByBurst,
				mockAsset("photo1.raw"),
				mockAsset("photo2.jpg"),
			),
			rawCount: 1,
			jpgCount: 1,
			expected: assets.NewGroup(assets.GroupByBurst,
				mockAsset("photo2.jpg"),
			),
		},
		{
			name: "NotGroupByBurst",
			group: assets.NewGroup(assets.GroupByOther,
				mockAsset("photo1.jpg"),
				mockAsset("photo2.jpg"),
			),
			jpgCount: 2,
			expected: assets.NewGroup(assets.GroupByOther,
				mockAsset("photo1.jpg"),
				mockAsset("photo2.jpg"),
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stackBurstKeepJPEG(tt.group)
			if len(result.Assets) != len(tt.expected.Assets) {
				t.Errorf("expected %v assets, got %v", len(tt.expected.Assets), len(result.Assets))
			}
			for i, asset := range result.Assets {
				if asset.File.Name() != tt.expected.Assets[i].File.Name() {
					t.Errorf("expected asset %v, got %v", tt.expected.Assets[i].File.Name(), asset.File.Name())
				}
			}
		})
	}
}

func Test_groupBurstKeepRaw(t *testing.T) {
	tests := []struct {
		name      string
		group     *assets.Group
		jpgCount  int
		rawCount  int
		heicCount int
		expected  *assets.Group
	}{
		{
			name: "GroupByBurstWithRaw",
			group: assets.NewGroup(assets.GroupByBurst,
				mockAsset("photo1.raw"),
				mockAsset("photo2.raw"),
			),
			rawCount: 2,
			expected: assets.NewGroup(assets.GroupByBurst,
				mockAsset("photo1.raw"),
				mockAsset("photo2.raw"),
			),
		},
		{
			name: "GroupByBurstWithMixed",
			group: assets.NewGroup(assets.GroupByBurst,
				mockAsset("photo1.raw"),
				mockAsset("photo2.jpg"),
			),
			rawCount: 1,
			jpgCount: 1,
			expected: assets.NewGroup(assets.GroupByBurst,
				mockAsset("photo1.raw"),
			),
		},
		{
			name: "NotGroupByBurst",
			group: assets.NewGroup(assets.GroupByOther,
				mockAsset("photo1.raw"),
				mockAsset("photo2.jpg"),
			),
			rawCount: 1,
			jpgCount: 1,
			expected: assets.NewGroup(assets.GroupByOther,
				mockAsset("photo1.raw"),
				mockAsset("photo2.jpg"),
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := groupBurstKeepRaw(tt.group)
			if len(result.Assets) != len(tt.expected.Assets) {
				t.Errorf("expected %v assets, got %v", len(tt.expected.Assets), len(result.Assets))
			}
			for i, asset := range result.Assets {
				if asset.File.Name() != tt.expected.Assets[i].File.Name() {
					t.Errorf("expected asset %v, got %v", tt.expected.Assets[i].File.Name(), asset.File.Name())
				}
			}
		})
	}
}
