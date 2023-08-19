package immich

import (
	"context"
	"fmt"
)

type AlbumSimplified struct {
	ID string `json:"id,omitempty"`
	// OwnerID                    string    `json:"ownerId"`
	AlbumName string `json:"albumName"`
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

func (ic *ImmichClient) GetAllAlbums(ctx context.Context) ([]AlbumSimplified, error) {
	var albums []AlbumSimplified
	err := ic.newServerCall(ctx, "GetAllAlbums").do(get("/album", setAcceptJSON()), responseJSON(&albums))
	if err != nil {
		return nil, err
	}
	return albums, nil

}

type AlbumContent struct {
	ID string `json:"id,omitempty"`
	// OwnerID                    string    `json:"ownerId"`
	AlbumName string `json:"albumName"`
	// CreatedAt                  time.Time `json:"createdAt"`
	// UpdatedAt                  time.Time `json:"updatedAt"`
	// AlbumThumbnailAssetID      string    `json:"albumThumbnailAssetId"`
	// SharedUsers                []string  `json:"sharedUsers"`
	// Owner                      User      `json:"owner"`
	// Shared                     bool      `json:"shared"`
	// AssetCount                 int       `json:"assetCount"`
	// LastModifiedAssetTimestamp time.Time `json:"lastModifiedAssetTimestamp"
	Assets []AssetSimplified `json:"assets,omitempty"`
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

func (ic *ImmichClient) GetAlbumInfo(ctx context.Context, id string) (AlbumContent, error) {
	var album AlbumContent
	err := ic.newServerCall(ctx, "GetAlbumInfo").do(get("/album/"+id, setAcceptJSON()), responseJSON(&album))
	return album, err
}

func (ic *ImmichClient) GetAssetsAlbums(ctx context.Context, id string) ([]AlbumSimplified, error) {
	var albums []AlbumSimplified
	err := ic.newServerCall(ctx, "GetAllAlbums").do(get("/album", setAcceptJSON()), responseJSON(&albums))
	if err != nil {
		return nil, err
	}
	return albums, nil

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

	var r []UpdateAlbumResult
	body := UpdateAlbum{
		IDS: assets,
	}
	err := ic.newServerCall(ctx, "AddAssetToAlbum").do(
		put(fmt.Sprintf("/album/%s/assets", albumID), setAcceptJSON(),
			setJSONBody(body)),
		responseJSON(&r))
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (ic *ImmichClient) CreateAlbum(ctx context.Context, name string, assets []string) (AlbumSimplified, error) {
	body := AlbumSimplified{
		AlbumName: name,
		AssetIds:  assets,
	}
	var r AlbumSimplified
	err := ic.newServerCall(ctx, "CreateAlbum").do(
		post("/album", "application/json", setAcceptJSON(), setJSONBody(body)),
		responseJSON(&r))
	if err != nil {
		return AlbumSimplified{}, err
	}
	return r, nil
}
