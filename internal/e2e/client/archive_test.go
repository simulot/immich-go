//go:build e2e

package client

import (
	"os"
	"testing"

	"github.com/simulot/immich-go/app/root"
	e2eutils "github.com/simulot/immich-go/internal/e2e/e2eUtils"
	"github.com/simulot/immich-go/internal/fileevent"
)

func Test_ArchiveFromGP(t *testing.T) {
	t.Run("ArchiveFromFolder", func(t *testing.T) {
		// patch3 
		// patch 4
		tempDir, err := os.MkdirTemp("", "immich-go-test-ArchiveFromFolder*")
		if err != nil {
			t.Fatal(err.Error())
		}
		t.Cleanup(func() {
			os.RemoveAll(tempDir)
		})

		ctx := t.Context()
		c, a := root.RootImmichGoCommand(ctx)
		c.SetArgs([]string{
			// "--concurrent-tasks=0", // for debugging
			"archive",
			"--write-to-folder=/tmp/immich-go-test",
			"from-folder",
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
			fileevent.DiscoveredImage:        40,
			fileevent.ProcessedFileArchived:  40,
			fileevent.ProcessedUploadSuccess: 0,
			fileevent.ProcessedAlbumAdded:    0,
			fileevent.ProcessedTagged:        0,
		}, false, a.FileProcessor())
	})
	t.Run("ArchiveFromGP", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "immich-go-test-ArchiveFromGP*")
		if err != nil {
			t.Fatal(err.Error())
		}
		t.Cleanup(func() {
			os.RemoveAll(tempDir)
		})

		ctx := t.Context()
		c, a := root.RootImmichGoCommand(ctx)
		c.SetArgs([]string{
			// "--concurrent-tasks=0", // for debugging
			"archive",
			"--write-to-folder=" + tempDir,
			"from-google-photos",
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
			fileevent.DiscoveredImage:         10,
			fileevent.ProcessedUploadSuccess:  0,
			fileevent.ProcessedAlbumAdded:     0,
			fileevent.ProcessedTagged:         0,
			fileevent.DiscardedLocalDuplicate: 5,
		}, false, a.FileProcessor())
	})
	t.Run("ArchiveFromImmich", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "immich-go-test-ArchiveFromImmich*")
		if err != nil {
			t.Fatal(err.Error())
		}
		t.Cleanup(func() {
			os.RemoveAll(tempDir)
		})

		// 1. Upload photos on immich

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
			"upload", "from-google-photos",
			"--server=" + ImmichURL,
			"--api-key=" + u1.APIKey,
			"--admin-api-key=" + adm.APIKey,
			"--no-ui",
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

		// e2eutils.CheckResults(t, map[fileevent.Code]int64{
		// 	fileevent.ProcessedUploadSuccess:     5,
		// 	fileevent.ProcessedAlbumAdded: 5,
		// 	fileevent.ProcessedTagged:     5,
		// }, false, a.FileProcessor())

		// 2. Archive from-immich
		c, a = root.RootImmichGoCommand(ctx)
		c.SetArgs([]string{
			"archive",
			"--write-to-folder=" + tempDir,
			"from-immich",
			"--from-server=" + ImmichURL,
			"--from-api-key=" + u1.APIKey,
			"--from-admin-api-key=" + adm.APIKey,
			"--from-api-trace",
			"--log-level=debug",
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
			fileevent.DiscoveredImage:       5,
			fileevent.ProcessedFileArchived: 5,
		}, false, a.FileProcessor())
	})
}
