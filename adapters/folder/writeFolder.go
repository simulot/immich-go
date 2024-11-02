package folder

import "github.com/simulot/immich-go/internal/assets"

type LocalAssetWriter struct {
	WriteTo string
}

func NewLocalAssetWriter(writeTo string) *LocalAssetWriter {
	return &LocalAssetWriter{
		WriteTo: writeTo,
	}


func (w *LocalAssetWriter) Write(group *assets.Group) error {
	return nil
}