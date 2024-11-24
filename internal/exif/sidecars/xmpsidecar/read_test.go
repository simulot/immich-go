package xmpsidecar

import (
	"os"
	"testing"
	"time"

	"github.com/simulot/immich-go/internal/assets"
)

func TestRead(t *testing.T) {
	tc := []struct {
		path   string
		expect assets.Metadata
	}{
		{
			path: "DATA/159d9172-2a1e-4d95-aef1-b5133549927b.jpg.xmp",
			expect: assets.Metadata{
				Description: "Alors!",
				DateTaken:   time.Date(2018, 8, 11, 17, 38, 25, 0, time.UTC),
				Rating:      4,
				Tags: []assets.Tag{
					{Value: "activities/outdoors", Name: "outdoors"},
				},
			},
		},
		{
			path: "DATA/IMG_2477.CR2.xmp",
			expect: assets.Metadata{
				Latitude:  48.408376,
				Longitude: -3.090590,
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
			md := &assets.Metadata{}
			err = ReadXMP(r, md)
			if err != nil {
				t.Fatal(err.Error())
			}
			if md.Description != c.expect.Description {
				t.Errorf("expected description %s, got %s", c.expect.Description, md.Description)
			}
			if !md.DateTaken.Equal(c.expect.DateTaken) {
				t.Errorf("expected date taken %s, got %s", c.expect.DateTaken, md.DateTaken)
			}
			if md.Rating != c.expect.Rating {
				t.Errorf("expected rating %d, got %d", c.expect.Rating, md.Rating)
			}
			if len(md.Tags) != len(c.expect.Tags) {
				t.Errorf("expected %d tags, got %d", len(c.expect.Tags), len(md.Tags))
			} else {
				for i, tag := range md.Tags {
					if tag != c.expect.Tags[i] {
						t.Errorf("expected tag %v, got %v", c.expect.Tags[i], tag)
					}
				}
			}
			if !floatIsEqual(md.Latitude, c.expect.Latitude) {
				t.Errorf("expected latitude %f, got %f", c.expect.Latitude, md.Latitude)
			}
			if !floatIsEqual(md.Longitude, c.expect.Longitude) {
				t.Errorf("expected longitude %f, got %f", c.expect.Longitude, md.Longitude)
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
