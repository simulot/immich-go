package assets

import (
	"reflect"
	"testing"
	"time"
)

func TestTakeTimeFromName(t *testing.T) {
	tests := []struct {
		name     string
		expected time.Time
	}{
		{
			name:     "PXL_20220909_154515546.TS.mp4",
			expected: time.Date(2022, 9, 9, 15, 45, 15, 0, time.UTC),
		},
		{
			name:     "Screenshot from 2022-12-17 19-45-43.png",
			expected: time.Date(2022, 12, 17, 19, 45, 43, 0, time.UTC),
		},
		{
			name:     "Bebop2_20180719194940+0200.mp4",
			expected: time.Date(2018, 07, 19, 19, 49, 40, 0, time.UTC),
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
			expected: time.Date(2023, 07, 20, 10, 9, 20, 0, time.UTC),
		},
		{
			name:     "19991231",
			expected: time.Time{},
		},
		{
			name:     "991231-125200",
			expected: time.Time{},
		},
		{
			name:     "20223112-125200",
			expected: time.Time{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TakeTimeFromName(tt.name); !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("GuessTimeTakeOnName() = %v, want %v", got, tt.expected)
			}
		})
	}
}
