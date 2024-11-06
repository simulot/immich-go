package xmpwriter

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/xmp/xmpreader"
)

func TestStructure(t *testing.T) {
	s := &XmpMeta{
		NS: []xml.Attr{
			{Name: xml.Name{Local: "xmlns:x"}, Value: "adobe:ns:meta/"},
			{Name: xml.Name{Local: "xmlns:rdf"}, Value: "http://www.w3.org/1999/02/22-rdf-syntax-ns#"},
			{Name: xml.Name{Local: "xmlns:dc"}, Value: "http://purl.org/dc/elements/1.1/"},
			{Name: xml.Name{Local: "xmlns:exif"}, Value: "http://ns.adobe.com/exif/1.0/"},
			{Name: xml.Name{Local: "xmlns:xmp"}, Value: "http://ns.adobe.com/xap/1.0/"},
			{Name: xml.Name{Local: "xmlns:tiff"}, Value: "http://ns.adobe.com/tiff/1.0/"},
			{Name: xml.Name{Local: "xmlns:digikam"}, Value: "http://www.digikam.org/ns/1.0/"},
			{Name: xml.Name{Local: "xmlns:immichgo"}, Value: "http://ns.immich-go.com/immich-go/1.0/"},
		},
		Xmptk: "immich-go",
		RDF: RDF{
			Descriptions: []Descriptioner{
				ImmichGoProperties{
					XMLName:          xml.Name{Local: "immichgo:ImmichGoProperties"},
					DateTimeOriginal: "2023-10-10T01:11:00-04:00",
					Trashed:          "False",
					Archived:         "False",
					FromPartner:      "True",
					Favorite:         "True",
					Rating:           "5",
					Latitude:         "-16.5516903372",
					Longitude:        "-62.6748284952",
					Albums: &ImmichAlbums{
						Albums: Bag{
							Li: []lier{
								ImmichAlbum{Title: "Vacation 2024", Description: "Vacation 2024 hawaii and more"},
								ImmichAlbum{Title: "Family Reunion", Description: "Family Reunion with grand parents"},
							},
						},
					},
				},
			},
		},
	}

	buf, err := xml.MarshalIndent(s, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout.Write(buf)
	fmt.Println()
}

func TestWrite(t *testing.T) {
	tc := []struct {
		path  string
		asset assets.Asset
	}{
		{
			path: "DATA/image01.jpg.xmp",
			asset: assets.Asset{
				Title:       "THIS IS A TITLE with special characters & < >",
				Latitude:    -16.5516903372,
				Longitude:   -62.6748284952,
				CaptureDate: time.Time{},
				Rating:      5,
			},
		},
		{
			path: "DATA/image02.jpg.xmp",
			asset: assets.Asset{
				Title:       "This a description",
				Latitude:    0,
				Longitude:   0,
				CaptureDate: time.Date(2023, 10, 10, 1, 11, 0, 0, time.FixedZone("-0400", -4*60*60)),
				Rating:      3,
				Albums: []assets.Album{
					{Title: "Vacation 2024", Description: "Vacation 2024 hawaii and more", Latitude: 19.8206101, Longitude: -155.4732542},
					{Title: "Family Reunion", Latitude: 48.8583701, Longitude: 2.291901},
				},
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
			// debug: fmt.Println(buf.String())
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
			if b.Rating != c.asset.Rating {
				t.Errorf("Stars: got %d, expected %d", b.Rating, c.asset.Rating)
			}
			if len(b.Albums) != len(c.asset.Albums) {
				t.Errorf("Albums: got %d, expected %d", len(b.Albums), len(c.asset.Albums))
			} else {
				for i := range b.Albums {
					if b.Albums[i].Title != c.asset.Albums[i].Title {
						t.Errorf("Album %d: got %s, expected %s", i, b.Albums[i].Title, c.asset.Albums[i].Title)
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
