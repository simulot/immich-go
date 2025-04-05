package exif

import (
	"os"
	"testing"
	"time"

	"github.com/simulot/immich-go/internal/assets"
)

func Test_MetadataFromDirectRead(t *testing.T) {
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
				DateTaken: time.Date(2023, 10, 6, 8, 30, 0, int(139*time.Millisecond), time.Local), // 2023:10:06 06:29:56Z
				Latitude:  +48.8583736,
				Longitude: +2.2919010,
			},
			wantErr: false,
		},
		{
			name:     "read mp4",
			fileName: "DATA/PXL_20220724_210650210.NIGHT.mp4",
			want: &assets.Metadata{
				DateTaken: time.Date(2022, 7, 24, 21, 10, 56, 0, time.UTC),
				// Latitude:  47.538300,
				// Longitude: -2.891900,
			},
			// 	wantErr: false,
		},
		{
			name:     "read OLYMPUS",
			fileName: "DATA/YG816507.jpg",
			want: &assets.Metadata{
				DateTaken: time.Date(2024, 7, 8, 4, 35, 7, 0, time.Local),
			},
			wantErr: false,
		},
		{
			name:     "read OLYMPUS orf",
			fileName: "DATA/YG816507.orf",
			want: &assets.Metadata{
				DateTaken: time.Date(2024, 7, 7, 19, 37, 7, 0, time.UTC),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := os.Open(tt.fileName)
			if err != nil {
				t.Errorf("Can't open file %s: %v", tt.fileName, err)
				return
			}
			defer f.Close()
			got, err := MetadataFromDirectRead(f, tt.fileName, time.Local)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExifTool.ReadMetaData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if !tt.want.DateTaken.IsZero() && !got.DateTaken.Equal(tt.want.DateTaken) {
				t.Errorf("DateTaken = %v, want %v", got.DateTaken, tt.want.DateTaken)
			}
			if !floatEquals(got.Latitude, tt.want.Latitude, 1e-6) {
				t.Errorf("Latitude = %v, want %v", got.Latitude, tt.want.Latitude)
			}
			if !floatEquals(got.Longitude, tt.want.Longitude, 1e-6) {
				t.Errorf("Longitude = %v, want %v", got.Longitude, tt.want.Longitude)
			}
		})
	}
}

func Test_parseExifTime(t *testing.T) {
	tests := []struct {
		name     string
		date     string
		location *time.Location
		want     time.Time
		wantErr  bool
	}{
		{
			name:     "valid date with different separators in local timezone",
			date:     "2023-10-06 08:30:00",
			location: time.Local,
			want:     time.Date(2023, 10, 6, 8, 30, 0, 0, time.Local),
			wantErr:  false,
		},
		{
			name:     "valid date with milliseconds in local timezone",
			date:     "2023-10-06 08:30:00.123",
			location: time.Local,
			want:     time.Date(2023, 10, 6, 8, 30, 0, int(123*time.Millisecond), time.Local),
			wantErr:  false,
		},
		{
			name:     "valid date with different separators in UTC timezone",
			date:     "2023-10-06 08:30:00",
			location: time.UTC,
			want:     time.Date(2023, 10, 6, 8, 30, 0, 0, time.UTC),
			wantErr:  false,
		},
		{
			name:     "valid date with milliseconds in UTC timezone",
			date:     "2023/10/06 08:30:00.123",
			location: time.UTC,
			want:     time.Date(2023, 10, 6, 8, 30, 0, int(123*time.Millisecond), time.UTC),
			wantErr:  false,
		},
		{
			name:     "date format with different separators",
			date:     "2023/10/06 08:30:00",
			location: time.Local,
			want:     time.Date(2023, 10, 6, 8, 30, 0, 0, time.Local),
			wantErr:  false,
		},
		{
			name:     "date format with milliseconds",
			date:     "2023/10/06 08:30:00.123",
			location: time.Local,
			want:     time.Date(2023, 10, 6, 8, 30, 0, int(123*time.Millisecond), time.Local),
			wantErr:  false,
		},
		{
			name:     "empty date string",
			date:     "",
			location: time.Local,
			want:     time.Time{},
			wantErr:  true,
		},
		{
			name:     "date string with extra spaces",
			date:     "  2023-10-06 08:30:00  ",
			location: time.Local,
			want:     time.Date(2023, 10, 6, 8, 30, 0, 0, time.Local),
			wantErr:  false,
		},
		{
			name:     "date string with extra spaces and milliseconds",
			date:     "  2023-10-06 08:30:00.123  ",
			location: time.Local,
			want:     time.Date(2023, 10, 6, 8, 30, 0, int(123*time.Millisecond), time.Local),
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseExifTime(tt.date, tt.location)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseExifTime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.want.IsZero() && !got.Equal(tt.want) {
				t.Errorf("parseExifTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func floatEquals(a, b, epsilon float64) bool {
	return (a-b) < epsilon && (b-a) < epsilon
}
