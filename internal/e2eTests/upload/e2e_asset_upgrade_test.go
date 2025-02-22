//go:build e2e
// +build e2e

package upload

import (
	"context"
	"os"
	"testing"

	"github.com/simulot/immich-go/app/cmd"
	"github.com/simulot/immich-go/internal/e2eTests/e2e"
	"github.com/simulot/immich-go/internal/fshelper/hash"
)

func TestUpgradePhotoFolder(t *testing.T) {
	e2e.InitMyEnv()
	e2e.ResetImmich(t)
	ctx := context.Background()

	client, err := e2e.GetImmichClient()
	if err != nil {
		t.Fatal(err)
		return
	}

	fsys := os.DirFS("../../../app/cmd/upload/TEST_DATA")
	lowSHA1, err := hash.Base64Encode(hash.FileSHA1Hash(fsys, "folder/low/PXL_20231006_063000139.jpg"))
	if err != nil {
		t.Fatal(err)
		return
	}
	highSHA1, err := hash.Base64Encode(hash.FileSHA1Hash(fsys, "folder/high/AlbumA/PXL_20231006_063000139.jpg"))
	if err != nil {
		t.Fatal(err)
		return
	}

	c, a := cmd.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"upload", "from-folder",
		"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
		"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
		"--folder-as-album=FOLDER",
		"--log-level=debug",
		"../../../app/cmd/upload/TEST_DATA/folder/low",
	})
	err = c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
		t.Error(err)
		return
	}

	e2e.WaitingForJobsEnding(ctx, client, t)

	assets, err := client.GetAssetsByImageName(ctx, "PXL_20231006_063000139.jpg")
	if err != nil {
		t.Fatal(err)
		return
	}
	if len(assets) != 1 {
		t.Fatal("Asset not found")
	}

	AssetID := assets[0].ID // Keep the ID for further checks

	c, a = cmd.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"upload", "from-folder",
		"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
		"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
		"--folder-as-album=FOLDER",
		"--log-level=debug",
		"--api-trace",
		"../../../app/cmd/upload/TEST_DATA/folder/high",
	})
	err = c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
		t.Error(err)
		return
	}

	e2e.WaitingForJobsEnding(ctx, client, t)

	asset, err := client.GetAssetInfo(ctx, AssetID)
	if err != nil {
		t.Fatal(err)
		return
	}

	if asset.Checksum == lowSHA1 {
		t.Errorf("Asset not upgraded")
	}
	if asset.Checksum != highSHA1 {
		t.Errorf("High quality version hash is unexpected")
	}
}

func TestUpgradeGooglePhotoFolder(t *testing.T) {
	e2e.InitMyEnv()
	e2e.ResetImmich(t)
	client, err := e2e.GetImmichClient()
	if err != nil {
		t.Fatal(err)
		return
	}

	ctx := context.Background()

	c, a := cmd.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"upload", "from-folder",
		"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
		"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
		"--folder-as-album=FOLDER",
		"--log-level=debug",

		"../../../app/cmd/upload/TEST_DATA/folder/low",
	})
	err = c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
		t.Fatal(err)
		return
	}

	e2e.WaitingForJobsEnding(ctx, client, t)

	assets, err := client.GetAssetsByImageName(ctx, "PXL_20231006_063000139.jpg")
	if err != nil {
		t.Fatal(err)
		return
	}
	if len(assets) != 1 {
		t.Fatal("Asset not found")
	}

	AssetID := assets[0].ID // Keep the ID for further checks

	c, a = cmd.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"upload", "from-google-photos",
		"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
		"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
		"--api-trace",
		"--log-level=debug",

		"../../../app/cmd/upload/TEST_DATA/Takeout1",
	})
	err = c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
		t.Fatal(err)
		return
	}

	e2e.WaitingForJobsEnding(ctx, client, t)

	al, err := client.GetAssetAlbums(ctx, AssetID)
	if err != nil {
		t.Fatal(err)
		return
	}

	if len(al) != 2 {
		t.Errorf("Expecting 3 albums, got %d", len(al))
	}
}

func TestUpgradeGooglePhotoFolder2(t *testing.T) {
	e2e.InitMyEnv()
	e2e.ResetImmich(t)
	client, err := e2e.GetImmichClient()
	if err != nil {
		t.Fatal(err)
		return
	}

	ctx := context.Background()

	c, a := cmd.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"upload", "from-google-photos",
		"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
		"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
		"--api-trace",
		"--log-level=debug",

		"../../../app/cmd/upload/TEST_DATA/Takeout1",
	})
	err = c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
		t.Fatal(err)
		return
	}

	e2e.WaitingForJobsEnding(ctx, client, t)

	c, a = cmd.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"upload", "from-folder",
		"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
		"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
		"--folder-as-album=FOLDER",
		"--log-level=debug",

		"../../../app/cmd/upload/TEST_DATA/folder/high",
	})
	err = c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
		t.Fatal(err)
		return
	}

	e2e.WaitingForJobsEnding(ctx, client, t)

	assets, err := client.GetAssetsByImageName(ctx, "PXL_20231006_063000139.jpg")
	if err != nil {
		t.Fatal(err)
		return
	}
	if len(assets) != 2 { // 1 visible, 1 trashed (low quality)
		t.Fatal("Asset not found")
	}

	AssetID := assets[0].ID // Keep the ID for further checks

	al, err := client.GetAssetAlbums(ctx, AssetID)
	if err != nil {
		t.Fatal(err)
		return
	}

	if len(al) != 2 {
		t.Errorf("Expecting asset in 3 albums, got %d", len(al))
	}
}
