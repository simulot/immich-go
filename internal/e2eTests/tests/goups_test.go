//go:build e2e

package tests

import (
	"context"
	"reflect"
	"testing"

	"github.com/simulot/immich-go/app/cmd"
	"github.com/simulot/immich-go/internal/e2eTests/e2e"
	"github.com/simulot/immich-go/internal/fileevent"
)

func TestGroupsFolders(t *testing.T) {
	t.Run("simple upload", func(t *testing.T) {
		e2e.InitMyEnv()
		e2e.ResetImmich(t)

		ctx := context.Background()
		c, a := cmd.RootImmichGoCommand(ctx)
		c.SetArgs([]string{
			"upload", "from-folder",
			"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
			"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
			"--no-ui",
			"--api-trace",
			"--log-level=debug",
			"DATA/groups",
		})
		err := c.ExecuteContext(ctx)
		if err != nil && a.Log().GetSLog() != nil {
			a.Log().Error(err.Error())
		}

		if err != nil {
			t.Error("Unexpected error", err)
			return
		}

		e2e.CheckResults(t, map[fileevent.Code]int64{
			fileevent.Uploaded: 4,
		}, false, a.Jnl())
	})

	t.Run("KeepRaw,KeepHeic", func(t *testing.T) {
		e2e.InitMyEnv()
		e2e.ResetImmich(t)

		client, err := e2e.GetImmichClient()
		if err != nil {
			t.Fatal(err)
			return
		}

		sourceDir := e2e.ScanDirectory(t, "DATA/groups")
		ctx := context.Background()
		c, a := cmd.RootImmichGoCommand(ctx)
		c.SetArgs([]string{
			"upload", "from-folder",
			"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
			"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
			"--no-ui",
			"--manage-raw-jpeg=KeepRaw",
			"--manage-heic-jpeg=KeepHeic",
			"--api-trace",
			"--log-level=debug",
			"DATA/groups",
		})
		err = c.ExecuteContext(ctx)
		if err != nil && a.Log().GetSLog() != nil {
			a.Log().Error(err.Error())
		}

		if err != nil {
			t.Error("Unexpected error", err)
			return
		}

		e2e.WaitingForJobsEnding(ctx, client, t)

		serverAssets := e2e.ImmichScan(t, client)
		upFiles := e2e.ExtensionFilter(sourceDir, []string{".dng", ".heic"})

		e2e.CheckResults(t, map[fileevent.Code]int64{
			fileevent.Uploaded:            int64(len(upFiles)),
			fileevent.DiscoveredDiscarded: 2,
		}, false, a.Jnl())

		if !reflect.DeepEqual(serverAssets, upFiles) {
			t.Error("Unexpected assets on server", serverAssets, "expected", upFiles)
		}
	})
	t.Run("KeepJPG", func(t *testing.T) {
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
			"--no-ui",
			"--manage-raw-jpeg=KeepJPG",
			"--manage-heic-jpeg=KeepJPG",
			"--api-trace",
			"--log-level=debug",
			"DATA/groups",
		})
		err = c.ExecuteContext(ctx)
		if err != nil && a.Log().GetSLog() != nil {
			a.Log().Error(err.Error())
		}

		if err != nil {
			t.Error("Unexpected error", err)
			return
		}

		e2e.WaitingForJobsEnding(ctx, client, t)

		sourceDir := e2e.ScanDirectory(t, "DATA/groups")
		serverAssets := e2e.ImmichScan(t, client)
		upFiles := e2e.ExtensionFilter(sourceDir, []string{".jpg"})

		e2e.CheckResults(t, map[fileevent.Code]int64{
			fileevent.Uploaded:            int64(len(upFiles)),
			fileevent.DiscoveredDiscarded: 2,
		}, false, a.Jnl())

		if !reflect.DeepEqual(serverAssets, upFiles) {
			t.Error("Unexpected assets on server", serverAssets, "expected", upFiles)
		}
	})
}
