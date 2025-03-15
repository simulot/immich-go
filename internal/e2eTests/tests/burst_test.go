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

func TestBurstsFolders(t *testing.T) {
	t.Run("camera burst raw+jpg, Stack", func(t *testing.T) {
		const source = "DATA/bursts/jpg"
		e2e.InitMyEnv()
		e2e.ResetImmich(t)
		// client, err := e2e.GetImmichClient()
		// if err != nil {
		// 	t.Fatal(err)
		// 	return
		// }

		ctx := context.Background()
		c, a := cmd.RootImmichGoCommand(ctx)
		c.SetArgs([]string{
			"upload", "from-folder",
			"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
			"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
			"--no-ui",
			"--api-trace",
			"--log-level=debug",
			"--manage-burst=Stack",
			source,
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
			fileevent.Uploaded: 11,
			fileevent.Stacked:  7,
		}, false, a.Jnl())

		// sourceDir := e2e.ScanDirectory(t, source)
		// serverAssets := e2e.ImmichScan(t, client)
		// upFiles := e2e.ExtensionFilter(sourceDir, []string{".cr3"})
		// if !reflect.DeepEqual(serverAssets, upFiles) {
		// 	t.Error("Unexpected assets on server", serverAssets, "expected", upFiles)
		// }
	})
	t.Run("camera burst raw+jpg Stack", func(t *testing.T) {
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
			"--manage-burst=Stack",
			"DATA/bursts/raw-jpg",
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
			fileevent.Uploaded: 8, // TODO: fine tune burst detection
			fileevent.Stacked:  7,
		}, false, a.Jnl())
	})
	t.Run("camera burst raw+jpg, stack StackKeepRaw", func(t *testing.T) {
		const source = "DATA/bursts/raw-jpg"
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
			"--api-trace",
			"--log-level=debug",
			"--manage-burst=StackKeepRaw",
			source,
		})
		err = c.ExecuteContext(ctx)
		if err != nil && a.Log().GetSLog() != nil {
			a.Log().Error(err.Error())
		}

		if err != nil {
			t.Error("Unexpected error", err)
			return
		}

		e2e.CheckResults(t, map[fileevent.Code]int64{
			fileevent.Uploaded:            5,
			fileevent.Stacked:             4,
			fileevent.DiscoveredDiscarded: 3,
		}, false, a.Jnl())

		sourceDir := e2e.ScanDirectory(t, source)
		serverAssets := e2e.ImmichScan(t, client)
		upFiles := e2e.ExtensionFilter(sourceDir, []string{".cr3"})
		if !reflect.DeepEqual(serverAssets, upFiles) {
			t.Error("Unexpected assets on server", serverAssets, "expected", upFiles)
		}
	})

	t.Run("camera burst raw+jpg, stack StackKeepJPEG", func(t *testing.T) {
		const source = "DATA/bursts/raw-jpg"
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
			"--api-trace",
			"--log-level=debug",
			"--manage-burst=StackKeepJPEG",
			source,
		})
		err = c.ExecuteContext(ctx)
		if err != nil && a.Log().GetSLog() != nil {
			a.Log().Error(err.Error())
		}

		if err != nil {
			t.Error("Unexpected error", err)
			return
		}

		e2e.CheckResults(t, map[fileevent.Code]int64{
			fileevent.Uploaded:            4,
			fileevent.Stacked:             4,
			fileevent.DiscoveredDiscarded: 4,
		}, false, a.Jnl())

		sourceDir := e2e.ScanDirectory(t, source)
		serverAssets := e2e.ImmichScan(t, client)
		upFiles := e2e.ExtensionFilter(sourceDir, []string{".jpg"})
		if !reflect.DeepEqual(serverAssets, upFiles) {
			t.Error("Unexpected assets on server", serverAssets, "expected", upFiles)
		}
	})

	t.Run("phone burst", func(t *testing.T) {
		const source = "DATA/bursts/phone"
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
			"--api-trace",
			"--log-level=debug",
			"--manage-burst=Stack",
			source,
		})
		err = c.ExecuteContext(ctx)
		if err != nil && a.Log().GetSLog() != nil {
			a.Log().Error(err.Error())
		}

		if err != nil {
			t.Error("Unexpected error", err)
			return
		}

		e2e.CheckResults(t, map[fileevent.Code]int64{
			fileevent.Uploaded:            7,
			fileevent.Stacked:             7,
			fileevent.DiscoveredDiscarded: 0,
		}, false, a.Jnl())

		sourceDir := e2e.ScanDirectory(t, source)
		serverAssets := e2e.ImmichScan(t, client)
		upFiles := e2e.ExtensionFilter(sourceDir, []string{".jpg"})
		if !reflect.DeepEqual(serverAssets, upFiles) {
			t.Error("Unexpected assets on server", serverAssets, "expected", upFiles)
		}
	})

	t.Run("phone jpg burst but keepRaw ", func(t *testing.T) {
		const source = "DATA/bursts/phone"
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
			"--api-trace",
			"--log-level=debug",
			"--manage-burst=StackKeepRaw",
			source,
		})
		err = c.ExecuteContext(ctx)
		if err != nil && a.Log().GetSLog() != nil {
			a.Log().Error(err.Error())
		}

		if err != nil {
			t.Error("Unexpected error", err)
			return
		}

		e2e.CheckResults(t, map[fileevent.Code]int64{
			fileevent.Uploaded:            7,
			fileevent.Stacked:             7,
			fileevent.DiscoveredDiscarded: 0,
		}, false, a.Jnl())

		sourceDir := e2e.ScanDirectory(t, source)
		serverAssets := e2e.ImmichScan(t, client)
		upFiles := e2e.ExtensionFilter(sourceDir, []string{".jpg"})
		if !reflect.DeepEqual(serverAssets, upFiles) {
			t.Error("Unexpected assets on server", serverAssets, "expected", upFiles)
		}
	})
}
