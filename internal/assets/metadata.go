package assets

import (
	"log/slog"
	"time"

	"github.com/simulot/immich-go/internal/fshelper"
)

type Metadata struct {
	File        fshelper.FSAndName // File name and file system that holds the metadata. Could be empty
	FileName    string             // File name as presented to users
	Latitude    float64            // GPS
	Longitude   float64            // GPS
	DateTaken   time.Time          // Date of exposure
	Description string             // Long description
	Albums      []Album            // Used to list albums that contain the file
	Tags        []Tag              // Used to list tags
	Rating      byte               // 0 to 5
	Trashed     bool               // Flag to indicate if the image has been trashed
	Archived    bool               // Flag to indicate if the image has been archived
	Favorited   bool               // Flag to indicate if the image has been favorited
	FromPartner bool               // Flag to indicate if the image is from a partner
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
