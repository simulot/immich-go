package immich

import "fmt"

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

func (ic *ImmichClient) GetAllAlbums() ([]AlbumSimplified, error) {
	var albums []AlbumSimplified
	err := ic.newServerCall("GetAllAlbums").do(get("/album", setAcceptJSON()), responseJSON(&albums))
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

func (ic *ImmichClient) GetAlbumInfo(id string) (AlbumContent, error) {
	var album AlbumContent
	err := ic.newServerCall("GetAlbumInfo").do(get("/album/"+id, setAcceptJSON()), responseJSON(&album))
	return album, err
}

func (ic *ImmichClient) GetAssetsAlbums(id string) ([]AlbumSimplified, error) {
	var albums []AlbumSimplified
	err := ic.newServerCall("GetAllAlbums").do(get("/album", setAcceptJSON()), responseJSON(&albums))
	if err != nil {
		return nil, err
	}
	return albums, nil

}

type UpdateAlbumResponse struct {
	SuccessfullyAdded int `json:"successfullyAdded"`
}

type UpdateAlbum struct {
	AssetIds []string `json:"assetIds"`
}

func (ic *ImmichClient) UpdateAlbum(albumID string, assets []string) (UpdateAlbumResponse, error) {
	var r UpdateAlbumResponse
	body := UpdateAlbum{
		AssetIds: assets,
	}
	err := ic.newServerCall("UpdateAlbum").do(
		put(fmt.Sprintf("/album/%s/assets", albumID), setAcceptJSON(),
			setJSONBody(body)),
		responseJSON(&r))
	if err != nil {
		return UpdateAlbumResponse{}, err
	}
	return r, nil
}

func (ic *ImmichClient) CreateAlbum(name string, assets []string) (AlbumSimplified, error) {
	body := AlbumSimplified{
		AlbumName: name,
		AssetIds:  assets,
	}
	var r AlbumSimplified
	err := ic.newServerCall("CreateAlbum").do(
		post("/album", "application/json", setAcceptJSON(), setJSONBody(body)),
		responseJSON(&r))
	if err != nil {
		return AlbumSimplified{}, err
	}
	return r, nil
}
