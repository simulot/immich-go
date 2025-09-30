package cliflags

import (
	"encoding/json"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestDateRange_InRange(t *testing.T) {
	tests := []struct {
		name  string
		check []struct {
			date string
			want bool
		}
	}{
		{
			name: "2017-08-07,2017-09-07",
			check: []struct {
				date string
				want bool
			}{
				{
					date: "2017-08-31 17:55:20",
					want: true,
				},
				{
					date: "2017-08-07 00:00:00",
					want: true,
				},
				{
					date: "2017-09-07 23:59:59",
					want: true,
				},
				{
					date: "2017-01-31 07:50:00",
					want: false,
				},
				{
					date: "2017-09-08 00:00:00",
					want: false,
				},
				{
					date: "2017-12-01 00:00:00",
					want: false,
				},
			},
		},

		{
			name: "2017-08-31",
			check: []struct {
				date string
				want bool
			}{
				{
					date: "2017-08-31 17:55:20",
					want: true,
				},
				{
					date: "2017-08-31 00:00:00",
					want: true,
				},
				{
					date: "2017-08-31 23:59:59",
					want: true,
				},
				{
					date: "2017-01-31 07:50:00",
					want: false,
				},
				{
					date: "2017-09-01 00:00:00",
					want: false,
				},
				{
					date: "2017-12-01 00:00:00",
					want: false,
				},
			},
		},
		{
			name: "2017-08",
			check: []struct {
				date string
				want bool
			}{
				{
					date: "2017-08-31 17:55:20",
					want: true,
				},
				{
					date: "2017-08-01 00:00:00",
					want: true,
				},
				{
					date: "2017-08-31 23:59:59",
					want: true,
				},
				{
					date: "2017-01-31 07:50:00",
					want: false,
				},
				{
					date: "2017-09-01 00:00:00",
					want: false,
				},
				{
					date: "2017-12-01 00:00:00",
					want: false,
				},
			},
		},
		{
			name: "2017",
			check: []struct {
				date string
				want bool
			}{
				{
					date: "2017-08-31 17:55:20",
					want: true,
				},
				{
					date: "2017-01-01 00:00:00",
					want: true,
				},
				{
					date: "2017-12-31 23:59:59",
					want: true,
				},
				{
					date: "2016-12-31 23:59:00",
					want: false,
				},
				{
					date: "2018-01-01 00:00:00",
					want: false,
				},
				{
					date: "2018-12-01 00:00:00",
					want: false,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tz := time.Local
			var dr DateRange
			dr.SetTZ(tz)
			err := dr.Set(tt.name)
			if err != nil {
				t.Errorf("set DateRange %q fails: %s", tt.name, err)
			}
			if dr.String() != tt.name {
				t.Errorf("the String() gives %q, want %q", dr.String(), tt.name)
			}
			for _, check := range tt.check {
				d, err := time.ParseInLocation(time.DateTime, check.date, tz)
				if err != nil {
					t.Errorf("can't parse check date %q fails: %s", check.date, err)
				}
				if got := dr.InRange(d); got != check.want {
					t.Errorf("DateRange.InRange(%q) = %v, want %v", check.date, got, check.want)
				}
			}
		})
	}
}

func TestDateRange_JSONMarshaling(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expected     string
		canUnmarshal bool
	}{
		{
			name:         "date range",
			input:        "2023-01-01,2023-12-31",
			expected:     `"2023-01-01,2023-12-31"`,
			canUnmarshal: true,
		},
		{
			name:         "single day",
			input:        "2023-06-15",
			expected:     `"2023-06-15"`,
			canUnmarshal: true,
		},
		{
			name:         "month",
			input:        "2023-06",
			expected:     `"2023-06"`,
			canUnmarshal: true,
		},
		{
			name:         "year",
			input:        "2023",
			expected:     `"2023"`,
			canUnmarshal: true,
		},
		{
			name:         "unset",
			input:        "",
			expected:     `"unset"`,
			canUnmarshal: false, // "unset" is not a valid input format
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dr DateRange
			if tt.input != "" {
				err := dr.Set(tt.input)
				if err != nil {
					t.Fatalf("failed to set DateRange: %v", err)
				}
			}

			// Test marshaling
			data, err := json.Marshal(dr)
			if err != nil {
				t.Fatalf("failed to marshal DateRange: %v", err)
			}

			if string(data) != tt.expected {
				t.Errorf("MarshalJSON() = %q, want %q", string(data), tt.expected)
			}

			// Test unmarshaling only if it's a valid input format
			if tt.canUnmarshal {
				var dr2 DateRange
				err = json.Unmarshal(data, &dr2)
				if err != nil {
					t.Fatalf("failed to unmarshal DateRange: %v", err)
				}

				if dr.String() != dr2.String() {
					t.Errorf("UnmarshalJSON() resulted in %q, want %q", dr2.String(), dr.String())
				}
			}
		})
	}
}

func TestDateRange_YAMLMarshaling(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expected     string
		canUnmarshal bool
	}{
		{
			name:         "date range",
			input:        "2023-01-01,2023-12-31",
			expected:     "2023-01-01,2023-12-31\n",
			canUnmarshal: true,
		},
		{
			name:         "single day",
			input:        "2023-06-15",
			expected:     "\"2023-06-15\"\n", // YAML quotes strings that look like dates
			canUnmarshal: true,
		},
		{
			name:         "month",
			input:        "2023-06",
			expected:     "2023-06\n", // YAML does not quote this format
			canUnmarshal: true,
		},
		{
			name:         "year",
			input:        "2023",
			expected:     "\"2023\"\n", // YAML quotes strings that look like dates
			canUnmarshal: true,
		},
		{
			name:         "unset",
			input:        "",
			expected:     "unset\n",
			canUnmarshal: false, // "unset" is not a valid input format
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dr DateRange
			if tt.input != "" {
				err := dr.Set(tt.input)
				if err != nil {
					t.Fatalf("failed to set DateRange: %v", err)
				}
			}

			// Test marshaling
			data, err := yaml.Marshal(dr)
			if err != nil {
				t.Fatalf("failed to marshal DateRange: %v", err)
			}

			if string(data) != tt.expected {
				t.Errorf("MarshalYAML() = %q, want %q", string(data), tt.expected)
			}

			// Test unmarshaling only if it's a valid input format
			if tt.canUnmarshal {
				var dr2 DateRange
				err = yaml.Unmarshal(data, &dr2)
				if err != nil {
					t.Fatalf("failed to unmarshal DateRange: %v", err)
				}

				if dr.String() != dr2.String() {
					t.Errorf("UnmarshalYAML() resulted in %q, want %q", dr2.String(), dr.String())
				}
			}
		})
	}
}

func TestDateRange_TextMarshaling(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expected     string
		canUnmarshal bool
	}{
		{
			name:         "date range",
			input:        "2023-01-01,2023-12-31",
			expected:     "2023-01-01,2023-12-31",
			canUnmarshal: true,
		},
		{
			name:         "single day",
			input:        "2023-06-15",
			expected:     "2023-06-15",
			canUnmarshal: true,
		},
		{
			name:         "month",
			input:        "2023-06",
			expected:     "2023-06",
			canUnmarshal: true,
		},
		{
			name:         "year",
			input:        "2023",
			expected:     "2023",
			canUnmarshal: true,
		},
		{
			name:         "unset",
			input:        "",
			expected:     "unset",
			canUnmarshal: false, // "unset" is not a valid input format
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dr DateRange
			if tt.input != "" {
				err := dr.Set(tt.input)
				if err != nil {
					t.Fatalf("failed to set DateRange: %v", err)
				}
			}

			// Test marshaling
			data, err := dr.MarshalText()
			if err != nil {
				t.Fatalf("failed to marshal DateRange: %v", err)
			}

			if string(data) != tt.expected {
				t.Errorf("MarshalText() = %q, want %q", string(data), tt.expected)
			}

			// Test unmarshaling only if it's a valid input format
			if tt.canUnmarshal {
				var dr2 DateRange
				err = dr2.UnmarshalText(data)
				if err != nil {
					t.Fatalf("failed to unmarshal DateRange: %v", err)
				}

				if dr.String() != dr2.String() {
					t.Errorf("UnmarshalText() resulted in %q, want %q", dr2.String(), dr.String())
				}
			}
		})
	}
}
