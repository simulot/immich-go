//go:build e2e

package fromfolder

import (
	"path"
	"testing"

	"github.com/simulot/immich-go/app/root"
	e2eutils "github.com/simulot/immich-go/internal/e2e/e2eUtils"
	"github.com/simulot/immich-go/internal/fileevent"
)

const (
	immichTestPath = "../immich-test"
	immichURL      = "http://localhost:2283"
	u1KeyPath      = "users/user1@immich.app/keys/e2eMinimal"
	admKeyPath     = "users/admin@immich.app/keys/e2eAll"
)

func resetImmich(t *testing.T) {
	ic, err := e2eutils.OpenImmichController(immichTestPath)
	if err != nil {
		t.Fatalf("can't create immich controller: %s", err.Error())
	}
	err = ic.ResetImmich(t.Context())
	if err != nil {
		t.Fatalf("can't create immich controller: %s", err.Error())
	}
	err = ic.WaitAPI(t.Context())
	if err != nil {
		t.Fatalf("can't create immich controller: %s", err.Error())
	}
}

func Test_FromFolder(t *testing.T) {
	keys, err := e2eutils.KeysFromFile(path.Join(immichTestPath, "e2eusers.yml"))
	if err != nil {
		t.Fatalf("Can't get the keys: %s", err.Error())
	}

	resetImmich(t)
	u1Key := keys.Get(u1KeyPath)
	admKey := keys.Get(admKeyPath)

	ctx := t.Context()
	c, a := root.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"upload", "from-folder",
		"--server=" + immichURL,
		"--api-key=" + u1Key,
		"--admin-api-key=" + admKey,
		"--no-ui",
		"--api-trace",
		"--log-level=debug",
		"DATA/fromFolder/recursive",
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
		fileevent.Uploaded:         40,
		fileevent.UploadAddToAlbum: 0,
		fileevent.Tagged:           0,
	}, false, a.Jnl())
}
