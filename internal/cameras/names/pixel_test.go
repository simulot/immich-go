package names_test

import (
	"testing"
	"time"

	"github.com/simulot/immich-go/internal/cameras/names"
	"github.com/simulot/immich-go/internal/metadata"
)

func TestPixel(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
		info     names.NameInfo
	}{
		{
			name:     "Normal",
			filename: "PXL_20231026_210642603.dng",
			expected: true,
			info: names.NameInfo{
				Radical: "PXL_20231026_210642603",
				Base:    "PXL_20231026_210642603.dng",
				IsCover: false,
				Ext:     ".dng",
				Type:    metadata.TypeImage,
				Taken:   time.Date(2023, 10, 26, 21, 6, 42, 0, time.UTC),
			},
		},
		{
			name:     "RawJpg",
			filename: "PXL_20231207_032111247.RAW-02.ORIGINAL.dng",
			expected: true,
			info: names.NameInfo{
				Radical: "PXL_20231207_032111247",
				Base:    "PXL_20231207_032111247.RAW-02.ORIGINAL.dng",
				IsCover: false,
				Ext:     ".dng",
				Type:    metadata.TypeImage,
				Kind:    names.KindRawJpg,
				Index:   2,
				Taken:   time.Date(2023, 12, 7, 3, 21, 11, 0, time.UTC),
			},
		},
		{
			name:     "RawJpg Cover",
			filename: "PXL_20231207_032111247.RAW-01.COVER.jpg",
			expected: true,
			info: names.NameInfo{
				Radical: "PXL_20231207_032111247",
				Base:    "PXL_20231207_032111247.RAW-01.COVER.jpg",
				IsCover: true,
				Ext:     ".jpg",
				Type:    metadata.TypeImage,
				Kind:    names.KindRawJpg,
				Index:   1,
				Taken:   time.Date(2023, 12, 7, 3, 21, 11, 0, time.UTC),
			},
		},
		{
			name:     "MotionCover",
			filename: "PXL_20230330_184138390.MOTION-01.COVER.jpg",
			expected: true,
			info: names.NameInfo{
				Radical: "PXL_20230330_184138390",
				Base:    "PXL_20230330_184138390.MOTION-01.COVER.jpg",
				IsCover: true,
				Ext:     ".jpg",
				Type:    metadata.TypeImage,
				Kind:    names.KindMotion,
				Index:   1,
				Taken:   time.Date(2023, 3, 30, 18, 41, 38, 0, time.UTC),
			},
		},
		{
			name:     "LONG_EXPOSURE_COVER",
			filename: "PXL_20230809_203029471.LONG_EXPOSURE-01.COVER.jpg",
			expected: true,
			info: names.NameInfo{
				Radical: "PXL_20230809_203029471",
				Base:    "PXL_20230809_203029471.LONG_EXPOSURE-01.COVER.jpg",
				IsCover: true,
				Ext:     ".jpg",
				Type:    metadata.TypeImage,
				Kind:    names.KindLongExposure,
				Index:   1,
				Taken:   time.Date(2023, 8, 9, 20, 30, 29, 0, time.UTC),
			},
		},
		{
			name:     "InvalidFilename",
			filename: "IMG_1123.jpg",
			expected: false,
			info:     names.NameInfo{},
		},
	}

	ic := names.InfoCollector{
		TZ: time.UTC,
		SM: metadata.DefaultSupportedMedia,
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, info := ic.Pixel(tt.filename)
			if got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
			if got && info != tt.info {
				t.Errorf("expected \n%+v,\n  got \n%+v", tt.info, info)
			}
		})
	}
}
