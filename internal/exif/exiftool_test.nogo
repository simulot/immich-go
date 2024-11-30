package exif

import (
	"testing"
	"time"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/tzone"
)

func TestExifTool_ReadMetaData(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		want     *assets.Metadata
		wantErr  bool
	}{
		{
			name:     "read JPG",
			fileName: "DATA/PXL_20231006_063000139.jpg",
			want: &assets.Metadata{
				DateTaken: time.Date(2023, 10, 6, 6, 29, 56, 0, time.UTC), // 2023:10:06 06:29:56Z
				Latitude:  +48.8583736,
				Longitude: +2.2919010,
			},
			wantErr: false,
		},
		{
			name:     "read mp4",
			fileName: "DATA/PXL_20220724_210650210.NIGHT.mp4",
			want: &assets.Metadata{
				DateTaken: time.Date(2022, 7, 24, 21, 10, 56, 0, time.Local),
				Latitude:  47.538300,
				Longitude: -2.891900,
			},
			wantErr: false,
		},
		{
			name:     "read OLYMPUS",
			fileName: "DATA/YG816507.jpg",
			want: &assets.Metadata{
				DateTaken: time.Date(2024, 7, 7, 19, 37, 7, 0, time.UTC), // 2024:07:07 19:37:07Z
			},
			wantErr: false,
		},
		{
			name:     "read OLYMPUS orf",
			fileName: "DATA/YG816507.orf",
			want: &assets.Metadata{
				DateTaken: time.Date(2024, 7, 7, 19, 37, 7, 0, time.UTC), // 2024:07:07 19:37:07Z
			},
			wantErr: false,
		},
	}
	flag := &ExifToolFlags{
		UseExifTool: true,
		Timezone:    tzone.Timezone{TZ: time.Local},
	}
	err := NewExifTool(flag)
	if err != nil {
		t.Error(err)
		return
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := flag.et.ReadMetaData(tt.fileName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExifTool.ReadMetaData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !got.DateTaken.Equal(tt.want.DateTaken) {
				t.Errorf("DateTaken = %v, want %v", got.DateTaken, tt.want.DateTaken)
			}
			if !float64Equal(got.Latitude, tt.want.Latitude) {
				t.Errorf("Latitude = %v, want %v", got.Latitude, tt.want.Latitude)
			}
			if !float64Equal(got.Longitude, tt.want.Longitude) {
				t.Errorf("Longitude = %v, want %v", got.Longitude, tt.want.Longitude)
			}
		})
	}
}

func float64Equal(a, b float64) bool {
	const epsilon = 1e-6
	return (a-b) < epsilon && (b-a) < epsilon
}
