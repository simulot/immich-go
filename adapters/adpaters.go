package adapters

import (
	"context"

	"github.com/simulot/immich-go/internal/assets"
)

type Reader interface {
	Browse(cxt context.Context) chan *assets.Group
}
