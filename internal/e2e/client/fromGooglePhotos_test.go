//go:build e2e

package client

import (
	"testing"

	"github.com/simulot/immich-go/app/root"
	e2eutils "github.com/simulot/immich-go/internal/e2e/e2eUtils"
	"github.com/simulot/immich-go/internal/fileevent"
)

func Test_FromGooglePhotos(t *testing.T) {
	// Load user credentials
	keys, err := e2eutils.KeysFromFile(keysFile)
	if err != nil {
		t.Fatalf("Can't get the keys from %s: %s", keysFile, err.Error())
	}

	// Reset Immich before test
	resetImmich(t)

	// Get API keys for users
	u1Key := keys.Get(u1KeyPath)
	admKey := keys.Get(admKeyPath)

	if u1Key == "" || admKey == "" {
		t.Fatalf("Missing required API keys in %s", keysFile)
	}

	ctx := t.Context()
	c, a := root.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"upload", "from-google-photos",
		"--server=" + immichURL,
		"--api-key=" + u1Key,
		"--admin-api-key=" + admKey,
		"--no-ui",
		// "--api-trace",
		"--log-level=debug",
		"DATA/fromGooglePhotos/gophers",
	})
	err = c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
	}

	if err != nil {
		t.Error("Unexpected error", err)
		return
	}

	e2eutils.CheckResults(t, map[fileevent.Code]int64{
		fileevent.Uploaded:         5,
		fileevent.UploadAddToAlbum: 5,
		fileevent.Tagged:           5,
	}, false, a.Jnl())
}
