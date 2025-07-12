//go:build !windows

package immich_test

import (
	"testing"
	"time"

	"github.com/simulot/immich-go/immich"
)

func TestImmichExifTimeUnmarshalJSON_Valid(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Milliseconds .000+00:00", `"2024-06-01T12:34:56.789+00:00"`, `2024-06-01T12:34:56.789+00:00`},
		{"Hundredths .00+00:00", `"2024-06-01T12:34:56.78+00:00"`, `2024-06-01T12:34:56.780+00:00`},
		{"Tenths .0+00:00", `"2024-06-01T12:34:56.7+00:00"`, `2024-06-01T12:34:56.700+00:00`},
		{"No fraction +00:00", `"2024-06-01T12:34:56+00:00"`, `2024-06-01T12:34:56.000+00:00`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected, _ := time.ParseInLocation(
				"2006-01-02T15:04:05.000+00:00",
				tt.expected,
				time.UTC,
			)
			expected = expected.In(time.Local)

			var et immich.ImmichExifTime
			err := et.UnmarshalJSON([]byte(tt.input))
			if err != nil {
				t.Fatalf("unexpected error for input %s: %v", tt.input, err)
			}
			if !et.Time.Equal(expected) {
				t.Errorf("for input %s: expected %v, got %v", tt.input, expected, et.Time)
			}
		})
	}
}

func TestImmichExifTimeUnmarshalJSON_Invalid(t *testing.T) {
	tests := []string{
		`""`,
		`"not-a-date"`,
		`"2024-06-01T12:34:56.000Z"`, // unsupported format for this implementation
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			var et immich.ImmichExifTime
			err := et.UnmarshalJSON([]byte(input))
			if err != nil {
				t.Fatalf("unexpected error for input %s: %v", input, err)
			}
			if !et.Time.IsZero() {
				t.Errorf("expected zero time for input %s, got %v", input, et.Time)
			}
		})
	}
}
