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
	tmpDir, err := os.MkdirTemp(tmpDir, "upload_test_folder")
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
		"--write-to-folder=" + tmpDir,
		e2e.MyEnv("IMMICHGO_TESTFILES") + "/demo takeout/Takeout.zip",
	})

	// let's start
	err = c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
	}
}
