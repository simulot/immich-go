package immich

import (
	"context"
	"image"
	"io"
)

func (ic *ImmichClient) GetAssetThumbnail(ctx context.Context, id string) (image.Image, error) {
	var img image.Image
	err := ic.newServerCall(ctx, "getAssetThumbnail").do(get("/asset/thumbnail/"+id, setAcceptType("application/octet-stream")),
		responseBodyHandler(func(r io.Reader) error {
			var err error
			img, _, err = image.Decode(r)
			return err
		}))
	return img, err
}
