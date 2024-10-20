package filenames

import (
	"testing"
	"time"

	"github.com/simulot/immich-go/internal/metadata"
)

func TestNexus(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
		info     NameInfo
	}{
		{
			name:     "BURST",
			filename: "00001IMG_00001_BURST20171111030039.jpg",
			expected: true,
			info: NameInfo{
				Radical: "BURST20171111030039",
				Base:    "00001IMG_00001_BURST20171111030039.jpg",
				IsCover: false,
				Ext:     ".jpg",
				Type:    metadata.TypeImage,
				Kind:    KindBurst,
				Index:   1,
				Taken:   time.Date(2017, 11, 11, 3, 0, 39, 0, time.Local),
			},
		},
		{
			name:     "BURST cover",
			filename: "00015IMG_00015_BURST20171111030039_COVER.jpg",
			expected: true,
			info: NameInfo{
				Radical: "BURST20171111030039",
				Base:    "00015IMG_00015_BURST20171111030039_COVER.jpg",
				IsCover: true,
				Ext:     ".jpg",
				Type:    metadata.TypeImage,
				Kind:    KindBurst,
				Index:   15,
				Taken:   time.Date(2017, 11, 11, 3, 0, 39, 0, time.Local),
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
			got, info := ic.Nexus(tt.filename)
			if got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
			if got && info != tt.info {
				t.Errorf("expected \n%+v,\n  got \n%+v", tt.info, info)
			}
		})
	}
}
