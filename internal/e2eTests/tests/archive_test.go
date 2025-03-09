//go:build e2e

package tests

import (
	"context"
	"os"
	"reflect"
	"testing"

	"github.com/simulot/immich-go/app/cmd"
	"github.com/simulot/immich-go/internal/e2eTests/e2e"
	"github.com/simulot/immich-go/internal/fileevent"
)

func TestArchiveFromImmich(t *testing.T) {
	e2e.InitMyEnv()
	e2e.ResetImmich(t)

	const fileSource = "DATA/low_jpg"

	fileList := e2e.ScanDirectory(t, fileSource)
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
		fileSource,
	})
	err := c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
	}

	if err != nil {
		t.Error("Unexpected error", err)
		return
	}

	tmpFolder := os.TempDir() + "/immich-go-test"
	os.RemoveAll(tmpFolder)

	defer os.RemoveAll(tmpFolder)

	c.SetArgs([]string{
		"archive",
		"from-immich",
		"--from-server=" + e2e.MyEnv("IMMICHGO_SERVER"),
		"--from-api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
		// "--no-ui",
		"--from-api-trace",
		"--log-level=debug",
		"--write-to-folder", tmpFolder,
	})

	err = c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
	}

	if err != nil {
		t.Error("Unexpected error", err)
		return
	}

	archivedFile := e2e.ScanDirectory(t, tmpFolder)
	archiveImages := e2e.BaseNameFilter(e2e.ExtensionFilter(archivedFile, []string{".jpg"}))

	if !reflect.DeepEqual(fileList, archiveImages) {
		t.Error("Unexpected file list", archivedFile)
	}

	// reload the server with the archive
	e2e.ResetImmich(t)
	c, a = cmd.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"upload", "from-folder",
		"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
		"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
		"--no-ui",
		"--api-trace",
		"--log-level=debug",
		tmpFolder,
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
		fileevent.Uploaded:         int64(len(fileList)),
		fileevent.UploadAddToAlbum: int64(len(fileList)),
		// fileevent.Tagged:           int64(len(fileList)), // see https://github.com/immich-app/immich/issues/16747
	}, false, a.Jnl())
}
