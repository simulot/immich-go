package upload

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/simulot/immich-go/app/cmd"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/e2eTests/e2e"
)

func Test_BadRequest(t *testing.T) {
	t.Log("Test_BadRequest")
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

	err := c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
		t.Fatal(err)
	}

	client, err := immich.NewImmichClient(
		e2e.MyEnv("IMMICHGO_SERVER"),
		e2e.MyEnv("IMMICHGO_APIKEY"),
	)
	ctx2, cancel := context.WithDeadline(ctx, time.Now().Add(30*time.Second))
check:
	for {
		select {
		case <-ctx2.Done():
			t.Fatal("Timeout waiting for metadata job to terminate")
		default:
			jobs, err := client.GetJobs(ctx2)
			if err != nil {
				t.Fatal(err)
			}
			if jobs["metadataExtraction"].JobCounts.Active == 0 {
				cancel()
				break check
			}
			fmt.Println("Waiting for metadata extraction to finish")
			time.Sleep(1 * time.Second)
		}
	}

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
