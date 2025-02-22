package filenames

import (
	"reflect"
	"testing"
	"time"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/filetypes"
)

func normalizeTime(t time.Time) time.Time {
	return t.Round(0).UTC()
}

func TestGetInfo(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
		info     assets.NameInfo
	}{
		{
			name:     "Normal",
			filename: "PXL_20231026_210642603.dng",
			expected: true,
			info: assets.NameInfo{
				Radical: "PXL_20231026_210642603",
				Base:    "PXL_20231026_210642603.dng",
				IsCover: false,
				Ext:     ".dng",
				Type:    filetypes.TypeImage,
				Taken:   time.Date(2023, 10, 26, 21, 6, 42, 0, time.UTC),
			},
		},
		{
			name:     "Nexus BURST cover",
			filename: "00015IMG_00015_BURST20171111030039_COVER.jpg",
			expected: true,
			info: assets.NameInfo{
				Radical: "BURST20171111030039",
				Base:    "00015IMG_00015_BURST20171111030039_COVER.jpg",
				IsCover: true,
				Ext:     ".jpg",
				Type:    filetypes.TypeImage,
				Kind:    assets.KindBurst,
				Index:   15,
				Taken:   time.Date(2017, 11, 11, 3, 0, 39, 0, time.Local),
			},
		},
		{
			name:     "Samsung BURST",
			filename: "20231207_101605_031.jpg",
			expected: true,
			info: assets.NameInfo{
				Radical: "20231207_101605",
				Base:    "20231207_101605_031.jpg",
				IsCover: false,
				Ext:     ".jpg",
				Type:    filetypes.TypeImage,
				Kind:    assets.KindBurst,
				Index:   31,
				Taken:   time.Date(2023, 12, 7, 10, 16, 5, 0, time.Local),
			},
		},
		{
			name:     "Regular",
			filename: "IMG_20171111_030128.jpg",
			expected: false,
			info: assets.NameInfo{
				Radical: "IMG_20171111_030128",
				Base:    "IMG_20171111_030128.jpg",
				Ext:     ".jpg",
				Type:    filetypes.TypeImage,
				Taken:   time.Date(2017, 11, 11, 3, 1, 28, 0, time.Local),
			},
		},
		{
			name:     "Sony Xperia BURST",
			filename: "DSC_0001_BURST20230709220904977.JPG",
			expected: true,
			info: assets.NameInfo{
				Radical: "BURST20230709220904977",
				Base:    "DSC_0001_BURST20230709220904977.JPG",
				IsCover: false,
				Ext:     ".jpg",
				Type:    filetypes.TypeImage,
				Kind:    assets.KindBurst,
				Index:   1,
				Taken:   time.Date(2023, 7, 9, 22, 9, 4, int(977*time.Millisecond), time.Local),
			},
		},
		{
			name:     "#743 Nexus BURST cover with unix timestamp",
			filename: "00001IMG_00001_BURST1723801037429_COVER.jpg",
			expected: true,
			info: assets.NameInfo{
				Radical: "BURST1723801037429",
				Base:    "00001IMG_00001_BURST1723801037429_COVER.jpg",
				IsCover: true,
				Ext:     ".jpg",
				Type:    filetypes.TypeImage,
				Kind:    assets.KindBurst,
				Index:   1,
				Taken:   time.UnixMilli(1723801037429),
			},
		},
		{
			name:     "#743 Nexus BURST cover with unix timestamp",
			filename: "00002IMG_00002_BURST1723801037429.jpg",
			expected: true,
			info: assets.NameInfo{
				Radical: "BURST1723801037429",
				Base:    "00002IMG_00002_BURST1723801037429.jpg",
				IsCover: false,
				Ext:     ".jpg",
				Type:    filetypes.TypeImage,
				Kind:    assets.KindBurst,
				Index:   2,
				Taken:   time.UnixMilli(1723801037429),
			},
		},
		{
			name:     "InvalidFilename",
			filename: "IMG_1123.jpg",
			expected: false,
			info: assets.NameInfo{
				Base:    "IMG_1123.jpg",
				Radical: "IMG_1123",
				Ext:     ".jpg",
				Type:    filetypes.TypeImage,
			},
		},
	}

	ic := InfoCollector{
		TZ: time.Local,
		SM: filetypes.DefaultSupportedMedia,
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := ic.GetInfo(tt.filename)

			// Normalize time fields
			info.Taken = normalizeTime(info.Taken)
			tt.info.Taken = normalizeTime(tt.info.Taken)

			if !reflect.DeepEqual(info, tt.info) {
				t.Errorf("expected \n%+v,\n  got \n%+v", tt.info, info)
			}
		})
	}
}
