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

func TestFromFolders(t *testing.T) {
	highJpgs := e2e.ScanDirectory(t, "DATA/high_jpg")
	// lowJpgs := e2e.ScanDirectory(t, "DATA/low_jpg")

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
			"--tag=tag/subtag",
			"--into-album=album",
			"DATA/high_jpg",
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
			fileevent.Uploaded:         int64(len(highJpgs)),
			fileevent.UploadAddToAlbum: int64(len(highJpgs)),
			fileevent.Tagged:           int64(len(highJpgs)),
		}, false, a.Jnl())
	})

	t.Run("upload duplicates with same names discarded", func(t *testing.T) {
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
			"--tag=tag/subtag",
			"--into-album=album",
			"DATA/low_jpg", "DATA/low_jpg",
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
			fileevent.Uploaded:               5,
			fileevent.AnalysisLocalDuplicate: 5,
			fileevent.UploadAddToAlbum:       5,
			fileevent.Tagged:                 5,
		}, false, a.Jnl())
	})

	t.Run("force upload duplicates", func(t *testing.T) {
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
			"--tag=tag/subtag",
			"--into-album=album",
			"DATA/low_jpg",
		})
		err := c.ExecuteContext(ctx)
		if err != nil && a.Log().GetSLog() != nil {
			a.Log().Error(err.Error())
		}

		if err != nil {
			t.Error("Unexpected error", err)
			return
		}

		c, a = cmd.RootImmichGoCommand(ctx)
		c.SetArgs([]string{
			"upload", "from-folder",
			"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
			"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
			"--no-ui",
			"--api-trace",
			"--log-level=debug",
			"--overwrite",
			"DATA/low_jpg_altered",
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
			fileevent.Uploaded:               0,
			fileevent.AnalysisLocalDuplicate: 0,
			fileevent.UploadAddToAlbum:       0,
			fileevent.UploadUpgraded:         1,
			fileevent.UploadServerDuplicate:  4,
			fileevent.Tagged:                 0,
		}, false, a.Jnl())
	})

	t.Run("low quality same names are discarded", func(t *testing.T) {
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
			"--tag=tag/subtag",
			"--into-album=album",
			"DATA/high_jpg",
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

		c, a = cmd.RootImmichGoCommand(ctx)
		c.SetArgs([]string{
			"upload", "from-folder",
			"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
			"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
			"--no-ui",
			"--api-trace",
			"--log-level=debug",
			"--tag=tag/subtag2",
			"--into-album=album",
			"DATA/low_jpg",
		})
		err = c.ExecuteContext(ctx)
		if err != nil && a.Log().GetSLog() != nil {
			a.Log().Error(err.Error())
		}

		e2e.CheckResults(t, map[fileevent.Code]int64{
			fileevent.Uploaded:           0,
			fileevent.UploadServerBetter: int64(len(highJpgs)),
			fileevent.UploadAddToAlbum:   0,
			fileevent.Tagged:             0,
		}, false, a.Jnl())
	})

	t.Run("different names same upload are discarded", func(t *testing.T) {
		e2e.InitMyEnv()
		e2e.ResetImmich(t)
		ctx := context.Background()
		c, a := cmd.RootImmichGoCommand(ctx)
		c.SetArgs([]string{
			"upload", "from-folder",
			"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
			"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
			"--no-ui",
			"--tag=tag/subtag",
			"--into-album=album",
			"--api-trace",
			"--log-level=debug",
			"DATA/low_duplicates",
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
			fileevent.Uploaded:               2,
			fileevent.AnalysisLocalDuplicate: 1,
			fileevent.UploadServerBetter:     0,
			fileevent.UploadAddToAlbum:       2,
			fileevent.Tagged:                 2,
		}, false, a.Jnl())
	})
	t.Run("different names, separate uploads, duplicates are discarded", func(t *testing.T) {
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
			"--tag=tag/subtag1",
			"--into-album=album1",
			"--api-trace",
			"--log-level=debug",
			"DATA/low_jpg",
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

		c, a = cmd.RootImmichGoCommand(ctx)
		c.SetArgs([]string{
			"upload", "from-folder",
			"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
			"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
			"--no-ui",
			"--tag=tag/subtag2",
			"--into-album=album2",
			"--api-trace",
			"--log-level=debug",
			"DATA/low_other_names",
		})
		err = c.ExecuteContext(ctx)
		if err != nil && a.Log().GetSLog() != nil {
			a.Log().Error(err.Error())
		}

		e2e.CheckResults(t, map[fileevent.Code]int64{
			fileevent.Uploaded:               1,
			fileevent.AnalysisLocalDuplicate: 0,
			fileevent.UploadServerBetter:     0,
			fileevent.UploadServerDuplicate:  2,
			fileevent.UploadAddToAlbum:       3,
			fileevent.Tagged:                 3,
		}, false, a.Jnl())
	})

	t.Run("low quality upgraded by high quality", func(t *testing.T) {
		e2e.InitMyEnv()
		e2e.ResetImmich(t)
		ctx := context.Background()

		client, err := e2e.GetImmichClient()
		if err != nil {
			t.Fatal(err)
			return
		}

		c, a := cmd.RootImmichGoCommand(ctx)
		c.SetArgs([]string{
			"upload", "from-folder",
			"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
			"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
			"--tag=tag/subtag1",
			"--into-album=album1",
			"--no-ui",
			"DATA/low_jpg",
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

		c, a = cmd.RootImmichGoCommand(ctx)
		c.SetArgs([]string{
			"upload", "from-folder",
			"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
			"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
			"--no-ui",
			"--tag=tag/subtag1",
			"--into-album=album1",
			"--api-trace",
			"--log-level=debug",
			"DATA/high_jpg",
		})
		err = c.ExecuteContext(ctx)
		if err != nil && a.Log().GetSLog() != nil {
			a.Log().Error(err.Error())
		}

		serverAssets := e2e.ImmichScan(t, client)
		if !reflect.DeepEqual(serverAssets, highJpgs) {
			t.Error("Unexpected assets on server", serverAssets, "expected", highJpgs)
		}

		e2e.CheckResults(t, map[fileevent.Code]int64{
			fileevent.Uploaded:               0,
			fileevent.AnalysisLocalDuplicate: 0,
			fileevent.UploadServerBetter:     0,
			fileevent.UploadServerDuplicate:  0,
			fileevent.UploadUpgraded:         5,
			fileevent.UploadAddToAlbum:       0,
			fileevent.Tagged:                 5,
		}, false, a.Jnl())
	})

	t.Run("date-from-name", func(t *testing.T) {
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
			"--tag=tag/subtag",
			"--into-album=album",
			"--date-from-name",
			"DATA/dates",
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
			fileevent.Uploaded:         1,
			fileevent.UploadAddToAlbum: 1,
			fileevent.Tagged:           1,
		}, false, a.Jnl())
	})

	t.Run("#843 no longitude", func(t *testing.T) {
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
			"--into-album=album",
			"--date-from-name",
			"DATA/#843 longitude",
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
			fileevent.Uploaded:         2,
			fileevent.UploadAddToAlbum: 2,
		}, false, a.Jnl())
	})
}
