package adapters

import (
	"context"
)

// TODO: rename to Importer
type Adapter interface {
	Prepare(cxt context.Context) error
	Browse(cxt context.Context) chan *LocalAssetFile
}
