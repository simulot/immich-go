package gp

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/metadata"
	"github.com/simulot/immich-go/internal/tzone"
)

type Metablock struct {
	Title          string             `json:"title"`
	Description    string             `json:"description"`
	Category       string             `json:"category"`
	Date           *googTimeObject    `json:"date,omitempty"`
	PhotoTakenTime googTimeObject     `json:"photoTakenTime"`
	GeoDataExif    googGeoData        `json:"geoDataExif"`
	GeoData        googGeoData        `json:"geoData"`
	Trashed        bool               `json:"trashed,omitempty"`
	Archived       bool               `json:"archived,omitempty"`
	URLPresent     googIsPresent      `json:"url,omitempty"`         // true when the file is an asset metadata
	Favorited      bool               `json:"favorited,omitempty"`   // true when starred in GP
	Enrichments    *googleEnrichments `json:"enrichments,omitempty"` // Album enrichments
}

type GoogleMetaData struct {
	Metablock
	GooglePhotosOrigin struct {
		FromPartnerSharing googIsPresent `json:"fromPartnerSharing,omitempty"` // true when this is a partner's asset
	} `json:"googlePhotosOrigin"`
	AlbumData *Metablock `json:"albumdata"`
	// Not in the JSON, for local treatment
	foundInPaths []fileevent.FileAndName //  keep track of paths where the json has been found
}

func (gmd GoogleMetaData) AsMetadata() *metadata.Metadata {
	latitude, longitude := gmd.GeoDataExif.Latitude, gmd.GeoDataExif.Longitude
	if latitude == 0 && longitude == 0 {
		latitude, longitude = gmd.GeoData.Latitude, gmd.GeoData.Longitude
	}

	return &metadata.Metadata{
		FileName:    gmd.Title,
		Description: gmd.Description,
		DateTaken:   gmd.PhotoTakenTime.Time(),
		Latitude:    latitude,
		Longitude:   longitude,
		Trashed:     gmd.Trashed,
		Archived:    gmd.Archived,
		Favorited:   gmd.Favorited,
		FromPartner: gmd.isPartner(),
	}
}

func (gmd *GoogleMetaData) UnmarshalJSON(data []byte) error {
	type gmetadata GoogleMetaData
	var gg gmetadata

	err := json.Unmarshal(data, &gg)
	if err != nil {
		return err
	}

	// compensate metadata version
	if gg.AlbumData != nil {
		gg.Metablock = *gg.AlbumData
		gg.AlbumData = nil
	}

	*gmd = GoogleMetaData(gg)
	return nil
}

func (gmd GoogleMetaData) isAlbum() bool {
	return gmd.Date != nil
}

func (gmd GoogleMetaData) isAsset() bool {
	return gmd.PhotoTakenTime.Timestamp != ""
}

func (gmd GoogleMetaData) isPartner() bool {
	return bool(gmd.GooglePhotosOrigin.FromPartnerSharing)
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

// googTimeObject to handle the epoch timestamp
type googTimeObject struct {
	Timestamp string `json:"timestamp"`
	// Formatted string    `json:"formatted"`
}

// Time return the time.Time of the epoch
func (gt *googTimeObject) Time() time.Time {
	if gt == nil {
		return time.Time{}
	}
	ts, _ := strconv.ParseInt(gt.Timestamp, 10, 64)
	if ts == 0 {
		return time.Time{}
	}
	t := time.Unix(ts, 0)
	local, _ := tzone.Local()
	//	t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC)
	return t.In(local)
}

type googleEnrichments struct {
	Text      string
	Latitude  float64
	Longitude float64
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
