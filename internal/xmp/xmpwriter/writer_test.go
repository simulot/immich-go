package xmpwriter

import (
	"strings"
	"testing"
	"time"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/xmp/xmpreader"
)

func TestWrite(t *testing.T) {
	tc := []struct {
		path  string
		asset assets.Asset
	}{
		// {
		// 	path: "DATA/image01.jpg.xmp",
		// 	asset: assets.Asset{
		// 		Title:       "C'est une <grotte>",
		// 		Latitude:    -16.5516903372,
		// 		Longitude:   -62.6748284952,
		// 		CaptureDate: time.Time{},
		// 		Stars:       5,
		// 	},
		// },
		{
			path: "DATA/image02.jpg.xmp",
			asset: assets.Asset{
				Title:       "This a description",
				Latitude:    -16.5516903372,
				Longitude:   -62.6748284952,
				CaptureDate: time.Date(2023, 10, 10, 1, 11, 0, 0, time.FixedZone("-0400", -4*60*60)),
				Stars:       3,
			},
		},
	}

	for _, c := range tc {
		t.Run(c.path, func(t *testing.T) {
			buf := strings.Builder{}
			err := WriteXMP(&c.asset, &buf)
			if err != nil {
				t.Fatal(err.Error())
			}

			b := assets.Asset{}
			err = xmpreader.ReadXMP(&b, strings.NewReader(buf.String()))
			if b.Title != c.asset.Title {
				t.Errorf("Title: got %s, expected %s", b.Title, c.asset.Title)
			}
			if !floatIsEqual(b.Latitude, c.asset.Latitude) {
				t.Errorf("Latitude: got %f, expected %f", b.Latitude, c.asset.Latitude)
			}
			if !floatIsEqual(b.Longitude, c.asset.Longitude) {
				t.Errorf("Longitude: got %f, expected %f", b.Longitude, c.asset.Longitude)
			}
			if !b.CaptureDate.Equal(c.asset.CaptureDate) {
				t.Errorf("CaptureDate: got %v, expected %v", b.CaptureDate, c.asset.CaptureDate)
			}
			if b.Stars != c.asset.Stars {
				t.Errorf("Stars: got %d, expected %d", b.Stars, c.asset.Stars)
			}
		})
	}
}

func floatIsEqual(a, b float64) bool {
	const epsilon = 1e-6
	return (a-b) < epsilon && (b-a) < epsilon
}
