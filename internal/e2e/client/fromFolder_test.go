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
			fileevent.ProcessedUploadSuccess: 40,
			fileevent.ProcessedAlbumAdded:    0,
			fileevent.ProcessedTagged:        0,
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
			fileevent.ProcessedUploadSuccess:  2,
			fileevent.DiscardedLocalDuplicate: 2,
			fileevent.ProcessedAlbumAdded:     0,
			fileevent.ProcessedTagged:         0,
		}, false, a.FileProcessor())
	})
	t.Run("into-album", func(t *testing.T) {
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
			"--into-album=bananas",
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
			fileevent.ProcessedUploadSuccess: 40,
			fileevent.ProcessedAlbumAdded:    40,
			fileevent.ProcessedTagged:        0,
		}, false, a.FileProcessor())
	})
	t.Run("folder-as-tags", func(t *testing.T) {
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
			"--folder-as-tags=true",
			"--pause-immich-jobs=false", // has to scan sidecars to get tags immediately
			"--no-ui",
			"--api-trace",
			"--log-level=debug",
			"DATA/fromFolder/folder-as-tags-test",
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
			fileevent.ProcessedUploadSuccess: 4,
			fileevent.ProcessedAlbumAdded:    0,
			fileevent.ProcessedTagged:        4,
		}, false, a.FileProcessor())

		// Map filenames to expected tags (derived from folder structure)
		e2eutils.VerifyTagList(t, u1.Email, u1.Password, map[string][]string{
			"telescopes_01.jpg": {"folder-as-tags-test/one/same"},
			"telescopes_02.jpg": {"folder-as-tags-test/one/unique1"},
			"telescopes_03.jpg": {"folder-as-tags-test/2/same"},
			"telescopes_04.jpg": {"folder-as-tags-test/2/unique2"},
		})
	})
}
