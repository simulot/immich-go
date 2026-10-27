package folder

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/simulot/immich-go/internal/fshelper/osfs"
)

// An edit list that could not be read from the server (EditsUnread) keeps the one the .JSON file
// has. When that file cannot be read or parsed, its list cannot be kept: the writer logs one
// warning that names it, and the asset is archived all the same. A missing .JSON file is no loss.
func TestEditsUnreadSidecarUnreadable(t *testing.T) {
	for _, tc := range []struct {
		name     string
		sidecar  func(t *testing.T, p string) // spoils the .JSON file p
		warnings int
	}{
		{"missing", func(t *testing.T, p string) {
			if err := os.Remove(p); err != nil {
				t.Fatal(err)
			}
		}, 0},
		{"not JSON", func(t *testing.T, p string) {
			if err := os.WriteFile(p, []byte("{not JSON"), 0o644); err != nil {
				t.Fatal(err)
			}
		}, 1},
		{"unreadable", func(t *testing.T, p string) {
			if err := os.Chmod(p, 0); err != nil {
				t.Fatal(err)
			}
			if _, err := os.ReadFile(p); err == nil {
				t.Skip("this file system lets the test read a file of mode 000")
			}
		}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("IMMICHGO_TEMPDIR", t.TempDir())
			srcDir, dst := t.TempDir(), t.TempDir()
			const body = "photo"
			writeSources(t, srcDir, map[string]string{"IMG_1.jpg": body})
			src := &countingFS{FS: os.DirFS(srcDir), opens: map[string]int{}}
			a := newImmichAsset(src, "IMG_1.jpg", "IMG_1.jpg", len(body), sha1Base64(body))
			a.FromApplication.EditsUnread = true

			w, err := NewLocalAssetWriter(osfs.DirFS(dst))
			if err != nil {
				t.Fatal(err)
			}
			var log bytes.Buffer
			w.SetLogger(slog.New(slog.NewTextHandler(&log, nil)))
			name := path.Join(w.pathOfAsset(a), "IMG_1.jpg")
			writeArchived(t, dst, name, body, sha1Base64(body))
			tc.sidecar(t, filepath.Join(dst, filepath.FromSlash(name)+".JSON"))

			err = w.WriteAsset(context.Background(), a)
			_ = a.Close()
			if err != nil {
				t.Fatal(err)
			}
			if got := listFiles(t, dst); len(got) != 2 {
				t.Fatalf("files %v, want the photo and its .JSON file", got)
			}
			var warnings []string
			for line := range strings.Lines(log.String()) {
				if strings.Contains(line, "edit list") {
					warnings = append(warnings, line)
				}
			}
			if len(warnings) != tc.warnings {
				t.Fatalf("%d warnings about the edit list, want %d:\n%s", len(warnings), tc.warnings, log.String())
			}
			for _, l := range warnings {
				if !strings.Contains(l, "level=WARN") || !strings.Contains(l, "IMG_1.jpg.JSON") {
					t.Fatalf("the warning is not one that names the .JSON file: %s", l)
				}
			}
		})
	}
}
