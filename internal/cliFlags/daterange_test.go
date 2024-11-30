package cliflags

import (
	"testing"
	"time"
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
