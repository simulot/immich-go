package assets

import (
	"log/slog"

	"github.com/simulot/immich-go/internal/gen/syncset"
)

type Album struct {
	ID          string               `json:"-"`                     // The album ID
	Title       string               `json:"title,omitempty"`       // either the directory base name, or metadata
	Description string               `json:"description,omitempty"` // As found in the metadata
	Latitude    float64              `json:"latitude,omitempty"`    // As found in the metadata
	Longitude   float64              `json:"longitude,omitempty"`   // As found in the metadata
	Assets      *syncset.Set[string] `json:"-"`                     // The assets in the album
}

func NewAlbum(id string, title string, description string) *Album {
	return &Album{
		ID:          id,
		Title:       title,
		Description: description,
		Assets:      syncset.NewSet[string](),
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

func (a *Album) AddAsset(assetID string) bool {
	return a.Assets.Add(assetID)
}

func (a *Album) AddAssets(assetIDs ...string) {
	for _, assetID := range assetIDs {
		a.Assets.Add(assetID)
	}
}

func (a *Album) RemoveAsset(assetID string) {
	a.Assets.Remove(assetID)
}

func (a *Album) ContainsAsset(assetID string) bool {
	return a.Assets.Contains(assetID)
}

func (a *Album) AssetIDs() []string {
	return a.Assets.Items()
}
