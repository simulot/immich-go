//go:build e2e
// +build e2e

package upload_test

import (
	"context"
	"os"
	"testing"

	"github.com/simulot/immich-go/app/cmd"
	"github.com/simulot/immich-go/internal/e2eTests/e2e"
)

func TestResetImmich(t *testing.T) {
	e2e.InitMyEnv()
	e2e.ResetImmich(t)
}

func TestUploadFromGooglePhotos(t *testing.T) {
	e2e.InitMyEnv()
	e2e.ResetImmich(t)

	ctx := context.Background()

	c, a := cmd.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"upload", "from-google-photos",
		"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
		"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
		"--log-level=DEBUG",
		// "--no-ui",
		e2e.MyEnv("IMMICHGO_TESTFILES") + "/demo takeout/Takeout",
	})

	// let's start
	err := c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
	}
}

func TestUploadFromGooglePhotosZipped(t *testing.T) {
	e2e.InitMyEnv()
	e2e.ResetImmich(t)

	ctx := context.Background()

	c, a := cmd.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"upload", "from-google-photos",
		"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
		"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
		"--log-level=DEBUG",
		"--manage-burst=Stack",
		"--manage-raw-jpeg=StackCoverJPG",
		// "--no-ui",
		// e2e.MyEnv("IMMICHGO_TESTFILES") + "/demo takeout/zip/takeout-*.zip",
		e2e.MyEnv("IMMICHGO_TESTFILES") + "/#380 duplicates in GP/Takeout*.zip",
	})

	// let's start
	err := c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
	}
}

func TestUploadFromGooglePhotosZippedIssue608(t *testing.T) {
	e2e.InitMyEnv()
	e2e.ResetImmich(t)

	ctx := context.Background()

	c, a := cmd.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"upload", "from-google-photos",
		"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
		"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
		"--log-level=DEBUG",
		"--manage-burst=Stack",
		"--manage-raw-jpeg=StackCoverJPG",
		"--include-unmatched",
		// "--no-ui",
		e2e.MyEnv("IMMICHGO_TESTFILES") + "/burst/takeout-reflex.zip",
	})

	// let's start
	err := c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
	}
}

func TestUploadFromGPInCurrent(t *testing.T) {
	curDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
		return
	}

	t.Cleanup(func() {
		_ = os.Chdir(curDir)
	})

	e2e.InitMyEnv()
	e2e.ResetImmich(t)

	ctx := context.Background()
	d := e2e.MyEnv("IMMICHGO_TESTFILES") + "/demo takeout/Takeout"
	err = os.Chdir(d)
	if err != nil {
		t.Fatal(err)
		return
	}

	c, a := cmd.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"upload", "from-google-photos",
		"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
		"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
		"--from-album-name=Duplicated album",
		// "--no-ui",
		".",
	})

	// let's start
	err = c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
	}
}

func TestUploadFromGP_issue613(t *testing.T) {
	e2e.InitMyEnv()
	e2e.ResetImmich(t)

	ctx := context.Background()

	c, a := cmd.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"upload", "from-google-photos",
		"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
		"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
		"-u",
		"--from-album-name", "Family & friends",
		// "--no-ui",
		e2e.MyEnv("IMMICHGO_TESTFILES") + "/#613 Segfault on Album/Family & friends",
	})

	// let's start
	err := c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
	}
}

func TestUploadFromGP_issue608(t *testing.T) {
	e2e.InitMyEnv()
	e2e.ResetImmich(t)

	ctx := context.Background()

	c, a := cmd.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"upload", "from-google-photos",
		"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
		"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
		// "--no-ui",
		e2e.MyEnv("IMMICHGO_TESTFILES") + "/#608 missing temp files/takeout",
	})

	// let's start
	err := c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
	}
}

func TestUploadFromGooglePhotosPeopleTag(t *testing.T) {
	e2e.InitMyEnv()
	e2e.ResetImmich(t)

	ctx := context.Background()

	c, a := cmd.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"upload", "from-google-photos",
		"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
		"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
		"--log-level=DEBUG",
		"--people-tag=true",
		// "--no-ui",
		e2e.MyEnv("IMMICHGO_TESTFILES") + "/#713 Tag People",
	})

	// let's start
	err := c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
	}
}
