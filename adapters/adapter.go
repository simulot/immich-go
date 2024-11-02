package adapters

import (
	"context"

	"github.com/simulot/immich-go/internal/assets"
)

// TODO: rename to Importer
type Adapter interface {
	Browse(cxt context.Context) chan *assets.Group
}
