package folder

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/simulot/immich-go/internal/fshelper/osfs"
)

// countingFS is a source whose Open calls are counted: an Open is a download. Like an Immich
// download (immichfs), what it returns is a plain fs.File, so Asset.OpenFile copies it through the
// cache reader, which computes its SHA-1; plain (localFS) returns the *os.File, opened in place.
type countingFS struct {
	fs.FS
	opens map[string]int
	plain bool
}

type streamFile struct{ fs.File }

func (c *countingFS) Open(name string) (fs.File, error) {
	c.opens[name]++
	f, err := c.FS.Open(name)
	if err != nil || c.plain {
		return f, err
	}
	return streamFile{f}, nil
}

func newAsset(src *countingFS, file, base string, size int, desc string) *assets.Asset {
	a := &assets.Asset{
		File:        fshelper.FSName(src, file),
		FileSize:    size,
		CaptureDate: time.Date(2021, 7, 4, 12, 30, 0, 0, time.UTC),
	}
	a.SetNameInfo(assets.NameInfo{Base: base})
	a.FromApplication = &assets.Metadata{FileName: base, Description: desc}
	return a
}

func listFiles(t *testing.T, root string) []string {
	var out []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			r, _ := filepath.Rel(root, p)
			out = append(out, r)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(out)
	return out
}

func TestWriteAssetSkipsPresentFiles(t *testing.T) {
	srcDir, dst := t.TempDir(), t.TempDir()
	for name, body := range map[string]string{"a": "first asset", "b": "a different asset, same name"} {
		if err := os.WriteFile(filepath.Join(srcDir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	src := &countingFS{FS: os.DirFS(srcDir), opens: map[string]int{}}
	w, err := NewLocalAssetWriter(osfs.DirFS(dst), ".")
	if err != nil {
		t.Fatal(err)
	}
	run := func(descA string) {
		for _, a := range []*assets.Asset{
			newAsset(src, "a", "IMG_1.JPG", len("first asset"), descA),
			newAsset(src, "b", "IMG_1.JPG", len("a different asset, same name"), ""),
		} {
			if err := w.WriteAsset(context.Background(), a); err != nil {
				t.Fatal(err)
			}
			_ = a.Close()
		}
	}

	run("one")
	want := []string{"2021/2021-07/IMG_1.JPG", "2021/2021-07/IMG_1.JPG.JSON", "2021/2021-07/IMG_1~1.JPG", "2021/2021-07/IMG_1~1.JPG.JSON"}
	if got := listFiles(t, dst); !equal(got, want) {
		t.Fatalf("run 1 wrote %v, want %v", got, want)
	}
	if src.opens["a"] != 1 || src.opens["b"] != 1 {
		t.Fatalf("run 1 opened %v, want each once", src.opens)
	}
	stamp := map[string]time.Time{}
	old := time.Now().Add(-time.Hour)
	for _, f := range want {
		p := filepath.Join(dst, f)
		if err := os.Chtimes(p, old, old); err != nil {
			t.Fatal(err)
		}
		fi, _ := os.Stat(p)
		stamp[f] = fi.ModTime()
	}

	run("one")
	if src.opens["a"] != 1 || src.opens["b"] != 1 {
		t.Fatalf("run 2 downloaded again: opens %v", src.opens)
	}
	if got := listFiles(t, dst); !equal(got, want) {
		t.Fatalf("run 2 left %v, want %v", got, want)
	}
	for _, f := range want {
		fi, _ := os.Stat(filepath.Join(dst, f))
		if !fi.ModTime().Equal(stamp[f]) {
			t.Fatalf("run 2 rewrote %s", f)
		}
	}

	run("two") // a's metadata changed: only its sidecar is rewritten
	for _, f := range want {
		fi, _ := os.Stat(filepath.Join(dst, f))
		changed := !fi.ModTime().Equal(stamp[f])
		if changed != (f == "2021/2021-07/IMG_1.JPG.JSON") {
			t.Fatalf("run 3: %s changed=%v", f, changed)
		}
	}
	if src.opens["a"] != 1 || src.opens["b"] != 1 {
		t.Fatalf("run 3 downloaded again: opens %v", src.opens)
	}

	// A short file under the name (another asset's leftover) is not taken for this asset.
	if err := os.WriteFile(filepath.Join(dst, "2021/2021-07/IMG_2.JPG"), []byte("short"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := newAsset(src, "a", "IMG_2.JPG", len("first asset"), "")
	if err := w.WriteAsset(context.Background(), a); err != nil {
		t.Fatal(err)
	}
	_ = a.Close()
	if _, err := os.Stat(filepath.Join(dst, "2021/2021-07/IMG_2~1.JPG")); err != nil {
		t.Fatalf("a different-size file under the name must push the asset to ~1: %v", err)
	}
	for _, f := range listFiles(t, dst) {
		if filepath.Ext(f) == ".partial" {
			t.Fatalf("temporary file left: %s", f)
		}
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// The premise of sameContent: after Asset.OpenFile on a streamed source (an Immich download),
// a.Checksum is the base64 SHA-1 of the bytes the cache reader wrote.
func TestOpenFileChecksumIsDownloadSHA1(t *testing.T) {
	srcDir := t.TempDir()
	body := "streamed bytes"
	if err := os.WriteFile(filepath.Join(srcDir, "s"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	src := &countingFS{FS: os.DirFS(srcDir), opens: map[string]int{}}
	a := newAsset(src, "s", "S.JPG", 0, "")
	a.Checksum = "stale"
	r, err := a.OpenFile()
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	defer r.Close()
	sum := sha1.Sum([]byte(body))
	if want := base64.StdEncoding.EncodeToString(sum[:]); a.Checksum != want {
		t.Fatalf("a.Checksum after OpenFile is %q, want %q (the download's SHA-1)", a.Checksum, want)
	}
}

// An asset whose size the server does not know yet (FileSize 0) is downloaded again to be
// compared, but never written a second time as ~N; another unknown-size asset with the same name
// and other bytes still gets ~1. Run with a streamed source (the SHA-1 comes from the cache
// reader) and with a local file opened in place (sameContent hashes it itself).
func TestWriteAssetUnknownSizeMakesNoCopy(t *testing.T) {
	for _, plain := range []bool{false, true} {
		t.Run(map[bool]string{false: "streamed", true: "local"}[plain], func(t *testing.T) { unknownSize(t, plain) })
	}
}

func unknownSize(t *testing.T, plain bool) {
	srcDir, dst := t.TempDir(), t.TempDir()
	for name, body := range map[string]string{"z": "size not known yet", "y": "another, also unsized"} {
		if err := os.WriteFile(filepath.Join(srcDir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	src := &countingFS{FS: os.DirFS(srcDir), opens: map[string]int{}, plain: plain}
	w, err := NewLocalAssetWriter(osfs.DirFS(dst), ".")
	if err != nil {
		t.Fatal(err)
	}
	write := func(file string) {
		a := newAsset(src, file, "IMG_3.JPG", 0, "")
		if err := w.WriteAsset(context.Background(), a); err != nil {
			t.Fatal(err)
		}
		_ = a.Close()
	}

	write("z")
	want := []string{"2021/2021-07/IMG_3.JPG", "2021/2021-07/IMG_3.JPG.JSON"}
	if got := listFiles(t, dst); !equal(got, want) {
		t.Fatalf("run 1 wrote %v, want %v", got, want)
	}
	old := time.Now().Add(-time.Hour)
	p := filepath.Join(dst, "2021/2021-07/IMG_3.JPG")
	if err := os.Chtimes(p, old, old); err != nil {
		t.Fatal(err)
	}
	before, _ := os.Stat(p)

	write("z")
	if got := listFiles(t, dst); !equal(got, want) {
		t.Fatalf("run 2 made a copy: %v", got)
	}
	if after, _ := os.Stat(p); !after.ModTime().Equal(before.ModTime()) {
		t.Fatal("run 2 rewrote the original")
	}
	if src.opens["z"] != 2 {
		t.Fatalf("an unsized asset is downloaded once per run to compare: opens %d, want 2", src.opens["z"])
	}

	write("y")
	want = []string{"2021/2021-07/IMG_3.JPG", "2021/2021-07/IMG_3.JPG.JSON", "2021/2021-07/IMG_3~1.JPG", "2021/2021-07/IMG_3~1.JPG.JSON"}
	if got := listFiles(t, dst); !equal(got, want) {
		t.Fatalf("a different unsized asset: %v, want %v", got, want)
	}
	if b, _ := os.ReadFile(filepath.Join(dst, "2021/2021-07/IMG_3~1.JPG")); string(b) != "another, also unsized" {
		t.Fatalf("IMG_3~1.JPG holds %q", b)
	}
}

// With a valid checksum, sameContent compares by it and does not read the download again: here the
// download is open write-only, so any read of it would fail.
func TestSameContentUsesChecksum(t *testing.T) {
	dst := t.TempDir()
	body := []byte("the same bytes")
	if err := os.WriteFile(filepath.Join(dst, "IMG_4.JPG"), body, 0o644); err != nil {
		t.Fatal(err)
	}
	dl := filepath.Join(t.TempDir(), "download")
	if err := os.WriteFile(dl, body, 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := os.OpenFile(dl, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	sum := sha1.Sum(body)
	match, err := sameContent(osfs.DirFS(dst), ".", []string{"IMG_4.JPG"}, r, base64.StdEncoding.EncodeToString(sum[:]))
	if err != nil || match != "IMG_4.JPG" {
		t.Fatalf("sameContent = %q, %v; want IMG_4.JPG without reading the download", match, err)
	}
	if _, err := sameContent(osfs.DirFS(dst), ".", []string{"IMG_4.JPG"}, r, ""); err == nil {
		t.Fatal("without a checksum sameContent must read the download (and fail here)")
	}
}
