package assets

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"path"
	"time"

	"github.com/simulot/immich-go/internal/fshelper"
)

type Metadata struct {
	File            fshelper.FSAndName `json:"-"`                         // File name and file system that holds the metadata. Could be empty
	FileName        string             `json:"fileName,omitempty"`        // File name as presented to users
	Latitude        float64            `json:"latitude,omitempty"`        // GPS
	Longitude       float64            `json:"longitude,omitempty"`       // GPS
	FileDate        time.Time          `json:"fileDate,omitzero"`         // Date of the file
	DateTaken       time.Time          `json:"dateTaken,omitzero"`        // Date of exposure
	Description     string             `json:"description,omitempty"`     // Long description
	Albums          []Album            `json:"albums,omitempty"`          // Used to list albums that contain the file
	Tags            []Tag              `json:"tags,omitempty"`            // Used to list tags
	Rating          byte               `json:"rating,omitempty"`          // 0 to 5
	Trashed         bool               `json:"trashed,omitempty"`         // Flag to indicate if the image has been trashed
	Archived        bool               `json:"archived,omitempty"`        // Flag to indicate if the image has been archived
	Favorited       bool               `json:"favorited,omitempty"`       // Flag to indicate if the image has been favorited
	FromPartner     bool               `json:"fromPartner,omitempty"`     // Flag to indicate if the image is from a partner
	FromSharedAlbum bool               `json:"fromSharedAlbum,omitempty"` // Flag to indicate if the image is from a shared album
}

func (m Metadata) LogValue() slog.Value {
	var gpsGroup slog.Value
	if m.Latitude != 0 || m.Longitude != 0 {
		gpsGroup = slog.GroupValue(
			slog.String("latitude", fmt.Sprintf("%0.f.xxxx", m.Latitude)),
			slog.String("longitude", fmt.Sprintf("%0.f.xxxx", m.Longitude)),
		)
	}

	return slog.GroupValue(
		slog.Any("GPS coordinates", gpsGroup),
		slog.Any("fileName", m.File),
		slog.Time("dateTaken", m.DateTaken),
		slog.String("description", m.Description),
		slog.Int("rating", int(m.Rating)),
		slog.Bool("trashed", m.Trashed),
		slog.Bool("archived", m.Archived),
		slog.Bool("favorited", m.Favorited),
		slog.Bool("fromPartner", m.FromPartner),
		slog.Bool("fromSharedAlbum", m.FromSharedAlbum),
		slog.Any("albums", m.Albums),
		slog.Any("tags", m.Tags),
	)
}

func (m Metadata) IsSet() bool {
	return m.Description != "" || !m.DateTaken.IsZero() || m.Latitude != 0 || m.Longitude != 0
}

func UnMarshalMetadata(data []byte) (*Metadata, error) {
	var m Metadata
	err := json.Unmarshal(data, &m)
	return &m, err
}

func (m *Metadata) AddTag(tag string) {
	for _, t := range m.Tags {
		if t.Value == tag {
			return
		}
	}
	m.Tags = append(m.Tags, Tag{Name: path.Base(tag), Value: tag})
}
