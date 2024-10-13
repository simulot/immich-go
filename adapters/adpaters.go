package adapters

import (
	"context"
)

// TODO: rename to Importer
type Adapter interface {
	Browse(cxt context.Context) (chan *AssetGroup, error)
}
