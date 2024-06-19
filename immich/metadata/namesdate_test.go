package metadata

import (
	"os"
	"testing"
	"time"

	"github.com/simulot/immich-go/helpers/tzone"
)

func TestTakeTimeFromName(t *testing.T) {
	os.Setenv("TZ", "Europe/Paris")
	local, err := tzone.Local()
	if err != nil {
		t.Error(err)
		return
	}
	tests := []struct {
		name     string
		expected time.Time
	}{
		{
			name:     "PXL_20220909_154515546.TS.mp4",
			expected: time.Date(2022, 9, 9, 17, 45, 15, 0, local),
		},
		{
			name:     "Screenshot from 2022-12-17 19-45-43.png",
			expected: time.Date(2022, 12, 17, 20, 45, 43, 0, local),
		},
		// {
		// 	name:     "Bebop2_20180719194940+0200.mp4",
		// 	expected: time.Date(2018, 07, 19, 19, 49, 40, 0, local),
		// },
		{
			name:     "AR_EFFECT_20141126193511.mp4",
			expected: time.Date(2014, 11, 26, 20, 35, 11, 0, local),
		},
		{
			name:     "2023-07-20 14:15:30", // Format: YYYY-MM-DD HH:MM:SS
			expected: time.Date(2023, 7, 20, 16, 15, 30, 0, local),
		},
		{
			name:     "20001010120000", // Format: YYYYMMDDHHMMSS
			expected: time.Date(2000, 10, 10, 14, 0, 0, 0, local),
		},
		{
			name:     "2023_07_20_10_09_20.mp4",
			expected: time.Date(2023, 0o7, 20, 12, 9, 20, 0, local),
		},
		{
			name:     "19991231",
			expected: time.Date(1999, 12, 31, 1, 0, 0, 0, local),
		},
		{
			name:     "991231-125200",
			expected: time.Time{},
		},
		{
			name:     "20223112-125200",
			expected: time.Time{},
		},
		{
			name:     "00015IMG_00015_BURST20171111030039_COVER.jpg",
			expected: time.Date(2017, 11, 11, 4, 0, 39, 0, local),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TakeTimeFromName(tt.name); !got.Equal(tt.expected) {
				t.Errorf("TakeTimeFromName() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTakeTimeFromPath(t *testing.T) {
	os.Setenv("TZ", "Europe/Paris")
	local, err := tzone.Local()
	if err != nil {
		t.Error(err)
		return
	}
	tests := []struct {
		name     string
		expected time.Time
	}{
		{
			name:     "A/B/PXL_20220909_154515546.TS.mp4",
			expected: time.Date(2022, 9, 9, 17, 45, 15, 0, local),
		},
		{
			name:     "A/B/IMG_1234.HEIC",
			expected: time.Time{},
		},
		{
			name:     "A/20221109/IMG_1234.HEIC",
			expected: time.Date(2022, 11, 9, 1, 0, 0, 0, local),
		},
		{
			name:     "A/20221109T2030/IMG_1234.HEIC",
			expected: time.Date(2022, 11, 9, 21, 30, 0, 0, local),
		},
		{
			name:     "A/2022.11.09T20.30/IMG_1234.HEIC",
			expected: time.Date(2022, 11, 9, 21, 30, 0, 0, local),
		},
		{
			name:     "A/2022.11.09T20.30/IMG_1234.HEIC",
			expected: time.Date(2022, 11, 9, 21, 30, 0, 0, local),
		},
		{
			name:     "A/2022/2022.11/2022.11.09/IMG_1234.HEIC",
			expected: time.Date(2022, 11, 9, 1, 0, 0, 0, local),
		},
		{
			name:     "A/2022/11/09/IMG_1234.HEIC",
			expected: time.Date(2022, 11, 9, 1, 0, 0, 0, local),
		},
		{
			name:     "/volume1/media/IPAD_20171113_video_nuotraukos/IMG_1552.JPG",
			expected: time.Date(2017, 11, 13, 1, 0, 0, 0, local),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TakeTimeFromPath(tt.name); !got.Equal(tt.expected) {
				t.Errorf("TakeTimeFromPath() = %v, want %v", got, tt.expected)
			}
		})
	}
}
