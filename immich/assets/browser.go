package assets

import "context"

type Browser interface {
	Browse(cxt context.Context) chan *LocalAssetFile
}

type RealNamer interface {
	RealFileName(a *LocalAssetFile) (string, error)
}
