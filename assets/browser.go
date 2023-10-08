package assets

import "context"

type Browser interface {
	Browse(cxt context.Context) chan *LocalAssetFile
	// BrowseAlbums(ctx context.Context) error
}
