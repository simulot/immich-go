package upload

import (
	"context"
	"testing"

	"github.com/simulot/immich-go/app/cmd"
	"github.com/simulot/immich-go/internal/e2eTests/e2e"
)

func Test_BadRequest(t *testing.T) {
	t.Log("Test_BadRequest")
	e2e.InitMyEnv()
	e2e.ResetImmich(t)

	ctx := context.Background()

	c, a := cmd.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"upload", "from-google-photos",
		"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
		"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
		"--no-ui",
		"--api-trace",
		e2e.MyEnv("IMMICHGO_TESTFILES") + "/bad request/Google_Photos",
	})

	// let's start
	err := c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
	}
}
