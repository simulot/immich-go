package immich

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/simulot/immich-go/internal/assets"
)

type AlbumSimplified struct {
	ID          string `json:"id,omitempty"`
	AlbumName   string `json:"albumName"`
	Description string `json:"description,omitempty"`
	// OwnerID                    string    `json:"ownerId"`
	// CreatedAt                  time.Time `json:"createdAt"`
	// UpdatedAt                  time.Time `json:"updatedAt"`
	// AlbumThumbnailAssetID      string    `json:"albumThumbnailAssetId"`
	// SharedUsers                []string  `json:"sharedUsers"`
	// Owner                      User      `json:"owner"`
	// Shared                     bool      `json:"shared"`
	// AssetCount                 int       `json:"assetCount"`
	// LastModifiedAssetTimestamp time.Time `json:"lastModifiedAssetTimestamp"
	AssetIds []string `json:"assetIds,omitempty"`
}

func AlbumsFromAlbumSimplified(albums []AlbumSimplified) []assets.Album {
	result := make([]assets.Album, 0, len(albums))
	for _, a := range albums {
		result = append(result, assets.Album{
			ID:          a.ID,
			Title:       a.AlbumName,
			Description: a.Description,
		})
	}
	return result
}

func (ic *ImmichClient) GetAllAlbums(ctx context.Context) ([]AlbumSimplified, error) {
	var albums []AlbumSimplified
	err := ic.newServerCall(ctx, EndPointGetAllAlbums).
		do(
			getRequest("/albums", setAcceptJSON()),
			responseJSON(&albums),
		)
	if err != nil {
		return nil, err
	}
	return albums, nil
}

type AlbumContent struct {
	ID string `json:"id,omitempty"`
	// OwnerID                    string    `json:"ownerId"`
	AlbumName   string   `json:"albumName"`
	Description string   `json:"description"`
	Shared      bool     `json:"shared"`
	Assets      []*Asset `json:"assets,omitempty"`
	AssetIDs    []string `json:"assetIds,omitempty"`
	// CreatedAt                  time.Time `json:"createdAt"`
	// UpdatedAt                  time.Time `json:"updatedAt"`
	// AlbumThumbnailAssetID      string    `json:"albumThumbnailAssetId"`
	// SharedUsers                []string  `json:"sharedUsers"`
	// Owner                      User      `json:"owner"`
	// AssetCount                 int       `json:"assetCount"`
	// LastModifiedAssetTimestamp time.Time `json:"lastModifiedAssetTimestamp"
}

// immich Asset simplified
type AssetSimplified struct {
	ID            string `json:"id"`
	DeviceAssetID string `json:"deviceAssetId"`
	// // OwnerID          string `json:"ownerId"`
	// DeviceID         string `json:"deviceId"`
	// Type             string `json:"type"`
	// OriginalPath     string `json:"originalPath"`
	// OriginalFileName string `json:"originalFileName"`
	// // Resized          bool      `json:"resized"`
	// // Thumbhash        string    `json:"thumbhash"`
	// FileCreatedAt time.Time `json:"fileCreatedAt"`
	// // FileModifiedAt time.Time `json:"fileModifiedAt"`
	// UpdatedAt time.Time `json:"updatedAt"`
	// // IsFavorite     bool      `json:"isFavorite"`
	// // IsArchived     bool      `json:"isArchived"`
	// // Duration       string    `json:"duration"`
	// // ExifInfo ExifInfo `json:"exifInfo"`
	// // LivePhotoVideoID any    `json:"livePhotoVideoId"`
	// // Tags             []any  `json:"tags"`
	// Checksum     string `json:"checksum"`
	// JustUploaded bool   `json:"-"`
}

func (ic *ImmichClient) GetAlbumInfo(ctx context.Context, id string, withoutAssets bool) (AlbumContent, error) {
	var album AlbumContent
	query := id
	if withoutAssets {
		query += "?withoutAssets=true"
	} else {
		query += "?withoutAssets=false"
	}
	err := ic.newServerCall(ctx, EndPointGetAlbumInfo).do(getRequest("/albums/"+query, setAcceptJSON()), responseJSON(&album))
	return album, err
}

func (ic *ImmichClient) GetAssetsAlbums(ctx context.Context, id string) ([]assets.Album, error) {
	var albums []AlbumSimplified
	err := ic.newServerCall(ctx, EndPointGetAlbumInfo).do(getRequest("/albums", setAcceptJSON()), responseJSON(&albums))
	if err != nil {
		return nil, err
	}
	return AlbumsFromAlbumSimplified(albums), nil
}

type UpdateAlbum struct {
	IDS []string `json:"ids"`
}

type UpdateAlbumResult struct {
	ID      string `json:"id"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func (ic *ImmichClient) AddAssetToAlbum(ctx context.Context, albumID string, assets []string) ([]UpdateAlbumResult, error) {
	if ic.dryRun {
		return []UpdateAlbumResult{}, nil
	}
	var r []UpdateAlbumResult
	body := UpdateAlbum{
		IDS: assets,
	}
	err := ic.newServerCall(ctx, EndPointAddAsstToAlbum).do(
		putRequest(fmt.Sprintf("/albums/%s/assets", albumID), setAcceptJSON(),
			setJSONBody(body)),
		responseJSON(&r))
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (ic *ImmichClient) CreateAlbum(ctx context.Context, name string, description string, assetsIDs []string) (assets.Album, error) {
	if ic.dryRun {
		return assets.Album{
			ID:    uuid.NewString(),
			Title: name,
		}, nil
	}
	body := AlbumContent{
		AlbumName:   name,
		Description: description,
		AssetIDs:    assetsIDs,
	}
	var r AlbumSimplified
	err := ic.newServerCall(ctx, EndPointCreateAlbum).do(
		postRequest("/albums", "application/json", setAcceptJSON(), setJSONBody(body)),
		responseJSON(&r))
	if err != nil {
		return assets.Album{}, err
	}
	return assets.Album{
		ID:          r.ID,
		Title:       r.AlbumName,
		Description: r.Description,
	}, nil
}

func (ic *ImmichClient) GetAssetAlbums(ctx context.Context, assetID string) ([]AlbumSimplified, error) {
	var r []AlbumSimplified
	err := ic.newServerCall(ctx, EndPointGetAssetAlbums).do(
		getRequest("/albums?assetId="+assetID, setAcceptJSON()),
		responseJSON(&r))
	return r, err
}

func (ic *ImmichClient) DeleteAlbum(ctx context.Context, id string) error {
	if ic.dryRun {
		return nil
	}
	return ic.newServerCall(ctx, EndPointDeleteAlbum).do(deleteRequest("/albums/" + id))
}
