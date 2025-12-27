//go:build e2e

package client

import (
	"context"
	"testing"

	"github.com/simulot/immich-go/app/root"
	e2eutils "github.com/simulot/immich-go/internal/e2e/e2eUtils"
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
		capture := e2eutils.NewStatsCapture()
		testCtx := context.WithValue(ctx, "test-stats-capture", capture)

		c, _ := root.RootImmichGoCommand(testCtx)
		c.SetArgs([]string{
			"upload", "from-folder",
			"--server=" + ImmichURL,
			"--api-key=" + u1.APIKey,
			"--admin-api-key=" + adm.APIKey,
			"--tui-experimental",
			"--ui=off",
			"--log-level=debug",
			"DATA/fromFolder/recursive",
		})
		err = c.ExecuteContext(testCtx)
		if err != nil {
			t.Error("Unexpected error", err)
			return
		}

		r := capture.Last(t)
		if r.Uploaded != 40 {
			t.Fatalf("uploaded=%d, want 40", r.Uploaded)
		}
		if r.Discarded != 0 {
			t.Fatalf("discarded=%d, want 0", r.Discarded)
		}
		if r.Failed != 0 {
			t.Fatalf("failed=%d, want 0", r.Failed)
		}
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
		capture := e2eutils.NewStatsCapture()
		testCtx := context.WithValue(ctx, "test-stats-capture", capture)

		c, _ := root.RootImmichGoCommand(testCtx)
		c.SetArgs([]string{
			"--concurrent-tasks=0", // for debugging
			"upload", "from-folder",
			"--server=" + ImmichURL,
			"--api-key=" + u1.APIKey,
			"--admin-api-key=" + adm.APIKey,
			"--tui-experimental",
			"--ui=off",
			"--api-trace",
			"--log-level=debug",
			"DATA/fromFolder/duplicates",
		})
		err = c.ExecuteContext(testCtx)
		if err != nil {
			t.Error("Unexpected error", err)
			return
		}

		r := capture.Last(t)
		if r.Uploaded != 2 {
			t.Fatalf("uploaded=%d, want 2", r.Uploaded)
		}
		if r.Discarded != 2 {
			t.Fatalf("discarded=%d, want 2", r.Discarded)
		}
		if r.Failed != 0 {
			t.Fatalf("failed=%d, want 0", r.Failed)
		}
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
		capture := e2eutils.NewStatsCapture()
		testCtx := context.WithValue(ctx, "test-stats-capture", capture)

		c, _ := root.RootImmichGoCommand(testCtx)
		c.SetArgs([]string{
			"upload", "from-folder",
			"--server=" + ImmichURL,
			"--api-key=" + u1.APIKey,
			"--admin-api-key=" + adm.APIKey,
			"--into-album=bananas",
			"--tui-experimental",
			"--ui=off",
			"--api-trace",
			"--log-level=debug",
			"DATA/fromFolder/recursive",
		})
		err = c.ExecuteContext(testCtx)
		if err != nil {
			t.Error("Unexpected error", err)
			return
		}

		r := capture.Last(t)
		if r.Uploaded != 40 {
			t.Fatalf("uploaded=%d, want 40", r.Uploaded)
		}
		if r.Failed != 0 {
			t.Fatalf("failed=%d, want 0", r.Failed)
		}
	})
}
