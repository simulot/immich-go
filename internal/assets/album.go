package assets

import (
	"log/slog"
)

type Album struct {
	ID          string  `json:"-"`                     // The album ID
	Title       string  `json:"title,omitempty"`       // either the directory base name, or metadata
	Description string  `json:"description,omitempty"` // As found in the metadata
	Latitude    float64 `json:"latitude,omitempty"`    // As found in the metadata
	Longitude   float64 `json:"longitude,omitempty"`   // As found in the metadata
}

func NewAlbum(id string, title string, description string) Album {
	return Album{
		ID:          id,
		Title:       title,
		Description: description,
	}
}

func (a Album) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("title", a.Title),
		slog.String("description", a.Description),
		slog.Float64("latitude", a.Latitude),
		slog.Float64("longitude", a.Longitude),
	)
}
