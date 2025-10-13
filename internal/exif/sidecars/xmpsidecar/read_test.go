package xmpsidecar

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/simulot/immich-go/internal/assets"
)

func TestReadAttributedProperties(t *testing.T) {
	const source = `<x:xmpmeta xmlns:x="adobe:ns:meta/">
 <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
  <rdf:Description xmlns:exif="http://ns.adobe.com/exif/1.0/"
   xmlns:xmp="http://ns.adobe.com/xap/1.0/" xmlns:digiKam="http://www.digikam.org/ns/1.0/">
   <exif:DateTimeOriginal xml:lang="x-default">2020-02-02T10:20:30Z</exif:DateTimeOriginal>
   <xmp:Rating xml:lang="x-default">3</xmp:Rating>
   <exif:GPSLatitude xml:lang="x-default">48,30N</exif:GPSLatitude>
   <exif:GPSLongitude xml:lang="x-default">3,30W</exif:GPSLongitude>
   <digiKam:TagsList><rdf:Seq>
    <rdf:li xml:lang="x-default">one/tag</rdf:li>
    <rdf:li xml:lang="x-default"></rdf:li>
    <rdf:li xml:lang="x-default">two/tag</rdf:li>
   </rdf:Seq></digiKam:TagsList>
  </rdf:Description>
 </rdf:RDF>
</x:xmpmeta>`
	md := &assets.Metadata{}
	if err := ReadXMP(strings.NewReader(source), md); err != nil {
		t.Fatal(err)
	}
	if md.Rating != 3 || md.Latitude != 48.5 || md.Longitude != -3.5 ||
		!md.DateTaken.Equal(time.Date(2020, 2, 2, 10, 20, 30, 0, time.UTC)) {
		t.Fatalf("attributed metadata was not read: %+v", md)
	}
	if len(md.Tags) != 2 || md.Tags[0].Value != "one/tag" || md.Tags[1].Value != "two/tag" {
		t.Fatalf("attributed tags were not read or empty tag was retained: %+v", md.Tags)
	}
}

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
		{
			// several tags in one list, properties spread over several rdf:Description elements
			path: "DATA/multi-tags.jpg.xmp",
			expect: assets.Metadata{
				Description: "Three of us",
				Latitude:    48.408376,
				Longitude:   -3.090590,
				Tags: []assets.Tag{
					{Value: "ari", Name: "ari"},
					{Value: "baldur", Name: "baldur"},
					{Value: "people/jakob", Name: "jakob"},
				},
			},
		},
		{
			// all properties in a single rdf:Description element
			path: "DATA/single-description.jpg.xmp",
			expect: assets.Metadata{
				Description: "All in one",
				DateTaken:   time.Date(2020, 2, 2, 10, 20, 30, 0, time.UTC),
				Rating:      3,
				Latitude:    48.408376,
				Longitude:   -3.090590,
				Tags: []assets.Tag{
					{Value: "solo", Name: "solo"},
				},
			},
		},
		{
			// the description in several languages: the x-default entry comes first
			path: "DATA/multi-language.jpg.xmp",
			expect: assets.Metadata{
				Description: "Sunset over the bay",
			},
		},
		{
			// an empty keyword from a template that renders to nothing, like osxphotos
			// "{favorite?Favorit,}" for a non-favorite photo (issue #1451)
			path: "DATA/empty-keyword.jpg.xmp",
			expect: assets.Metadata{
				Tags: []assets.Tag{
					{Value: "favorit", Name: "favorit"},
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
