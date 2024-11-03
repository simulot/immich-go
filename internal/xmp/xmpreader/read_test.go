package xmpreader

import (
	"os"
	"testing"
	"time"

	"github.com/simulot/immich-go/internal/assets"
)

func TestRead(t *testing.T) {
	tc := []struct {
		path   string
		expect assets.Asset
	}{
		{
			path: "DATA/image01.jpg.xmp",
			expect: assets.Asset{
				Title:       "C'est une <grotte>",
				Latitude:    -16.5516903372,
				Longitude:   -62.6748284952,
				CaptureDate: time.Time{},
				Albums: []assets.Album{
					{Title: "Vacation 2024"},
					{Title: "Family Reunion"},
				},
				Stars: 5,
			},
		},
		{
			path: "DATA/image02.jpg.xmp",
			expect: assets.Asset{
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
			r, err := os.Open(c.path)
			if err != nil {
				t.Fatal(err.Error())
			}
			defer r.Close()
			a := &assets.Asset{}
			err = ReadXMP(a, r)
			if err != nil {
				t.Fatal(err.Error())
			}
			if a.Title != c.expect.Title {
				t.Errorf("Title: got %s, expected %s", a.Title, c.expect.Title)
			}
			if !floatIsEqual(a.Latitude, c.expect.Latitude) {
				t.Errorf("Latitude: got %f, expected %f", a.Latitude, c.expect.Latitude)
			}
			if !floatIsEqual(a.Longitude, c.expect.Longitude) {
				t.Errorf("Longitude: got %f, expected %f", a.Longitude, c.expect.Longitude)
			}
			if !a.CaptureDate.Equal(c.expect.CaptureDate) {
				t.Errorf("CaptureDate: got %v, expected %v", a.CaptureDate, c.expect.CaptureDate)
			}
			if a.Stars != c.expect.Stars {
				t.Errorf("Stars: got %d, expected %d", a.Stars, c.expect.Stars)
			}
			if len(a.Albums) != len(c.expect.Albums) {
				t.Errorf("Albums: got %d, expected %d", len(a.Albums), len(c.expect.Albums))
			} else {
				for i, album := range a.Albums {
					if album.Title != c.expect.Albums[i].Title {
						t.Errorf("Album %d: got %s, expected %s", i, album.Title, c.expect.Albums[i].Title)
					}
				}
			}
		})
	}
}

func floatIsEqual(a, b float64) bool {
	const epsilon = 1e-6
	return (a-b) < epsilon && (b-a) < epsilon
}

/*

// explore the map
func exploreMap(m mxj.Map, padding string) {
	for key, value := range m {
		switch v := value.(type) {
		case map[string]interface{}:
			fmt.Printf("%skey: %s, value: map\n", padding, key)
			exploreMap(v, padding+"  ")
		case []interface{}:
			fmt.Printf("%skey: %s, value: array\n", padding, key)
			for i, item := range v {
				fmt.Printf("%s  index: %d\n", padding, i)
				if itemMap, ok := item.(map[string]interface{}); ok {
					exploreMap(itemMap, padding+"    ")
				} else {
					fmt.Printf("%s  value: %v\n", padding, item)
				}
			}
		default:
			fmt.Printf("%skey: %s, value: %v\n", padding, key, value)
		}
	}
}
*/
