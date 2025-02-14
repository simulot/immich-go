//go:build e2e
// +build e2e

package stack_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/simulot/immich-go/app/cmd"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/e2eTests/e2e"
)

func TestResetImmich(t *testing.T) {
	e2e.InitMyEnv()
	e2e.ResetImmich(t)
}

func TestStackBurst(t *testing.T) {
	ctx := context.Background()

	e2e.InitMyEnv()
	e2e.ResetImmich(t)

	c, a := cmd.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"upload", "from-folder",
		"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
		"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
		"--no-ui",
		"--dry-run=false",
		"--manage-burst=Stack",
		"--manage-heic-jpeg=StackCoverHeic",
		"--manage-raw-jpeg=StackCoverRaw",
		"--manage-epson-fastfoto=TRUE",
		// "--api-trace",
		// "--log-level=debug",
		e2e.MyEnv("IMMICHGO_TESTFILES") + "/EpsonfastFoto/EpsonFastFoto.zip",
		e2e.MyEnv("IMMICHGO_TESTFILES") + "/burst/Reflex",
		e2e.MyEnv("IMMICHGO_TESTFILES") + "/burst/PXL6",
		e2e.MyEnv("IMMICHGO_TESTFILES") + "/burst/Tel",
		e2e.MyEnv("IMMICHGO_TESTFILES") + "/burst/storm",
		// e2e.MyEnv("IMMICHGO_TESTFILES") + "/burst/storm full",
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
		"stack",
		"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
		"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
		"--api-trace",
		"--log-level=debug",
		// "--dry-run=false",
		"--manage-burst=Stack",
		"--manage-heic-jpeg=StackCoverHeic",
		"--manage-raw-jpeg=StackCoverRaw",
		"--manage-epson-fastfoto=TRUE",
	})
	err = c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
		t.Fatal(err)
	}
}
