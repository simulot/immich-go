package immich

import "fmt"

type AlbumSimplified struct {
	ID string `json:"id"`
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
	AssetIds []string `json:"assetIds"`
}

func (ic *ImmichClient) GetAllAlbums() ([]AlbumSimplified, error) {
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
		post("/album", "application/json", setTraceJSONRequest(), setAcceptJSON(), setJSONBody(body)),
		setTraceJSONResponse(), responseJSON(&r))
	if err != nil {
		return AlbumSimplified{}, err
	}
	return r, nil
}
