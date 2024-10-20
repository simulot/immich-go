package filenames

import (
	"testing"
	"time"

	"github.com/simulot/immich-go/internal/metadata"
)

func TestHuawei(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
		info     NameInfo
	}{
		{
			name:     "BURSTCOVER",
			filename: "IMG_20231014_183246_BURST001_COVER.jpg",
			expected: true,
			info: NameInfo{
				Radical: "IMG_20231014_183246",
				Base:    "IMG_20231014_183246_BURST001_COVER.jpg",
				IsCover: true,
				Ext:     ".jpg",
				Type:    metadata.TypeImage,
				Kind:    KindBurst,
				Index:   1,
				Taken:   time.Date(2023, 10, 14, 18, 32, 46, 0, time.Local),
			},
		},
		{
			name:     "BURST",
			filename: "IMG_20231014_183246_BURST002.jpg",
			expected: true,
			info: NameInfo{
				Radical: "IMG_20231014_183246",
				Base:    "IMG_20231014_183246_BURST002.jpg",
				IsCover: false,
				Ext:     ".jpg",
				Type:    metadata.TypeImage,
				Kind:    KindBurst,
				Index:   2,
				Taken:   time.Date(2023, 10, 14, 18, 32, 46, 0, time.Local),
			},
		},

		{
			name:     "InvalidFilename",
			filename: "IMG_1123.jpg",
			expected: false,
			info:     NameInfo{},
		},
	}

	ic := InfoCollector{
		TZ: time.Local,
		SM: metadata.DefaultSupportedMedia,
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, info := ic.Huawei(tt.filename)
			if got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
			if got && info != tt.info {
				t.Errorf("expected \n%+v,\n  got \n%+v", tt.info, info)
			}
		})
	}
}
