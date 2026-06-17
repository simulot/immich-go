package adapters

import (
	"context"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/spf13/cobra"
)

type AlbumUser struct {
	UserID string
	Role   string
}

type Reader interface {
	Browse(cxt context.Context) chan *assets.Group
}

type AlbumUserProvider interface {
	DesiredAlbumUsers(ctx context.Context, album assets.Album) ([]AlbumUser, error)
}

type AssetWriter interface {
	WriteAsset(context.Context, *assets.Asset) error
	// WriteGroup(ctx context.Context, group *assets.Group) error
}

type Runner interface {
	Run(cmd *cobra.Command, adapter Reader) error
}
