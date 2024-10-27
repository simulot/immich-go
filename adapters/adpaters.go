package adapters

import (
	"context"

	"github.com/simulot/immich-go/internal/groups"
)

// TODO: rename to Importer
type Adapter interface {
	Browse(cxt context.Context) chan *groups.AssetGroup
}
