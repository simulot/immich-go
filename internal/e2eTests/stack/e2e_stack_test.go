package stack_test

import (
	"context"
	"fmt"
	"os/exec"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/simulot/immich-go/app/cmd"
	"github.com/simulot/immich-go/immich"
)

func TestResetImmich(t *testing.T) {
	initMyEnv(t)
	reset_immich(t)
}

func TestStackBurst(t *testing.T) {
	ctx := context.Background()

	initMyEnv(t)

	reset_immich(t)
	c, a := cmd.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"upload", "from-folder",
		"--server=" + myEnv["IMMICHGO_SERVER"],
		"--api-key=" + myEnv["IMMICHGO_APIKEY"],
		"--no-ui",
		"--dry-run=false",
		// "--manage-burst=Stack",
		"--api-trace",
		"--log-level=debug",
		myEnv["IMMICHGO_TESTFILES"] + "/burst/storm",
		// myEnv["IMMICHGO_TESTFILES"] + "/burst/storm full",
	})

	err := c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
		t.Fatal(err)
	}

	client, err := immich.NewImmichClient(
		myEnv["IMMICHGO_SERVER"],
		myEnv["IMMICHGO_APIKEY"],
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
		"--server=" + myEnv["IMMICHGO_SERVER"],
		"--api-key=" + myEnv["IMMICHGO_APIKEY"],
		// "--dry-run=false",
		"--manage-burst=Stack",
		"--api-trace",
	})
	err = c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
		t.Fatal(err)
	}
}

var myEnv map[string]string

func initMyEnv(t *testing.T) {
	if len(myEnv) > 0 {
		return
	}
	var err error
	e, err := godotenv.Read("../../../e2e.env")
	if err != nil {
		t.Fatalf("cant initialize environment variables: %s", err)
	}
	myEnv = e
	if myEnv["IMMICHGO_TESTFILES"] == "" {
		t.Fatal("missing IMMICHGO_TESTFILES in .env file")
	}
}

func reset_immich(t *testing.T) {
	// Reset immich's database
	// https://github.com/immich-app/immich/blob/main/e2e/src/utils.ts
	//
	c := exec.Command("docker", "exec", "-i", "immich_postgres", "psql", "--dbname=immich", "--username=postgres", "-c",
		`
		DELETE FROM asset_stack CASCADE;
		DELETE FROM libraries CASCADE;
		DELETE FROM shared_links CASCADE;
		DELETE FROM person CASCADE;
		DELETE FROM albums CASCADE;
		DELETE FROM assets CASCADE;
		DELETE FROM asset_faces CASCADE;
		DELETE FROM activity CASCADE;
		--DELETE FROM api_keys CASCADE;
		--DELETE FROM sessions CASCADE;
		--DELETE FROM users CASCADE;
		DELETE FROM "system_metadata" where "key" NOT IN ('reverse-geocoding-state', 'system-flags');
		DELETE FROM tags CASCADE;
		`,
	)
	b, err := c.CombinedOutput()
	if err != nil {
		t.Log(string(b))
		t.Fatal(err)
	}
}
