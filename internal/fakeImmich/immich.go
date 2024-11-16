package fakeimmich

import (
	"context"
	"io"

	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/filetypes"
)

type MockedCLient struct{}

func (c *MockedCLient) GetAllAssetsWithFilter(context.Context, func(*immich.Asset) error) error {
	return nil
}

func (c *MockedCLient) AssetUpload(context.Context, *assets.Asset) (immich.AssetResponse, error) {
	return immich.AssetResponse{}, nil
}

func (c *MockedCLient) DeleteAssets(context.Context, []string, bool) error {
	return nil
}

func (c *MockedCLient) GetAllAlbums(context.Context) ([]immich.AlbumSimplified, error) {
	return nil, nil
}

func (c *MockedCLient) AddAssetToAlbum(context.Context, string, []string) ([]immich.UpdateAlbumResult, error) {
	return nil, nil
}

func (c *MockedCLient) CreateAlbum(context.Context, string, string, []string) (immich.AlbumSimplified, error) {
	return immich.AlbumSimplified{}, nil
}

func (c *MockedCLient) UpdateAssets(ctx context.Context, ids []string, isArchived bool, isFavorite bool, latitude float64, longitude float64, removeParent bool, stackParentID string) error {
	return nil
}

func (c *MockedCLient) StackAssets(ctx context.Context, cover string, ids []string) error {
	return nil
}

func (c *MockedCLient) UpdateAsset(ctx context.Context, id string, a *assets.Asset) (*immich.Asset, error) {
	return nil, nil
}

func (c *MockedCLient) EnableAppTrace(w io.Writer) {}

func (c *MockedCLient) GetServerStatistics(ctx context.Context) (immich.ServerStatistics, error) {
	return immich.ServerStatistics{}, nil
}

func (c *MockedCLient) PingServer(ctx context.Context) error {
	return nil
}

func (c *MockedCLient) SetDeviceUUID(string) {}

func (c *MockedCLient) SetEndPoint(string) {}

func (c *MockedCLient) ValidateConnection(ctx context.Context) (immich.User, error) {
	return immich.User{}, nil
}

func (c *MockedCLient) GetAssetAlbums(ctx context.Context, id string) ([]immich.AlbumSimplified, error) {
	return nil, nil
}

func (c *MockedCLient) GetAllAssets(ctx context.Context) ([]*immich.Asset, error) {
	return nil, nil
}

func (c *MockedCLient) DeleteAlbum(ctx context.Context, id string) error {
	return nil
}

func (c *MockedCLient) SupportedMedia() filetypes.SupportedMedia {
	return filetypes.DefaultSupportedMedia
}

func (c *MockedCLient) GetAssetStatistics(ctx context.Context) (immich.UserStatistics, error) {
	return immich.UserStatistics{
		Images: 1,
		Videos: 1,
		Total:  1,
	}, nil
}

func (c *MockedCLient) GetJobs(ctx context.Context) (map[string]immich.Job, error) {
	return nil, nil
}

func (c *MockedCLient) GetAlbumInfo(context.Context, string, bool) (immich.AlbumContent, error) {
	return immich.AlbumContent{}, nil
}
