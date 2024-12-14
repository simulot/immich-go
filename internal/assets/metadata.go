package assets

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/simulot/immich-go/internal/fshelper"
)

type Metadata struct {
	File        fshelper.FSAndName `json:"-"`                     // File name and file system that holds the metadata. Could be empty
	FileName    string             `json:"fileName,omitempty"`    // File name as presented to users
	Latitude    float64            `json:"latitude,omitempty"`    // GPS
	Longitude   float64            `json:"longitude,omitempty"`   // GPS
	DateTaken   time.Time          `json:"dateTaken,omitempty"`   // Date of exposure
	Description string             `json:"description,omitempty"` // Long description
	Albums      []Album            `json:"albums,omitempty"`      // Used to list albums that contain the file
	Tags        []Tag              `json:"tags,omitempty"`        // Used to list tags
	Rating      byte               `json:"rating,omitempty"`      // 0 to 5
	Trashed     bool               `json:"trashed,omitempty"`     // Flag to indicate if the image has been trashed
	Archived    bool               `json:"archived,omitempty"`    // Flag to indicate if the image has been archived
	Favorited   bool               `json:"favorited,omitempty"`   // Flag to indicate if the image has been favorited
	FromPartner bool               `json:"fromPartner,omitempty"` // Flag to indicate if the image is from a partner
}

func (m Metadata) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Float64("latitude", m.Latitude),
		slog.Float64("longitude", m.Longitude),
		slog.Any("fileName", m.File),
		slog.Time("dateTaken", m.DateTaken),
		slog.String("description", m.Description),
		slog.Int("rating", int(m.Rating)),
		slog.Bool("trashed", m.Trashed),
		slog.Bool("archived", m.Archived),
		slog.Bool("favorited", m.Favorited),
		slog.Bool("fromPartner", m.FromPartner),
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