package folder

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/simulot/immich-go/internal/fshelper/osfs"
)

// An original whose bytes do not match the asset's known size (a download that ended early with
// no transport error, as a body with no length framing would) is an error: its .partial file is
// removed, never renamed, and no sidecar is written. With the size unknown (0) it is written as is.
func TestWriteAssetRefusesWrongSize(t *testing.T) {
	t.Setenv("IMMICHGO_TEMPDIR", t.TempDir())
	srcDir := t.TempDir()
	body := "only part of the original"
	if err := os.WriteFile(filepath.Join(srcDir, "a"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name  string
		size  int
		wrote bool
	}{
		{"shorter than the asset", len(body) + 100, false},
		{"longer than the asset", len(body) - 5, false},
		{"the asset's size", len(body), true},
		{"size unknown", 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dst := t.TempDir()
			src := &countingFS{FS: os.DirFS(srcDir), opens: map[string]int{}}
			w, err := NewLocalAssetWriter(osfs.DirFS(dst))
			if err != nil {
				t.Fatal(err)
			}
			a := newAsset(src, "a", "IMG_0001.JPG", tc.size, "")
			err = w.WriteAsset(context.Background(), a)
			_ = a.Close()
			files := listFiles(t, dst)
			if tc.wrote {
				if err != nil || len(files) != 2 {
					t.Fatalf("WriteAsset: %v, files %q; want the original and its sidecar", err, files)
				}
				got, err := os.ReadFile(filepath.Join(dst, "2021", "2021-07", "IMG_0001.JPG"))
				if err != nil || string(got) != body {
					t.Fatalf("original %q, %v", got, err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), "bytes") {
				t.Fatalf("WriteAsset: %v; want an error about the size", err)
			}
			if len(files) != 0 {
				t.Fatalf("files %q left; want none (no original, no .partial, no sidecar)", files)
			}
		})
	}
}
