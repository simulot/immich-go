package filenames

import (
	"testing"
	"time"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/filetypes"
)

func TestSonyXperia(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
		info     assets.NameInfo
	}{
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
			name:     "Sony Xperia BURST cover",
			filename: "DSC_0052_BURST20230709220904977_COVER.JPG",
			expected: true,
			info: assets.NameInfo{
				Radical: "BURST20230709220904977",
				Base:    "DSC_0052_BURST20230709220904977_COVER.JPG",
				IsCover: true,
				Ext:     ".jpg",
				Type:    filetypes.TypeImage,
				Kind:    assets.KindBurst,
				Index:   52,
				Taken:   time.Date(2023, 7, 9, 22, 9, 4, int(977*time.Millisecond), time.Local),
			},
		},
		{
			name:     "InvalidFilename",
			filename: "IMG_1123.jpg",
			expected: false,
			info:     assets.NameInfo{},
		},
	}

	ic := InfoCollector{
		TZ: time.Local,
		SM: filetypes.DefaultSupportedMedia,
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, info := ic.SonyXperia(tt.filename)
			if got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
			if got && info != tt.info {
				t.Errorf("expected \n%+v,\n  got \n%+v", tt.info, info)
			}
		})
	}
}
