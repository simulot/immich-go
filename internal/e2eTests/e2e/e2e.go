package e2e

import (
	"context"
	"fmt"
	"os/exec"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/simulot/immich-go/immich"
)

var myEnv map[string]string

const e2eEnv = "../../../e2e.env"

func InitMyEnv() {
	if len(myEnv) > 0 {
		return
	}
	var err error
	e, err := godotenv.Read(e2eEnv)
	if err != nil {
		panic(fmt.Sprintf("cant initialize environment variables: %s", err))
	}
	myEnv = e
	if myEnv["IMMICHGO_TESTFILES"] == "" {
		panic("missing IMMICHGO_TESTFILES in .env file")
	}
}

func MyEnv(key string) string {
	if len(myEnv) == 0 {
		InitMyEnv()
	}
	return myEnv[key]
}

func ResetImmich(t *testing.T) {
	// Reset immich's database
	// https://github.com/immich-app/immich/blob/main/e2e/src/utils.ts
	//
	c := exec.Command("docker", "exec", "-i", "immich_postgres", "psql", "--dbname=immich", "--username=postgres", "-c",
		`
        delete from stack CASACDE;
        delete from library CASACDE;
        delete from shared_link CASACDE;
        delete from person CASACDE;
        delete from album CASACDE;
        delete from asset CASACDE;
        delete from asset_face CASACDE;
        delete from activity CASACDE;
        -- delete from api_key CASACDE;
        -- delete from session CASACDE;
        -- delete from user CASACDE;
        delete from system_metadata where "key" NOT IN ('reverse-geocoding-state', 'system-flags');
        delete from tag CASACDE;
		`,
	)
	out, err := c.CombinedOutput()
	if err != nil {
		t.Fatal(string(out), err.Error())
	}
}

func WaitingForJobsEnding(ctx context.Context, client *immich.ImmichClient, t *testing.T) {
	// Waiting for jobs to complete
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(30*time.Second))
check:
	for {
		select {
		case <-ctx.Done():
			t.Fatal("Timeout waiting for metadata job to terminate")
		default:
			jobs, err := client.GetJobs(ctx)
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
}

func GetImmichClient() (*immich.ImmichClient, error) {
	return immich.NewImmichClient(
		MyEnv("IMMICHGO_SERVER"),
		MyEnv("IMMICHGO_APIKEY"),
	)
}
