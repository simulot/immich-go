//go:build e2e

package client

import (
	"testing"

	"github.com/simulot/immich-go/app/root"
	e2eutils "github.com/simulot/immich-go/internal/e2e/e2eUtils"
	"github.com/simulot/immich-go/internal/fileevent"
)

func Test_FromFolder(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		adm, err := getUser("admin@immich.app")
		if err != nil {
			t.Fatalf("can't get admin user: %v", err)
		}
		// A fresh user for a new test
		u1, err := createUser("minimal")
		if err != nil {
			t.Fatalf("can't create user: %v", err)
		}

		ctx := t.Context()
		c, a := root.RootImmichGoCommand(ctx)
		c.SetArgs([]string{
			// "--concurrent-tasks=0", // for debugging
			"upload", "from-folder",
			"--server=" + ImmichURL,
			"--api-key=" + u1.APIKey,
			"--admin-api-key=" + adm.APIKey,
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
		}, false, a.FileProcessor())
	})
	t.Run("duplicates", func(t *testing.T) {
		adm, err := getUser("admin@immich.app")
		if err != nil {
			t.Fatalf("can't get admin user: %v", err)
		}
		// A fresh user for a new test
		u1, err := createUser("minimal")
		if err != nil {
			t.Fatalf("can't create user: %v", err)
		}

		ctx := t.Context()
		c, a := root.RootImmichGoCommand(ctx)
		c.SetArgs([]string{
			"--concurrent-tasks=0", // for debugging
			"upload", "from-folder",
			"--server=" + ImmichURL,
			"--api-key=" + u1.APIKey,
			"--admin-api-key=" + adm.APIKey,
			"--no-ui",
			"--api-trace",
			"--log-level=debug",
			"DATA/fromFolder/duplicates",
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
			fileevent.Uploaded:               2,
			fileevent.AnalysisLocalDuplicate: 2,
			fileevent.UploadAddToAlbum:       0,
			fileevent.Tagged:                 0,
		}, false, a.FileProcessor())
	})
}
