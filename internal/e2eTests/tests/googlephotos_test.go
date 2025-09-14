//go:build e2e

package tests

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/simulot/immich-go/app/cmd"
	"github.com/simulot/immich-go/internal/e2eTests/e2e"
	"github.com/simulot/immich-go/internal/fileevent"
)

func TestDuplicateFromFolder(t *testing.T) {
	for concurrent := range 10 {
		t.Run("TestDuplicateFromFolder with "+strconv.Itoa(concurrent), func(t *testing.T) {
			e2e.InitMyEnv()
			e2e.ResetImmich(t)
			start := time.Now()
			ctx := context.Background()
			c, a := cmd.RootImmichGoCommand(ctx)
			c.SetArgs([]string{
				"upload", "from-google-photos",
				"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
				"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
				"--api-trace",
				"--log-level=debug",
				"--concurrent-uploads=" + strconv.Itoa(concurrent),
				"DATA/taketouts/low_quality/folder",
			})
			c.SetOut(t.Output())
			err := c.ExecuteContext(ctx)
			if err != nil && a.Log().GetSLog() != nil {
				a.Log().Error(err.Error())
			}
			fmt.Fprintln(t.Output(), "Execution time: ", time.Since(start))

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
			time.Sleep(500 * time.Millisecond)
		})
	}
}

func TestDuplicateFromZippedFolder(t *testing.T) {
	for concurrent := range 4 {
		t.Run("TestDuplicateFromZippedFolder with "+strconv.Itoa(concurrent), func(t *testing.T) {
			e2e.InitMyEnv()
			e2e.ResetImmich(t)
			ctx := context.Background()
			c, a := cmd.RootImmichGoCommand(ctx)
			c.SetArgs([]string{
				"upload", "from-google-photos",
				"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
				"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
				"--api-trace",
				"--concurrent-uploads=" + strconv.Itoa(concurrent),
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
			time.Sleep(500 * time.Millisecond)
		})
	}
}

func TestImportFromGooglePhotos(t *testing.T) {
	for concurrent := range 4 {
		t.Run("TestImportFromGooglePhotos with "+strconv.Itoa(concurrent), func(t *testing.T) {
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
				"--concurrent-uploads=" + strconv.Itoa(concurrent),
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
			time.Sleep(500 * time.Millisecond)
		})
	}
}

func TestImportFromGooglePhotosNameDUplicates(t *testing.T) {
	for concurrent := range 4 {
		t.Run("TestImportFromGooglePhotosNameDUplicates with "+strconv.Itoa(concurrent), func(t *testing.T) {
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
				"--concurrent-uploads=" + strconv.Itoa(concurrent),
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
			time.Sleep(500 * time.Millisecond)
		})
	}
}

func TestImportFromGooglePhotosImmichDUplicates(t *testing.T) {
	for concurrent := range 4 {
		t.Run("TestImportFromGooglePhotosImmichDUplicates with "+strconv.Itoa(concurrent), func(t *testing.T) {
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
				"--concurrent-uploads=" + strconv.Itoa(concurrent),
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

			time.Sleep(500 * time.Millisecond)
		})
	}
}

func TestImportFromGooglePhotosUpgrade(t *testing.T) {
	for concurrent := range 4 {
		t.Run("TestImportFromGooglePhotosUpgrade with "+strconv.Itoa(concurrent), func(t *testing.T) {
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
				"--concurrent-uploads=" + strconv.Itoa(concurrent),
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
				"--concurrent-uploads=" + strconv.Itoa(concurrent),
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

			time.Sleep(500 * time.Millisecond)
		})
	}
}
