//go:build e2e
// +build e2e

package upload

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/simulot/immich-go/app/cmd"
	"github.com/simulot/immich-go/internal/e2eTests/e2e"
)

func Test_BadRequestOnPurpose(t *testing.T) {
	client, err := e2e.GetImmichClient()
	if err != nil {
		t.Fatal(err)
	}

	logfileName := os.TempDir() + "/immich.log"
	logfile, err := os.Create(logfileName)
	if err != nil {
		t.Fatal(err)
		return
	}
	defer logfile.Close()

	client.EnableAppTrace(logfile)
	u, err := client.ValidateConnection(context.Background())
	_ = u
	if err != nil {
		t.Fatal(err)
		return
	}

	err = client.DeleteAssets(context.Background(), []string{"bad-request"}, false)
	if err == nil {
		t.Fatal("Error expected")
		return
	}
}

func Test_BadRequest(t *testing.T) {
	t.Log("Test_BadRequest")

	client, err := e2e.GetImmichClient()
	if err != nil {
		t.Fatal(err)
	}

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
		"TEST_DATA/fixtures",
	})

	err = c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
		t.Fatal(err)
	}

	e2e.WaitingForJobsEnding(ctx, client, t)

	c, a = cmd.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"upload", "from-google-photos",
		"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
		"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
		"--no-ui",
		"--api-trace",
		"--log-level=debug",
		"TEST_DATA/takeout",
	})

	err = c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
		t.Fatal(err)
	}

	e2e.WaitingForJobsEnding(ctx, client, t)

	c, a = cmd.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"upload", "from-folder",
		"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
		"--api-key=" + e2e.MyEnv("IMMICHGO_ALTERNATE_APIKEY"),
		"--no-ui",
		"--api-trace",
		"--log-level=debug",
		"TEST_DATA/fixtures",
	})

	err = c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
		t.Fatal(err)
	}

	c, a = cmd.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"upload", "from-google-photos",
		"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
		"--api-key=" + e2e.MyEnv("IMMICHGO_ALTERNATE_APIKEY"),
		"--no-ui",
		"--api-trace",
		"--log-level=debug",
		"TEST_DATA/takeout",
	})
	err = c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
		t.Fatal(err)
	}
}

func Test_BadRequestByAzwillnj(t *testing.T) {
	t.Log("Test_BadRequestByAzwillnj")

	client, err := e2e.GetImmichClient()
	if err != nil {
		t.Fatal(err)
	}

	e2e.InitMyEnv()
	e2e.ResetImmich(t)

	ctx := context.Background()

	// Upload with immich-go v0.21.1
	err = exec.CommandContext(ctx, e2e.MyEnv("IMMICHGO_TESTFILES")+"/#700 Error 500 when upload/lbmh/immich-go.v0.21.1",
		"upload",
		"-server="+e2e.MyEnv("IMMICHGO_SERVER"),
		"-key="+e2e.MyEnv("IMMICHGO_APIKEY"),
		"-google-photos",
		e2e.MyEnv("IMMICHGO_TESTFILES")+"/#700 Error 500 when upload/azwillnj/takeout/Photos from 2016").Run()
	if err != nil {
		t.Log(err)
	}

	e2e.WaitingForJobsEnding(ctx, client, t)

	// Same upload with current immich-go
	c, a := cmd.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"upload", "from-google-photos",
		"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
		"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
		"--no-ui",
		"--api-trace",
		"--log-level=debug",
		e2e.MyEnv("IMMICHGO_TESTFILES") + "/#700 Error 500 when upload/azwillnj/takeout",
	})

	err = c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
		t.Fatal(err)
	}

	e2e.WaitingForJobsEnding(ctx, client, t)
}
