//go:build e2e

package client

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/simulot/immich-go/app/root"
	e2eutils "github.com/simulot/immich-go/internal/e2e/e2eUtils"
)

func Test_Sync(t *testing.T) {
	t.Run("sync_up_basic", func(t *testing.T) {
		u1, err := createUser("all")
		if err != nil {
			t.Fatalf("can't create user: %v", err)
		}

		ctx := t.Context()
		c, _ := root.RootImmichGoCommand(ctx)
		c.SetArgs([]string{
			"sync", "up",
			"--server=" + ImmichURL,
			"--api-key=" + u1.APIKey,
			"--log-level=debug",
			"-d", "DATA/fromFolder/recursive",
		})

		err = c.ExecuteContext(ctx)
		if err != nil {
			t.Fatalf("sync up failed: %v", err)
		}

		// Verify server has the assets
		assets, err := e2eutils.GetAllAssets(u1.Email, u1.Password)
		if err != nil {
			t.Fatalf("GetAllAssets: %v", err)
		}

		if len(assets) != 40 {
			t.Errorf("expected 40 assets on server, got %d", len(assets))
		}
	})

	t.Run("sync_down_basic", func(t *testing.T) {
		u1, err := createUser("all")
		if err != nil {
			t.Fatalf("can't create user: %v", err)
		}

		ctx := t.Context()

		// Step 1: Upload 40 assets via upload command
		c, a := root.RootImmichGoCommand(ctx)
		c.SetArgs([]string{
			"upload", "from-folder",
			"--server=" + ImmichURL,
			"--api-key=" + u1.APIKey,
			"--no-ui",
			"--log-level=debug",
			"DATA/fromFolder/recursive",
		})
		err = c.ExecuteContext(ctx)
		if err != nil && a.Log().GetSLog() != nil {
			a.Log().Error(err.Error())
		}
		if err != nil {
			t.Fatalf("upload failed: %v", err)
		}

		// Step 2: Sync down to a temp directory
		tmpDir := t.TempDir()
		c2, _ := root.RootImmichGoCommand(ctx)
		c2.SetArgs([]string{
			"sync", "down",
			"--server=" + ImmichURL,
			"--api-key=" + u1.APIKey,
			"--log-level=debug",
			"-d", tmpDir,
		})

		err = c2.ExecuteContext(ctx)
		if err != nil {
			t.Fatalf("sync down failed: %v", err)
		}

		// Step 3: Verify local files exist (count non-hidden, non-directory files)
		localCount := countLocalFiles(t, tmpDir)
		if localCount != 40 {
			t.Errorf("expected 40 local files after sync down, got %d", localCount)
		}
	})

	t.Run("sync_down_idempotent", func(t *testing.T) {
		u1, err := createUser("all")
		if err != nil {
			t.Fatalf("can't create user: %v", err)
		}

		ctx := t.Context()

		// Upload assets
		c, a := root.RootImmichGoCommand(ctx)
		c.SetArgs([]string{
			"upload", "from-folder",
			"--server=" + ImmichURL,
			"--api-key=" + u1.APIKey,
			"--no-ui",
			"--log-level=debug",
			"DATA/fromFolder/recursive",
		})
		err = c.ExecuteContext(ctx)
		if err != nil && a.Log().GetSLog() != nil {
			a.Log().Error(err.Error())
		}
		if err != nil {
			t.Fatalf("upload failed: %v", err)
		}

		tmpDir := t.TempDir()

		// First sync down
		c2, _ := root.RootImmichGoCommand(ctx)
		c2.SetArgs([]string{
			"sync", "down",
			"--server=" + ImmichURL,
			"--api-key=" + u1.APIKey,
			"--log-level=debug",
			"-d", tmpDir,
		})
		err = c2.ExecuteContext(ctx)
		if err != nil {
			t.Fatalf("first sync down failed: %v", err)
		}

		// Record file mod times
		modTimes := collectModTimes(t, tmpDir)

		// Second sync down — should be a no-op
		c3, _ := root.RootImmichGoCommand(ctx)
		c3.SetArgs([]string{
			"sync", "down",
			"--server=" + ImmichURL,
			"--api-key=" + u1.APIKey,
			"--log-level=debug",
			"-d", tmpDir,
		})
		err = c3.ExecuteContext(ctx)
		if err != nil {
			t.Fatalf("second sync down failed: %v", err)
		}

		// Verify no files were re-downloaded (mod times unchanged)
		modTimes2 := collectModTimes(t, tmpDir)
		for path, mt := range modTimes {
			if mt2, ok := modTimes2[path]; ok {
				if !mt.Equal(mt2) {
					t.Errorf("file %s was re-downloaded (mod time changed: %v → %v)", path, mt, mt2)
				}
			}
		}
	})

	t.Run("sync_up_idempotent", func(t *testing.T) {
		u1, err := createUser("all")
		if err != nil {
			t.Fatalf("can't create user: %v", err)
		}

		ctx := t.Context()

		// First sync up
		c, _ := root.RootImmichGoCommand(ctx)
		c.SetArgs([]string{
			"sync", "up",
			"--server=" + ImmichURL,
			"--api-key=" + u1.APIKey,
			"--log-level=debug",
			"-d", "DATA/fromFolder/recursive",
		})
		err = c.ExecuteContext(ctx)
		if err != nil {
			t.Fatalf("first sync up failed: %v", err)
		}

		assets1, err := e2eutils.GetAllAssets(u1.Email, u1.Password)
		if err != nil {
			t.Fatalf("GetAllAssets after first sync: %v", err)
		}
		if len(assets1) != 40 {
			t.Fatalf("expected 40 assets after first sync, got %d", len(assets1))
		}

		// Second sync up — should detect all assets as SameOnServer
		c2, _ := root.RootImmichGoCommand(ctx)
		c2.SetArgs([]string{
			"sync", "up",
			"--server=" + ImmichURL,
			"--api-key=" + u1.APIKey,
			"--log-level=debug",
			"-d", "DATA/fromFolder/recursive",
		})
		err = c2.ExecuteContext(ctx)
		if err != nil {
			t.Fatalf("second sync up failed: %v", err)
		}

		assets2, err := e2eutils.GetAllAssets(u1.Email, u1.Password)
		if err != nil {
			t.Fatalf("GetAllAssets after second sync: %v", err)
		}

		// Should still be exactly 40 — no duplicates created
		if len(assets2) != 40 {
			t.Errorf("expected 40 assets after second sync (no duplicates), got %d", len(assets2))
		}
	})
}

// countLocalFiles counts non-hidden files in a directory tree,
// excluding the .immich-sync state directory.
func countLocalFiles(t *testing.T, dir string) int {
	t.Helper()
	count := 0
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}
		count++
		return nil
	})
	return count
}

// collectModTimes returns a map of relative path → mod time for all files.
func collectModTimes(t *testing.T, dir string) map[string]time.Time {
	t.Helper()
	result := make(map[string]time.Time)
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}
		relPath, _ := filepath.Rel(dir, path)
		info, _ := d.Info()
		if info != nil {
			result[relPath] = info.ModTime()
		}
		return nil
	})
	return result
}
