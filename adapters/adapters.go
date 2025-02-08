package adapters

import (
	"context"

	"github.com/simulot/immich-go/internal/assets"
)

type Reader interface {
	Browse(cxt context.Context) chan *assets.Group
}

type AssetWriter interface {
	WriteAsset(context.Context, *assets.Asset) error
	// WriteGroup(ctx context.Context, group *assets.Group) error
}
