package upload

import (
	"context"
	"os"
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
		"TEST_DATA/takeout",
	})

	err = c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
		t.Fatal(err)
	}
}
