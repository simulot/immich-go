package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/simulot/immich-go/immich"
)

func ImmichScan(t *testing.T, client *immich.ImmichClient) map[string]FileInfo {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	list, err := client.GetAllAssets(ctx)
	defer cancel()
	if err != nil {
		t.Fatal(err)
	}
	assets := map[string]FileInfo{}

	for _, asset := range list {
		if !asset.IsTrashed {
			assets[asset.OriginalFileName] = FileInfo{
				Size: int(asset.ExifInfo.FileSizeInByte),
				SHA1: asset.Checksum,
			}
		}
	}
	return assets
}
