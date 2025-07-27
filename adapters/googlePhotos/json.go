package gp

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"time"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/fshelper"
)

type GoogleMetaData struct {
	Title              string             `json:"title"`
	Description        string             `json:"description"`
	Category           string             `json:"category"`
	Date               *googTimeObject    `json:"date,omitempty"`
	PhotoTakenTime     *googTimeObject    `json:"photoTakenTime"`
	GeoDataExif        *googGeoData       `json:"geoDataExif"`
	GeoData            *googGeoData       `json:"geoData"`
	Trashed            bool               `json:"trashed,omitempty"`
	Archived           bool               `json:"archived,omitempty"`
	URLPresent         googIsPresent      `json:"url,omitempty"`         // true when the file is an asset metadata
	Favorited          bool               `json:"favorited,omitempty"`   // true when starred in GP
	Enrichments        *googleEnrichments `json:"enrichments,omitempty"` // Album enrichments
	People             []Person           `json:"people,omitempty"`      // People tags
	GooglePhotosOrigin struct {
		FromPartnerSharing googIsPresent `json:"fromPartnerSharing,omitempty"` // true when this is a partner's asset
		FromSharedAlbum    googIsPresent `json:"fromSharedAlbum,omitempty"`    // true when this is from a shared album
	} `json:"googlePhotosOrigin"`
}

type Person struct {
	Name string `json:"name"`
}

func (gmd *GoogleMetaData) UnmarshalJSON(data []byte) error {
	// test the presence of the key albumData
	type md GoogleMetaData
	type album struct {
		AlbumData *md `json:"albumData"`
	}

	var t album
	err := json.Unmarshal(data, &t)
	if err == nil && t.AlbumData != nil {
		*gmd = GoogleMetaData(*(t.AlbumData))
		return nil
	}

	var gg md
	err = json.Unmarshal(data, &gg)
	if err != nil {
		return err
	}

	*gmd = GoogleMetaData(gg)
	return nil
}

func (gmd GoogleMetaData) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("Title", gmd.Title),
		slog.String("Description", gmd.Description),
		slog.String("Category", gmd.Category),
		slog.Any("Date", gmd.Date),
		slog.Any("PhotoTakenTime", gmd.PhotoTakenTime),
		slog.Any("GeoDataExif", gmd.GeoDataExif),
		slog.Any("GeoData", gmd.GeoData),
		slog.Bool("Trashed", gmd.Trashed),
		slog.Bool("Archived", gmd.Archived),
		slog.Bool("URLPresent", bool(gmd.URLPresent)),
		slog.Bool("Favorited", gmd.Favorited),
		slog.Any("Enrichments", gmd.Enrichments),
		slog.Any("People", gmd.People),
		slog.Bool("FromPartnerSharing", bool(gmd.GooglePhotosOrigin.FromPartnerSharing)),
	)
}

func (gmd GoogleMetaData) AsMetadata(name fshelper.FSAndName, tagPeople bool, flags *ImportFlags) *assets.Metadata {
	md := assets.Metadata{
		File:            name,
		FileName:        sanitizedTitle(gmd.Title),
		Description:     gmd.Description,
		Trashed:         gmd.Trashed,
		Archived:        gmd.Archived,
		Favorited:       gmd.Favorited,
		FromPartner:     gmd.isPartner(),
		FromSharedAlbum: gmd.isSharedAlbum(),
	}
	if gmd.GeoDataExif != nil {
		md.Latitude, md.Longitude = gmd.GeoDataExif.Latitude, gmd.GeoDataExif.Longitude
		if md.Latitude == 0 && md.Longitude == 0 && gmd.GeoData != nil {
			md.Latitude, md.Longitude = gmd.GeoData.Latitude, gmd.GeoData.Longitude
		}
	} else if gmd.GeoData != nil {
		md.Latitude, md.Longitude = gmd.GeoData.Latitude, gmd.GeoData.Longitude
	}

	// PhotoTakenTime is always present, but sometimes it's nul
	if gmd.PhotoTakenTime != nil && gmd.PhotoTakenTime.Timestamp != "" && gmd.PhotoTakenTime.Timestamp != "0" {
		md.DateTaken = gmd.PhotoTakenTime.Time()
	}
	if tagPeople {
		for _, p := range gmd.People {
			md.AddTag("People/" + p.Name)
		}
	}

	if flags.SharedAlbumTag && md.FromSharedAlbum {
		md.AddTag("From Shared Album")
	}

	return &md
}

func (gmd *GoogleMetaData) isAlbum() bool {
	if gmd == nil || gmd.isAsset() {
		return false
	}
	return gmd.Title != ""
}

func (gmd *GoogleMetaData) isAsset() bool {
	if gmd == nil || gmd.PhotoTakenTime == nil {
		return false
	}
	return gmd.PhotoTakenTime.Timestamp != ""
}

func (gmd *GoogleMetaData) isPartner() bool {
	if gmd == nil {
		return false
	}
	return bool(gmd.GooglePhotosOrigin.FromPartnerSharing)
}

func (gmd *GoogleMetaData) isSharedAlbum() bool {
	if gmd == nil {
		return false
	}
	return bool(gmd.GooglePhotosOrigin.FromSharedAlbum)
}

// Key return an expected unique key for the asset
// based on the title and the timestamp
func (gmd GoogleMetaData) Key() string {
	return fmt.Sprintf("%s,%s", gmd.Title, gmd.PhotoTakenTime.Timestamp)
}

// googIsPresent is set when the field is present. The content of the field is not relevant
type googIsPresent bool

func (p *googIsPresent) UnmarshalJSON(b []byte) error {
	var bl bool
	err := json.Unmarshal(b, &bl)
	if err == nil {
		return nil
	}

	*p = len(b) > 0
	return nil
}

func (p googIsPresent) MarshalJSON() ([]byte, error) {
	if p {
		return json.Marshal("present")
	}
	return json.Marshal(struct{}{})
}

// googGeoData contains GPS coordinates
type googGeoData struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Altitude  float64 `json:"altitude"`
}

func (ggd *googGeoData) LogValue() slog.Value {
	if ggd == nil {
		return slog.Value{}
	}
	return slog.GroupValue(
		slog.Float64("Latitude", ggd.Latitude),
		slog.Float64("Longitude", ggd.Longitude),
		slog.Float64("Altitude", ggd.Altitude),
	)
}

// googTimeObject to handle the epoch timestamp
type googTimeObject struct {
	Timestamp string `json:"timestamp"`
	// Formatted string    `json:"formatted"`
}

func (gt *googTimeObject) LogValue() slog.Value {
	if gt == nil {
		return slog.Value{}
	}
	return slog.TimeValue(gt.Time())
}

// Time return the time.Time of the epoch
func (gt googTimeObject) Time() time.Time {
	ts, _ := strconv.ParseInt(gt.Timestamp, 10, 64)
	if ts == 0 {
		return time.Time{}
	}
	t := time.Unix(ts, 0)
	return t.In(time.Local)
}

type googleEnrichments struct {
	Text      string
	Latitude  float64
	Longitude float64
}

func (ge *googleEnrichments) LogValue() slog.Value {
	if ge == nil {
		return slog.Value{}
	}
	return slog.GroupValue(
		slog.String("Text", ge.Text),
		slog.Float64("Latitude", ge.Latitude),
		slog.Float64("Longitude", ge.Longitude),
	)
}

func (ge *googleEnrichments) UnmarshalJSON(b []byte) error {
	type googleEnrichment struct {
		NarrativeEnrichment struct {
			Text string `json:"text"`
		} `json:"narrativeEnrichment,omitempty"`
		LocationEnrichment struct {
			Location []struct {
				Name        string `json:"name"`
				Description string `json:"description"`
				LatitudeE7  int    `json:"latitudeE7"`
				LongitudeE7 int    `json:"longitudeE7"`
			} `json:"location"`
		} `json:"locationEnrichment,omitempty"`
	}

	var enrichments []googleEnrichment

	err := json.Unmarshal(b, &enrichments)
	if err != nil {
		return err
	}

	for _, e := range enrichments {
		if e.NarrativeEnrichment.Text != "" {
			ge.Text = addString(ge.Text, "\n", e.NarrativeEnrichment.Text)
		}
		if e.LocationEnrichment.Location != nil {
			for _, l := range e.LocationEnrichment.Location {
				if l.Name != "" {
					ge.Text = addString(ge.Text, "\n", l.Name)
				}
				if l.Description != "" {
					ge.Text = addString(ge.Text, " - ", l.Description)
				}
				ge.Latitude = float64(l.LatitudeE7) / 10e6
				ge.Longitude = float64(l.LongitudeE7) / 10e6
			}
		}
	}
	return err
}

func addString(s string, sep string, t string) string {
	if s != "" {
		return s + sep + t
	}
	return t
}

func sanitizedTitle(title string) string {
	// Simple removal of commonly invalid filename characters
	return regexp.MustCompile(`[\r\n\\/:*?"<>|]`).ReplaceAllString(title, "_")
}
