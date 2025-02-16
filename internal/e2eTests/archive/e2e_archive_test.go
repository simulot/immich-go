//go:build e2e
// +build e2e

package archive

import (
	"context"
	"os"
	"testing"

	"github.com/simulot/immich-go/app/cmd"
	"github.com/simulot/immich-go/internal/e2eTests/e2e"
)

func TestArchiveFromGooglePhotos(t *testing.T) {
	e2e.InitMyEnv()
	e2e.ResetImmich(t)

	ctx := context.Background()

	tmpDir := os.TempDir()
	tmpDir, err := os.MkdirTemp(tmpDir, "archive_test_folder")
	if err != nil {
		t.Fatalf("os.MkdirTemp() error = %v", err)
		return
	}
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	c, a := cmd.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"archive", "from-google-photos",
		"--log-level=debug",
		"--write-to-folder=" + tmpDir,
		e2e.MyEnv("IMMICHGO_TESTFILES") + "/demo takeout/Takeout",
	})

	// let's start
	err = c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
		t.Error(err)
	}
}

func TestArchiveFromFolder(t *testing.T) {
	e2e.InitMyEnv()
	e2e.ResetImmich(t)

	ctx := context.Background()

	tmpDir := os.TempDir()
	tmpDir, err := os.MkdirTemp(tmpDir, "archive_test_folder")
	if err != nil {
		t.Fatalf("os.MkdirTemp() error = %v", err)
		return
	}
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	c, a := cmd.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"upload", "from-folder",
		"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
		"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
		"--log-level=debug",
		"--no-ui",
		"--into-album=ALBUM",
		"--manage-raw-jpeg=KeepRaw",
		"--manage-burst=stack",
		e2e.MyEnv("IMMICHGO_TESTFILES") + "/burst/storm",
	})

	// let's start
	err = c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
		t.Error(err)
		return
	}

	c, a = cmd.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"archive", "from-immich",
		"--from-server=" + e2e.MyEnv("IMMICHGO_SERVER"),
		"--from-api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
		"--write-to-folder=" + tmpDir,
		"--log-level=debug",
		e2e.MyEnv("IMMICHGO_TESTFILES") + "/burst/Reflex",
	})

	// let's start
	err = c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
		t.Error(err)
	}
}

func TestArchiveFromImmich(t *testing.T) {
	e2e.InitMyEnv()
	e2e.ResetImmich(t)

	ctx := context.Background()

	tmpDir := os.TempDir()
	tmpDir, err := os.MkdirTemp(tmpDir, "archive_test_folder")
	if err != nil {
		t.Fatalf("os.MkdirTemp() error = %v", err)
		return
	}
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	c, a := cmd.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"upload", "from-folder",
		"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
		"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
		"--log-level=debug",
		"--no-ui",
		"--into-album=ALBUM",
		"--manage-raw-jpeg=KeepRaw",
		"--manage-burst=stack",
		e2e.MyEnv("IMMICHGO_TESTFILES") + "/burst/storm",
	})

	// let's start
	err = c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
		t.Error(err)
	}
	c, a = cmd.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"archive", "from-immich",
		"--log-level=debug",
		"--from-server=" + e2e.MyEnv("IMMICHGO_SERVER"),
		"--from-api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
		"--write-to-folder=" + tmpDir,
	})

	// let's start
	err = c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
		t.Error(err)
	}
}
