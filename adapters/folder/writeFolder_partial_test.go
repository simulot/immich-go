package folder

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/simulot/immich-go/internal/fshelper/osfs"
)

// noRenameFS is an archive folder where a rename fails, as it does on a full disk or a lost mount.
type noRenameFS struct{ *archiveFS }

func (noRenameFS) Rename(string, string) error { return errors.New("rename refused") }

// A .partial file whose rename fails is removed: a complete copy left under a temporary name is
// never used, and only takes space.
func TestWriteAssetRenameErrorRemovesPartial(t *testing.T) {
	t.Setenv("IMMICHGO_TEMPDIR", t.TempDir())
	body := "the asset's bytes"
	srcDir, dst := t.TempDir(), t.TempDir()
	writeSources(t, srcDir, map[string]string{"a": body})
	src := &countingFS{FS: os.DirFS(srcDir), opens: map[string]int{}}
	w, err := NewLocalAssetWriter(noRenameFS{newArchiveFS(dst)})
	if err != nil {
		t.Fatal(err)
	}
	a := newImmichAsset(src, "a", "IMG_1.jpg", len(body), sha1Base64(body))
	err = w.WriteAsset(context.Background(), a)
	_ = a.Close()
	if err == nil || !strings.Contains(err.Error(), "rename refused") {
		t.Fatalf("WriteAsset: %v; want the rename's error", err)
	}
	if files := listFiles(t, dst); len(files) != 0 {
		t.Fatalf("files %q left; want none (no original, no .partial, no sidecar)", files)
	}
}

// hookReader calls hook once, before its first read.
type hookReader struct {
	io.Reader
	hook func()
}

func (h *hookReader) Read(p []byte) (int, error) {
	if h.hook != nil {
		hook := h.hook
		h.hook = nil
		hook()
	}
	return h.Reader.Read(p)
}

// Two runs that write the same name at the same time each write their own .partial file: a run
// never writes into, renames or removes the other's. Here run B writes the name while run A has
// its .partial file open and has written nothing yet. Both succeed, the name holds A's bytes,
// whole (A renames last), and no .partial file is left.
func TestWriteAtomicOwnPartialFile(t *testing.T) {
	dst := t.TempDir()
	fsys := osfs.DirFS(dst)
	bodyA, bodyB := "the bytes run A writes", "run B's"
	var errB error
	rA := &hookReader{Reader: strings.NewReader(bodyA), hook: func() {
		_, errB = writeAtomic(fsys, "IMG_1.jpg", strings.NewReader(bodyB), int64(len(bodyB)), "")
	}}
	_, errA := writeAtomic(fsys, "IMG_1.jpg", rA, int64(len(bodyA)), "")
	if errA != nil || errB != nil {
		t.Fatalf("run A: %v, run B: %v; want both to succeed", errA, errB)
	}
	if b, err := os.ReadFile(filepath.Join(dst, "IMG_1.jpg")); err != nil || string(b) != bodyA {
		t.Fatalf("IMG_1.jpg holds %q (%v), want %q", b, err, bodyA)
	}
	entries, err := os.ReadDir(dst)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != "IMG_1.jpg" {
			t.Fatalf("%s left beside IMG_1.jpg", e.Name())
		}
	}
}

// The .partial file is hidden, beside the file it becomes, and named after it.
func TestWriteAtomicPartialName(t *testing.T) {
	dst := t.TempDir()
	fsys := osfs.DirFS(dst)
	if err := os.Mkdir(filepath.Join(dst, "d"), 0o755); err != nil {
		t.Fatal(err)
	}
	var partial []fs.DirEntry
	r := &hookReader{Reader: strings.NewReader("bytes"), hook: func() {
		var err error
		if partial, err = os.ReadDir(filepath.Join(dst, "d")); err != nil {
			t.Error(err)
		}
	}}
	if _, err := writeAtomic(fsys, "d/IMG_1.jpg", r, 0, ""); err != nil {
		t.Fatal(err)
	}
	if len(partial) != 1 || !strings.HasPrefix(partial[0].Name(), ".IMG_1.jpg.") || !strings.HasSuffix(partial[0].Name(), ".partial") ||
		len(partial[0].Name()) <= len(".IMG_1.jpg..partial") {
		t.Fatalf("files while writing %v, want one .IMG_1.jpg.<random>.partial", partial)
	}
}
