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
				Title:       "This is a title",
				CaptureDate: time.Date(2023, 10, 10, 1, 11, 0, 0, time.FixedZone("-0400", -4*60*60)),
				Favorite:    true,
				Rating:      3,
				Albums: []assets.Album{
					{
						Title: "Vacation 2024", Description: "Vacation 2024 hawaii and more",
						Latitude: 19.820610, Longitude: -155.473254,
					},
					{Title: "Family Reunion", Latitude: 48.858370, Longitude: 2.291901},
				},
			},
		},
		{
			path: "DATA/image02.jpg.xmp",
			expect: assets.Asset{
				Latitude:  -16.5516903372,
				Longitude: -62.6748284952,
				Rating:    5,
				Tags: []assets.Tag{
					{Name: "tag2", Value: "tag1/tag2"},
				},
			},
		},
		{
			path: "DATA/image03.jpg.xmp",
			expect: assets.Asset{
				Latitude:  -16.5516903372,
				Longitude: -62.6748284952,
				Rating:    5,
				Albums: []assets.Album{
					{Title: "Vacation 2024"},
				},
				Tags: []assets.Tag{
					{Name: "tag2", Value: "tag1/tag2"},
					{Name: "tag3", Value: "tag1/tag3"},
				},
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
			if a.Rating != c.expect.Rating {
				t.Errorf("Stars: got %d, expected %d", a.Rating, c.expect.Rating)
			}
			if len(a.Albums) != len(c.expect.Albums) {
				t.Errorf("Albums: got %d, expected %d", len(a.Albums), len(c.expect.Albums))
			} else {
				for i, album := range a.Albums {
					if album.Title != c.expect.Albums[i].Title {
						t.Errorf("Album %d: got %s, expected %s", i, album.Title, c.expect.Albums[i].Title)
					}
					if album.Description != c.expect.Albums[i].Description {
						t.Errorf("Album %d: got %s, expected %s", i, album.Description, c.expect.Albums[i].Description)
					}
					if !floatIsEqual(album.Latitude, c.expect.Albums[i].Latitude) {
						t.Errorf("Album %d: Latitude: got %f, expected %f", i, album.Latitude, c.expect.Albums[i].Latitude)
					}
					if !floatIsEqual(album.Longitude, c.expect.Albums[i].Longitude) {
						t.Errorf("Album %d: Longitude: got %f, expected %f", i, album.Longitude, c.expect.Albums[i].Longitude)
					}
				}
			}
			if len(a.Tags) != len(c.expect.Tags) {
				t.Errorf("Tags: got %d, expected %d", len(a.Tags), len(c.expect.Tags))
			} else {
				for i, tag := range a.Tags {
					if tag.Name != c.expect.Tags[i].Name {
						t.Errorf("Tag %d: Name: got %s, expected %s", i, tag.Name, c.expect.Tags[i].Name)
					}
					if tag.Value != c.expect.Tags[i].Value {
						t.Errorf("Tag %d: Value: got %s, expected %s", i, tag.Value, c.expect.Tags[i].Value)
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
