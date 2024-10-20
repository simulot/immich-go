package filenames

import (
	"testing"
	"time"
)

func TestTakeTimeFromPath(t *testing.T) {
	tests := []struct {
		name     string
		expected time.Time
	}{
		{
			name:     "2024.png",
			expected: time.Time{},
		},
		{
			name:     "2024-05.png",
			expected: time.Time{},
		},
		{
			name:     "A/B/2022/2022.11/2022.11.09/IMG_1234.HEIC",
			expected: time.Date(2022, 11, 9, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "A/B/2022/2022.11/IMG_1234.HEIC",
			expected: time.Time{},
		},
		{
			name:     "A/B/2022.11.09/2022.11/2022/IMG_1234.HEIC",
			expected: time.Date(2022, 11, 9, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "2024-05-05.png",
			expected: time.Date(2024, 5, 5, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "PXL_20220909_154515546.TS.mp4",
			expected: time.Date(2022, 9, 9, 15, 45, 15, 0, time.UTC),
		},
		{
			name:     "Screenshot from 2022-12-17 19-45-43.png",
			expected: time.Date(2022, 12, 17, 19, 45, 43, 0, time.UTC),
		},
		{
			name:     "Bebop2_20180719194940+0200.mp4", // It's local time anyway, so ignore +0200 part
			expected: time.Date(2018, 0o7, 19, 19, 49, 40, 0, time.UTC),
		},
		{
			name:     "AR_EFFECT_20141126193511.mp4",
			expected: time.Date(2014, 11, 26, 19, 35, 11, 0, time.UTC),
		},
		{
			name:     "2023-07-20 14:15:30", // Format: YYYY-MM-DD HH:MM:SS
			expected: time.Date(2023, 7, 20, 14, 15, 30, 0, time.UTC),
		},
		{
			name:     "20001010120000", // Format: YYYYMMDDHHMMSS
			expected: time.Date(2000, 10, 10, 12, 0, 0, 0, time.UTC),
		},
		{
			name:     "2023_07_20_10_09_20.mp4",
			expected: time.Date(2023, 0o7, 20, 10, 9, 20, 0, time.UTC),
		},
		{
			name:     "19991231",
			expected: time.Date(1999, 12, 31, 0, 0, 0, 0, time.UTC),
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
			expected: time.Date(2017, 11, 11, 3, 0, 39, 0, time.UTC),
		},
		{
			name:     "PXL_20220909_154515546.TS.mp4",
			expected: time.Date(2022, 9, 9, 15, 45, 15, 0, time.UTC),
		},
		{
			name:     "IMG_1234.HEIC",
			expected: time.Time{},
		},
		{
			name:     "20221109/IMG_1234.HEIC",
			expected: time.Date(2022, 11, 9, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "20221109T2030/IMG_1234.HEIC",
			expected: time.Date(2022, 11, 9, 20, 30, 0, 0, time.UTC),
		},
		{
			name:     "2022.11.09T20.30/IMG_1234.HEIC",
			expected: time.Date(2022, 11, 9, 20, 30, 0, 0, time.UTC),
		},
		{
			name:     "2022/11/09/IMG_1234.HEIC",
			expected: time.Date(2022, 11, 9, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "something_2011-05-11 something/IMG_1234.JPG",
			expected: time.Date(2011, 0o5, 11, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TakeTimeFromPath(tt.name, time.UTC); !got.Equal(tt.expected) {
				t.Errorf("TakeTimeFromPath() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func BenchmarkTakeTimeFromPathPath(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TakeTimeFromPath("2022/2022.11/2022.11.09/IMG_1234.HEIC", time.UTC)
	}
}

func BenchmarkTakeTimeFromName(b *testing.B) {
	for i := 0; i < b.N; i++ {
		TakeTimeFromName("PXL_20220909_154515546.TS.mp4", time.UTC)
	}
}
