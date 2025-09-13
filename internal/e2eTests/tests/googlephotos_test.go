//go:build e2e

package tests

import (
	"context"
	"testing"

	"github.com/simulot/immich-go/app/cmd"
	"github.com/simulot/immich-go/internal/e2eTests/e2e"
	"github.com/simulot/immich-go/internal/fileevent"
)

func TestGPFolders(t *testing.T) {
	t.Run("duplicates upload from folder", func(t *testing.T) {
		e2e.InitMyEnv()
		e2e.ResetImmich(t)
		ctx := context.Background()
		c, a := cmd.RootImmichGoCommand(ctx)
		c.SetArgs([]string{
			"upload", "from-google-photos",
			"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
			"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
			"--api-trace",
			"--log-level=debug",
			"DATA/taketouts/low_quality/folder",
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
			fileevent.Uploaded:                   5,
			fileevent.UploadAddToAlbum:           5,
			fileevent.Tagged:                     5,
			fileevent.AnalysisLocalDuplicate:     5,
			fileevent.AnalysisAssociatedMetadata: 10,
		}, false, a.Jnl())
	})

	t.Run("duplicates upload from zip", func(t *testing.T) {
		e2e.InitMyEnv()
		e2e.ResetImmich(t)
		ctx := context.Background()
		c, a := cmd.RootImmichGoCommand(ctx)
		c.SetArgs([]string{
			"upload", "from-google-photos",
			"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
			"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
			"--api-trace",
			"--log-level=debug",
			"DATA/taketouts/low_quality/takeout.zip",
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
			fileevent.Uploaded:                   5,
			fileevent.UploadAddToAlbum:           5,
			fileevent.Tagged:                     5,
			fileevent.AnalysisLocalDuplicate:     5,
			fileevent.AnalysisAssociatedMetadata: 10,
		}, false, a.Jnl())
	})

	t.Run("duplicates in same input", func(t *testing.T) {
		e2e.InitMyEnv()
		e2e.ResetImmich(t)
		ctx := context.Background()
		c, a := cmd.RootImmichGoCommand(ctx)
		c.SetArgs([]string{
			"upload", "from-google-photos",
			"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
			"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
			"--api-trace",
			"--log-level=debug",
			"DATA/taketouts/duplicates",
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
			fileevent.Uploaded:                   2,
			fileevent.UploadAddToAlbum:           0, // TODO add originals into the album, result depend on the order of the upload
			fileevent.Tagged:                     2,
			fileevent.AnalysisLocalDuplicate:     2,
			fileevent.AnalysisAssociatedMetadata: 4,
		}, false, a.Jnl())
	})

	t.Run("duplicate names only in same input", func(t *testing.T) {
		e2e.InitMyEnv()
		e2e.ResetImmich(t)
		ctx := context.Background()
		c, a := cmd.RootImmichGoCommand(ctx)
		c.SetArgs([]string{
			"upload", "from-google-photos",
			"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
			"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
			"--api-trace",
			"--log-level=debug",
			"DATA/taketouts/duplicate in name only",
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
			fileevent.Uploaded:                   5,
			fileevent.UploadAddToAlbum:           0, // TODO add originals into the album, result depend on the order of the upload
			fileevent.Tagged:                     5,
			fileevent.AnalysisLocalDuplicate:     0,
			fileevent.AnalysisAssociatedMetadata: 5,
		}, false, a.Jnl())
	})

	t.Run("duplicates with existing assets", func(t *testing.T) {
		e2e.InitMyEnv()
		e2e.ResetImmich(t)
		ctx := context.Background()
		c, a := cmd.RootImmichGoCommand(ctx)

		// Preloads images with different names
		c.SetArgs([]string{
			"upload", "from-folder",
			"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
			"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
			"--api-trace",
			"--log-level=debug",
			"DATA/taketouts/duplicate with existing assets/fixtures",
		})
		err := c.ExecuteContext(ctx)
		if err != nil && a.Log().GetSLog() != nil {
			a.Log().Error(err.Error())
		}
		if err != nil {
			t.Error("Unexpected error", err)
			return
		}

		// Load the album when local duplicates and existing assets pre-existing on the server with a different name
		c, a = cmd.RootImmichGoCommand(ctx)
		c.SetArgs([]string{
			"upload", "from-google-photos",
			"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
			"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
			"--api-trace",
			"--log-level=debug",
			"DATA/taketouts/duplicate with existing assets/Google Photos",
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
			fileevent.Uploaded:                   3,
			fileevent.UploadAddToAlbum:           0,
			fileevent.Tagged:                     3,
			fileevent.AnalysisLocalDuplicate:     0,
			fileevent.AnalysisAssociatedMetadata: 5,
			fileevent.UploadServerDuplicate:      2,
		}, false, a.Jnl())
	})

	t.Run("upgrade and duplicate with existing assets", func(t *testing.T) {
		e2e.InitMyEnv()
		e2e.ResetImmich(t)
		ctx := context.Background()
		c, a := cmd.RootImmichGoCommand(ctx)

		// Preloads images with different names
		c.SetArgs([]string{
			"upload", "from-folder",
			"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
			"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
			"--api-trace",
			"--log-level=debug",
			"DATA/taketouts/upgrade and duplicate/fixtures",
		})
		err := c.ExecuteContext(ctx)
		if err != nil && a.Log().GetSLog() != nil {
			a.Log().Error(err.Error())
		}
		if err != nil {
			t.Error("Unexpected error", err)
			return
		}

		// Load the album when local duplicates and existing assets pre-existing on the server with a different name
		c, a = cmd.RootImmichGoCommand(ctx)
		c.SetArgs([]string{
			"upload", "from-google-photos",
			"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
			"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
			"--api-trace",
			"--log-level=debug",
			"DATA/taketouts/upgrade and duplicate/Google Photos",
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
			fileevent.Uploaded:                   0,
			fileevent.UploadUpgraded:             2,
			fileevent.UploadAddToAlbum:           0,
			fileevent.Tagged:                     2,
			fileevent.AnalysisLocalDuplicate:     0,
			fileevent.AnalysisAssociatedMetadata: 2,
			fileevent.UploadServerDuplicate:      0,
		}, false, a.Jnl())
	})
}
