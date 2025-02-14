package filters

import (
	"testing"

	"github.com/simulot/immich-go/internal/assets"
)

func Test_unGroupHeicJpeg(t *testing.T) {
	tests := []struct {
		name     string
		group    *assets.Group
		expected *assets.Group
	}{
		{
			name: "GroupByHeicJpg",
			group: &assets.Group{
				Grouping: assets.GroupByHeicJpg,
				Assets: []*assets.Asset{
					mockAsset("photo1.heic"),
					mockAsset("photo2.jpg"),
				},
			},
			expected: &assets.Group{
				Grouping: assets.GroupByNone,
				Assets: []*assets.Asset{
					mockAsset("photo1.heic"),
					mockAsset("photo2.jpg"),
				},
			},
		},
		{
			name: "NotGroupByHeicJpg",
			group: &assets.Group{
				Grouping: assets.GroupByOther,
				Assets: []*assets.Asset{
					mockAsset("photo1.heic"),
					mockAsset("photo2.jpg"),
				},
			},
			expected: &assets.Group{
				Grouping: assets.GroupByOther,
				Assets: []*assets.Asset{
					mockAsset("photo1.heic"),
					mockAsset("photo2.jpg"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := unGroupHeicJpeg(tt.group)
			if result.Grouping != tt.expected.Grouping {
				t.Errorf("expected %v, got %v", tt.expected.Grouping, result.Grouping)
			}
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

func Test_groupHeicJpgKeepHeic(t *testing.T) {
	tests := []struct {
		name     string
		group    *assets.Group
		expected *assets.Group
	}{
		{
			name: "GroupByHeicJpgWithMixedAssets",
			group: &assets.Group{
				Grouping: assets.GroupByHeicJpg,
				Assets: []*assets.Asset{
					mockAsset("photo1.heic"),
					mockAsset("photo2.jpg"),
				},
			},
			expected: &assets.Group{
				Grouping: assets.GroupByNone,
				Assets: []*assets.Asset{
					mockAsset("photo1.heic"),
				},
			},
		},
		{
			name: "GroupByHeicJpgWithMixedAssets2",
			group: &assets.Group{
				Grouping: assets.GroupByHeicJpg,
				Assets: []*assets.Asset{
					mockAsset("photo1.jpg"),
					mockAsset("photo2.heic"),
				},
			},
			expected: &assets.Group{
				Grouping: assets.GroupByNone,
				Assets: []*assets.Asset{
					mockAsset("photo2.heic"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := groupHeicJpgKeepHeic(tt.group)
			if result.Grouping != tt.expected.Grouping {
				t.Errorf("expected %v, got %v", tt.expected.Grouping, result.Grouping)
			}
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

func Test_groupHeicJpgStackHeic(t *testing.T) {
	tests := []struct {
		name     string
		group    *assets.Group
		expected *assets.Group
	}{
		{
			name: "GroupByHeicJpgWithHeicFirst",
			group: &assets.Group{
				Grouping: assets.GroupByHeicJpg,
				Assets: []*assets.Asset{
					mockAsset("photo1.heic"),
					mockAsset("photo2.jpg"),
				},
			},
			expected: &assets.Group{
				Grouping: assets.GroupByHeicJpg,
				Assets: []*assets.Asset{
					mockAsset("photo1.heic"),
					mockAsset("photo2.jpg"),
				},
				CoverIndex: 0,
			},
		},
		{
			name: "GroupByHeicJpgWithHeicSecond",
			group: &assets.Group{
				Grouping: assets.GroupByHeicJpg,
				Assets: []*assets.Asset{
					mockAsset("photo1.jpg"),
					mockAsset("photo2.heic"),
				},
			},
			expected: &assets.Group{
				Grouping: assets.GroupByHeicJpg,
				Assets: []*assets.Asset{
					mockAsset("photo1.jpg"),
					mockAsset("photo2.heic"),
				},
				CoverIndex: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := groupHeicJpgStackHeic(tt.group)
			if result.Grouping != tt.expected.Grouping {
				t.Errorf("expected %v, got %v", tt.expected.Grouping, result.Grouping)
			}
			if len(result.Assets) != len(tt.expected.Assets) {
				t.Errorf("expected %v assets, got %v", len(tt.expected.Assets), len(result.Assets))
			}
			for i, asset := range result.Assets {
				if asset.File.Name() != tt.expected.Assets[i].File.Name() {
					t.Errorf("expected asset %v, got %v", tt.expected.Assets[i].File.Name(), asset.File.Name())
				}
			}
			if result.CoverIndex != tt.expected.CoverIndex {
				t.Errorf("expected cover index %v, got %v", tt.expected.CoverIndex, result.CoverIndex)
			}
		})
	}
}

func Test_groupHeicJpgStackJPG(t *testing.T) {
	tests := []struct {
		name     string
		group    *assets.Group
		expected *assets.Group
	}{
		{
			name: "GroupByHeicJpgWithJPGFirst",
			group: &assets.Group{
				Grouping: assets.GroupByHeicJpg,
				Assets: []*assets.Asset{
					mockAsset("photo1.jpg"),
					mockAsset("photo2.heic"),
				},
			},
			expected: &assets.Group{
				Grouping: assets.GroupByHeicJpg,
				Assets: []*assets.Asset{
					mockAsset("photo1.jpg"),
					mockAsset("photo2.heic"),
				},
				CoverIndex: 0,
			},
		},
		{
			name: "GroupByHeicJpgWithJPGSecond",
			group: &assets.Group{
				Grouping: assets.GroupByHeicJpg,
				Assets: []*assets.Asset{
					mockAsset("photo1.heic"),
					mockAsset("photo2.jpg"),
				},
			},
			expected: &assets.Group{
				Grouping: assets.GroupByHeicJpg,
				Assets: []*assets.Asset{
					mockAsset("photo1.heic"),
					mockAsset("photo2.jpg"),
				},
				CoverIndex: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := groupHeicJpgStackJPG(tt.group)
			if result.Grouping != tt.expected.Grouping {
				t.Errorf("expected %v, got %v", tt.expected.Grouping, result.Grouping)
			}
			if len(result.Assets) != len(tt.expected.Assets) {
				t.Errorf("expected %v assets, got %v", len(tt.expected.Assets), len(result.Assets))
			}
			for i, asset := range result.Assets {
				if asset.File.Name() != tt.expected.Assets[i].File.Name() {
					t.Errorf("expected asset %v, got %v", tt.expected.Assets[i].File.Name(), asset.File.Name())
				}
			}
			if result.CoverIndex != tt.expected.CoverIndex {
				t.Errorf("expected cover index %v, got %v", tt.expected.CoverIndex, result.CoverIndex)
			}
		})
	}
}
