package filters

import (
	"testing"

	"github.com/simulot/immich-go/internal/assets"
)

func TestUnGroupRawJPGNothing(t *testing.T) {
	tests := []struct {
		name     string
		group    *assets.Group
		expected *assets.Group
	}{
		{
			name: "GroupByRawJpg",
			group: assets.NewGroup(assets.GroupByRawJpg,
				mockAsset("a.jpg"),
				mockAsset("a.raw"),
			),
			expected: assets.NewGroup(assets.GroupByNone,
				mockAsset("a.jpg"),
				mockAsset("a.raw"),
			),
		},
		{
			name: "NotGroupByRawJpg",
			group: assets.NewGroup(assets.GroupByBurst,
				mockAsset("a.jpg"),
				mockAsset("a.raw"),
			),
			expected: assets.NewGroup(assets.GroupByBurst,
				mockAsset("a.jpg"),
				mockAsset("a.raw"),
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := unGroupRawJPGNothing(tt.group)
			if result.Grouping != tt.expected.Grouping {
				t.Errorf("expected %v, got %v", tt.expected.Grouping, result.Grouping)
			}
		})
	}
}

func TestGroupRawJPGKeepRaw(t *testing.T) {
	tests := []struct {
		name     string
		group    *assets.Group
		expected *assets.Group
	}{
		{
			name: "GroupByRawJpgWithMixedFiles",
			group: assets.NewGroup(assets.GroupByRawJpg,
				mockAsset("a.jpg"),
				mockAsset("a.raw"),
			),
			expected: assets.NewGroup(assets.GroupByNone,
				mockAsset("a.raw"),
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := groupRawJPGKeepRaw(tt.group)
			if result.Grouping != tt.expected.Grouping {
				t.Errorf("expected grouping %v, got %v", tt.expected.Grouping, result.Grouping)
			}
			if len(result.Assets) != len(tt.expected.Assets) {
				t.Errorf("expected %d assets, got %d", len(tt.expected.Assets), len(result.Assets))
			}
			for i, asset := range result.Assets {
				if asset.File.Name() != tt.expected.Assets[i].File.Name() {
					t.Errorf("expected asset %v, got %v", tt.expected.Assets[i].File.Name(), asset.File.Name())
				}
			}
		})
	}
}

func TestGroupRawJPGKeepJPG(t *testing.T) {
	tests := []struct {
		name     string
		group    *assets.Group
		expected *assets.Group
	}{
		{
			name: "GroupByRawJpgWithMixedFiles",
			group: assets.NewGroup(assets.GroupByRawJpg,
				mockAsset("a.jpg"),
				mockAsset("a.raw"),
			),
			expected: assets.NewGroup(assets.GroupByNone,
				mockAsset("a.jpg"),
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := groupRawJPGKeepJPG(tt.group)
			if result.Grouping != tt.expected.Grouping {
				t.Errorf("expected grouping %v, got %v", tt.expected.Grouping, result.Grouping)
			}
			if len(result.Assets) != len(tt.expected.Assets) {
				t.Errorf("expected %d assets, got %d", len(tt.expected.Assets), len(result.Assets))
			}
			for i, asset := range result.Assets {
				if asset.File.Name() != tt.expected.Assets[i].File.Name() {
					t.Errorf("expected asset %v, got %v", tt.expected.Assets[i].File.Name(), asset.File.Name())
				}
			}
		})
	}
}
