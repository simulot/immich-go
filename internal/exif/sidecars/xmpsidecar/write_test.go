package xmpsidecar

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/simulot/immich-go/internal/assets"
)

func TestWrite(t *testing.T) {
	tc := []struct {
		name     string
		metadata assets.Metadata
		validate func(t *testing.T, output string)
	}{
		{
			name: "basic fields",
			metadata: assets.Metadata{
				Description: "Test description",
				DateTaken:   time.Date(2023, 10, 15, 14, 30, 0, 0, time.UTC),
				Rating:      4,
			},
			validate: func(t *testing.T, output string) {
				if !strings.Contains(output, "Test description") {
					t.Error("expected description in output")
				}
				if !strings.Contains(output, "2023-10-15T14:30:00Z") {
					t.Error("expected date in output")
				}
				if !strings.Contains(output, "<xmp:Rating>4</xmp:Rating>") {
					t.Error("expected rating in output")
				}
			},
		},
		{
			name: "GPS coordinates",
			metadata: assets.Metadata{
				Latitude:  48.8583736,
				Longitude: 2.291901,
			},
			validate: func(t *testing.T, output string) {
				if !strings.Contains(output, "exif:GPSLatitude") {
					t.Error("expected GPS latitude in output")
				}
				if !strings.Contains(output, "exif:GPSLongitude") {
					t.Error("expected GPS longitude in output")
				}
				// Verify positive latitude has N
				if !strings.Contains(output, "N") {
					t.Error("expected N for positive latitude")
				}
				// Verify positive longitude has E
				if !strings.Contains(output, "E") {
					t.Error("expected E for positive longitude")
				}
			},
		},
		{
			name: "negative GPS coordinates",
			metadata: assets.Metadata{
				Latitude:  -33.8688,
				Longitude: -151.2093,
			},
			validate: func(t *testing.T, output string) {
				// Verify negative latitude has S
				if !strings.Contains(output, "S") {
					t.Error("expected S for negative latitude")
				}
				// Verify negative longitude has W
				if !strings.Contains(output, "W") {
					t.Error("expected W for negative longitude")
				}
			},
		},
		{
			name: "tags",
			metadata: assets.Metadata{
				Tags: []assets.Tag{
					{Name: "outdoors", Value: "activities/outdoors"},
					{Name: "travel", Value: "travel"},
				},
			},
			validate: func(t *testing.T, output string) {
				if !strings.Contains(output, "digiKam:TagsList") {
					t.Error("expected TagsList in output")
				}
				if !strings.Contains(output, "activities/outdoors") {
					t.Error("expected tag value in output")
				}
				if !strings.Contains(output, "travel") {
					t.Error("expected second tag in output")
				}
			},
		},
		{
			name: "albums as tags",
			metadata: assets.Metadata{
				Albums: []assets.Album{
					{Title: "Vacation 2023"},
					{Title: "Family"},
				},
			},
			validate: func(t *testing.T, output string) {
				if !strings.Contains(output, "Albums/Vacation 2023") {
					t.Error("expected album as tag with Albums/ prefix")
				}
				if !strings.Contains(output, "Albums/Family") {
					t.Error("expected second album as tag")
				}
			},
		},
		{
			name: "favorited sets rating to 5",
			metadata: assets.Metadata{
				Favorited: true,
				Rating:    0, // No explicit rating
			},
			validate: func(t *testing.T, output string) {
				if !strings.Contains(output, "<xmp:Rating>5</xmp:Rating>") {
					t.Error("expected rating 5 for favorited item")
				}
			},
		},
		{
			name: "favorited does not override explicit rating",
			metadata: assets.Metadata{
				Favorited: true,
				Rating:    3,
			},
			validate: func(t *testing.T, output string) {
				if !strings.Contains(output, "<xmp:Rating>3</xmp:Rating>") {
					t.Error("expected original rating 3 to be preserved")
				}
				if strings.Contains(output, "<xmp:Rating>5</xmp:Rating>") {
					t.Error("should not override explicit rating with 5")
				}
			},
		},
		{
			name: "empty metadata",
			metadata: assets.Metadata{},
			validate: func(t *testing.T, output string) {
				// Should still have XMP header
				if !strings.Contains(output, "<?xpacket") {
					t.Error("expected xpacket header")
				}
				if !strings.Contains(output, "x:xmpmeta") {
					t.Error("expected xmpmeta element")
				}
				// Should not have any content blocks
				if strings.Contains(output, "dc:description") {
					t.Error("should not have description block for empty metadata")
				}
				if strings.Contains(output, "digiKam:TagsList") {
					t.Error("should not have tags block for empty metadata")
				}
			},
		},
		{
			name: "special characters in description",
			metadata: assets.Metadata{
				Description: "Test with <special> & \"characters\"",
			},
			validate: func(t *testing.T, output string) {
				// Should be XML escaped
				if strings.Contains(output, "<special>") {
					t.Error("special characters should be escaped")
				}
				if !strings.Contains(output, "&lt;special&gt;") {
					t.Error("expected < and > to be escaped")
				}
				if !strings.Contains(output, "&amp;") {
					t.Error("expected & to be escaped")
				}
			},
		},
	}

	for _, c := range tc {
		t.Run(c.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := Write(&c.metadata, &buf)
			if err != nil {
				t.Fatalf("Write failed: %v", err)
			}
			output := buf.String()
			c.validate(t, output)
		})
	}
}

func TestWriteReadRoundTrip(t *testing.T) {
	tc := []struct {
		name     string
		metadata assets.Metadata
	}{
		{
			name: "basic roundtrip",
			metadata: assets.Metadata{
				Description: "Test description",
				DateTaken:   time.Date(2023, 10, 15, 14, 30, 0, 0, time.UTC),
				Rating:      4,
				Tags: []assets.Tag{
					{Name: "outdoors", Value: "activities/outdoors"},
				},
			},
		},
		{
			name: "GPS roundtrip",
			metadata: assets.Metadata{
				Latitude:  48.8583736,
				Longitude: 2.291901,
			},
		},
		{
			name: "negative GPS roundtrip",
			metadata: assets.Metadata{
				Latitude:  -33.8688,
				Longitude: -151.2093,
			},
		},
	}

	for _, c := range tc {
		t.Run(c.name, func(t *testing.T) {
			// Write
			var buf bytes.Buffer
			err := Write(&c.metadata, &buf)
			if err != nil {
				t.Fatalf("Write failed: %v", err)
			}

			// Read back
			readMd := &assets.Metadata{}
			err = ReadXMP(bytes.NewReader(buf.Bytes()), readMd)
			if err != nil {
				t.Fatalf("ReadXMP failed: %v", err)
			}

			// Compare
			if c.metadata.Description != "" && readMd.Description != c.metadata.Description {
				t.Errorf("description mismatch: expected %q, got %q", c.metadata.Description, readMd.Description)
			}
			if !c.metadata.DateTaken.IsZero() && !readMd.DateTaken.Equal(c.metadata.DateTaken) {
				t.Errorf("date mismatch: expected %v, got %v", c.metadata.DateTaken, readMd.DateTaken)
			}
			if c.metadata.Rating != 0 && readMd.Rating != c.metadata.Rating {
				t.Errorf("rating mismatch: expected %d, got %d", c.metadata.Rating, readMd.Rating)
			}
			if c.metadata.Latitude != 0 && !floatIsEqual(readMd.Latitude, c.metadata.Latitude) {
				t.Errorf("latitude mismatch: expected %f, got %f", c.metadata.Latitude, readMd.Latitude)
			}
			if c.metadata.Longitude != 0 && !floatIsEqual(readMd.Longitude, c.metadata.Longitude) {
				t.Errorf("longitude mismatch: expected %f, got %f", c.metadata.Longitude, readMd.Longitude)
			}
			if len(c.metadata.Tags) > 0 && len(readMd.Tags) != len(c.metadata.Tags) {
				t.Errorf("tags count mismatch: expected %d, got %d", len(c.metadata.Tags), len(readMd.Tags))
			}
		})
	}
}
