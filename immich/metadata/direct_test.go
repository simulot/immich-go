//go:build e2e
// +build e2e

package metadata

import (
	"immich-go/helpers/tzone"
	"os"
	"path"
	"testing"
	"time"
)

func mustParse(s string) time.Time {
	local, err := tzone.Local()
	if err != nil {
		panic(err)
	}
	t, err := time.ParseInLocation("2006:01:02 15:04:05-07:00", s, local)
	if err != nil {
		panic(err)
	}
	return t
}

func TestGetFromReader(t *testing.T) {

	tests := []struct {
		name     string
		filename string
		want     time.Time
	}{
		{
			name:     "cr3",
			filename: "../../../test-data/burst/Reflex/3H2A0018.CR3",
			want:     mustParse("2023:06:23 13:32:52+02:00"),
		},
		{
			name:     "jpg",
			filename: "../../../test-data/burst/Reflex/3H2A0018.JPG",
			want:     mustParse("2023:06:23 13:32:52+02:00"),
		},
		{
			name:     "jpg",
			filename: "../../../test-data/burst/PXL6/PXL_20231029_062723981.jpg",
			want:     mustParse("2023:10:29 07:27:23+01:00"),
		},
		{
			name:     "dng",
			filename: "../../../test-data/burst/PXL6/PXL_20231029_062723981.dng",
			want:     mustParse("2023:10:29 07:27:24+01:00"),
		},
		{
			name:     "cr2",
			filename: "../../../test-data/burst/IMG_4879.CR2",
			want:     mustParse("2023:02:24 18:59:09+01:00"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := os.Open(tt.filename)
			if err != nil {
				t.Error(err)
				return
			}
			defer r.Close()
			ext := path.Ext(tt.filename)
			got, err := GetFromReader(r, ext)
			if err != nil {
				t.Error(err)
				return
			}
			if !tt.want.Equal(got.DateTaken) {
				t.Errorf("GetFromReader() = %v, want %v", got.DateTaken, tt.want)
			}
		})
	}
}
