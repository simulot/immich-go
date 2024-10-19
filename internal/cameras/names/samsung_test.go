package names_test

import (
	"testing"
	"time"

	"github.com/simulot/immich-go/internal/cameras/names"
	"github.com/simulot/immich-go/internal/metadata"
)

func TestSamsung(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
		info     names.NameInfo
	}{
		{
			name:     "BURST COVER",
			filename: "20231207_101605_001.jpg",
			expected: true,
			info: names.NameInfo{
				Radical: "20231207_101605",
				Base:    "20231207_101605_001.jpg",
				IsCover: true,
				Ext:     ".jpg",
				Type:    metadata.TypeImage,
				Kind:    names.KindBurst,
				Index:   1,
				Taken:   time.Date(2023, 12, 7, 10, 16, 5, 0, time.Local),
			},
		},
		{
			name:     "BURST",
			filename: "20231207_101605_031.jpg",
			expected: true,
			info: names.NameInfo{
				Radical: "20231207_101605",
				Base:    "20231207_101605_031.jpg",
				IsCover: false,
				Ext:     ".jpg",
				Type:    metadata.TypeImage,
				Kind:    names.KindBurst,
				Index:   31,
				Taken:   time.Date(2023, 12, 7, 10, 16, 5, 0, time.Local),
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
		TZ: time.Local,
		SM: metadata.DefaultSupportedMedia,
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, info := ic.Samsung(tt.filename)
			if got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
			if got && info != tt.info {
				t.Errorf("expected \n%+v,\n  got \n%+v", tt.info, info)
			}
		})
	}
}
