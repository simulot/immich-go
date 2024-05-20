package browser

import (
	"context"
)

type Browser interface {
	Prepare(cxt context.Context) error
	Browse(cxt context.Context) chan *LocalAssetFile
}
