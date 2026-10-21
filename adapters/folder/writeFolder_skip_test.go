package folder

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/base64"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/exif/sidecars/jsonsidecar"
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
	t.Setenv("IMMICHGO_TEMPDIR", t.TempDir())
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
	t.Setenv("IMMICHGO_TEMPDIR", t.TempDir())
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
	t.Setenv("IMMICHGO_TEMPDIR", t.TempDir())
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

// sha1Base64 is Immich's checksum of body: its SHA-1, base64.
func sha1Base64(body string) string {
	sum := sha1.Sum([]byte(body))
	return base64.StdEncoding.EncodeToString(sum[:])
}

// newImmichAsset is an asset as archive from-immich gives it: the server's checksum is on the asset
// and in the metadata its sidecar is written from.
func newImmichAsset(src *countingFS, file, base string, size int, checksum string) *assets.Asset {
	a := newAsset(src, file, base, size, "")
	a.Checksum = checksum
	a.FromApplication.Checksum = checksum
	return a
}

func writeSources(t *testing.T, dir string, bodies map[string]string) {
	t.Helper()
	for name, body := range bodies {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// writeArchived puts name in root as an earlier run left it: the file, and its .JSON sidecar with
// the given checksum ("" for a sidecar written by a release that had none).
func writeArchived(t *testing.T, root, name, body, checksum string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := jsonsidecar.Write(&assets.Metadata{FileName: path.Base(name), Checksum: checksum}, &buf); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p+".JSON", buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

// writeAll writes the assets with a new writer, as one archive run does, and closes them.
func writeAll(t *testing.T, dst fs.FS, as ...*assets.Asset) {
	t.Helper()
	w, err := NewLocalAssetWriter(dst, ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range as {
		err := w.WriteAsset(context.Background(), a)
		_ = a.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
}

// sidecarChecksum is the checksum in the .JSON sidecar of name.
func sidecarChecksum(t *testing.T, root, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)+".JSON"))
	if err != nil {
		t.Fatal(err)
	}
	var md assets.Metadata
	if err := jsonsidecar.Read(bytes.NewReader(b), &md); err != nil {
		t.Fatal(err)
	}
	return md.Checksum
}

// ageFiles sets the modification time of every file under root an hour back, so that a rewrite
// shows, and returns those times.
func ageFiles(t *testing.T, root string) map[string]time.Time {
	t.Helper()
	old := time.Now().Add(-time.Hour)
	stamps := map[string]time.Time{}
	for _, f := range listFiles(t, root) {
		p := filepath.Join(root, f)
		if err := os.Chtimes(p, old, old); err != nil {
			t.Fatal(err)
		}
		fi, err := os.Stat(p)
		if err != nil {
			t.Fatal(err)
		}
		stamps[f] = fi.ModTime()
	}
	return stamps
}

// checkUnchanged fails when a file of stamps, other than those in rewritten, was written or removed.
func checkUnchanged(t *testing.T, root string, stamps map[string]time.Time, rewritten ...string) {
	t.Helper()
	for f, stamp := range stamps {
		fi, err := os.Stat(filepath.Join(root, f))
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		if changed := !fi.ModTime().Equal(stamp); changed != slices.Contains(rewritten, f) {
			t.Fatalf("%s rewritten: %v, want %v", f, changed, !changed)
		}
	}
}

// archiveFS is an archive folder whose reads are counted, .JSON sidecars apart: any other Open is
// a read of an archived original.
type archiveFS struct {
	fs.FS
	reads map[string]int
}

func newArchiveFS(dir string) *archiveFS {
	return &archiveFS{FS: osfs.DirFS(dir), reads: map[string]int{}}
}

func (c *archiveFS) Open(name string) (fs.File, error) {
	if path.Ext(name) != ".JSON" {
		c.reads[name]++
	}
	return c.FS.Open(name)
}

func (c *archiveFS) Stat(name string) (fs.FileInfo, error) { return fs.Stat(c.FS, name) }

func (c *archiveFS) OpenFile(name string, flag int, perm fs.FileMode) (fshelper.WFile, error) {
	return fshelper.OpenFile(c.FS, name, flag, perm)
}

func (c *archiveFS) Mkdir(name string, perm fs.FileMode) error {
	return fshelper.Mkdir(c.FS, name, perm)
}

func (c *archiveFS) Remove(name string) error { return fshelper.Remove(c.FS, name) }

func (c *archiveFS) Rename(oldname, newname string) error {
	return fshelper.Rename(c.FS, oldname, newname)
}

// Two assets with the same name and the same size, in the same month, but other bytes: the
// second is not taken for the first and lands at ~1. Each sidecar carries its own asset's
// checksum, and a rerun finds each asset under its own name.
func TestSameNameSameSizeDifferentChecksum(t *testing.T) {
	t.Setenv("IMMICHGO_TEMPDIR", t.TempDir())
	bodies := map[string]string{"a": "the first asset", "b": "the other asset"}
	if len(bodies["a"]) != len(bodies["b"]) {
		t.Fatal("a and b must have the same size")
	}
	srcDir, dst := t.TempDir(), t.TempDir()
	writeSources(t, srcDir, bodies)
	src := &countingFS{FS: os.DirFS(srcDir), opens: map[string]int{}}
	run := func() {
		writeAll(t, osfs.DirFS(dst),
			newImmichAsset(src, "a", "IMG_1.jpg", len(bodies["a"]), sha1Base64(bodies["a"])),
			newImmichAsset(src, "b", "IMG_1.jpg", len(bodies["b"]), sha1Base64(bodies["b"])),
		)
	}

	run()
	want := []string{"2021/2021-07/IMG_1.jpg", "2021/2021-07/IMG_1.jpg.JSON", "2021/2021-07/IMG_1~1.jpg", "2021/2021-07/IMG_1~1.jpg.JSON"}
	if got := listFiles(t, dst); !equal(got, want) {
		t.Fatalf("wrote %v, want %v", got, want)
	}
	for name, file := range map[string]string{"2021/2021-07/IMG_1.jpg": "a", "2021/2021-07/IMG_1~1.jpg": "b"} {
		if b, _ := os.ReadFile(filepath.Join(dst, name)); string(b) != bodies[file] {
			t.Fatalf("%s holds %q, want %q", name, b, bodies[file])
		}
		if got := sidecarChecksum(t, dst, name); got != sha1Base64(bodies[file]) {
			t.Fatalf("%s sidecar checksum %q, want %q", name, got, sha1Base64(bodies[file]))
		}
	}

	run()
	if src.opens["a"] != 1 || src.opens["b"] != 1 {
		t.Fatalf("the rerun downloaded again: opens %v", src.opens)
	}
	if got := listFiles(t, dst); !equal(got, want) {
		t.Fatalf("the rerun left %v, want %v", got, want)
	}
}

// A rerun over an archive whose sidecars carry the checksums downloads nothing, reads no archived
// original and writes nothing. Asset c's size is not known (0): its checksum is enough.
func TestRerunWithChecksumOpensNothing(t *testing.T) {
	t.Setenv("IMMICHGO_TEMPDIR", t.TempDir())
	bodies := map[string]string{"a": "the first asset", "b": "the other asset", "c": "size not known yet"}
	srcDir, dst := t.TempDir(), t.TempDir()
	writeSources(t, srcDir, bodies)
	src := &countingFS{FS: os.DirFS(srcDir), opens: map[string]int{}}
	run := func(archive fs.FS) {
		writeAll(t, archive,
			newImmichAsset(src, "a", "IMG_1.jpg", len(bodies["a"]), sha1Base64(bodies["a"])),
			newImmichAsset(src, "b", "IMG_1.jpg", len(bodies["b"]), sha1Base64(bodies["b"])),
			newImmichAsset(src, "c", "IMG_2.jpg", 0, sha1Base64(bodies["c"])),
		)
	}

	run(osfs.DirFS(dst))
	want := []string{
		"2021/2021-07/IMG_1.jpg", "2021/2021-07/IMG_1.jpg.JSON", "2021/2021-07/IMG_1~1.jpg", "2021/2021-07/IMG_1~1.jpg.JSON",
		"2021/2021-07/IMG_2.jpg", "2021/2021-07/IMG_2.jpg.JSON",
	}
	if got := listFiles(t, dst); !equal(got, want) {
		t.Fatalf("run 1 wrote %v, want %v", got, want)
	}
	stamps := ageFiles(t, dst)

	archive := newArchiveFS(dst)
	run(archive)
	if src.opens["a"] != 1 || src.opens["b"] != 1 || src.opens["c"] != 1 {
		t.Fatalf("run 2 downloaded again: opens %v", src.opens)
	}
	if len(archive.reads) != 0 {
		t.Fatalf("run 2 read archived originals: %v", archive.reads)
	}
	if got := listFiles(t, dst); !equal(got, want) {
		t.Fatalf("run 2 left %v, want %v", got, want)
	}
	checkUnchanged(t, dst, stamps)
}

// An archive written by an older release has sidecars without a checksum. A rerun reads each
// archived file once instead of downloading the asset: the same bytes are the asset (no download,
// no ~1) and its sidecar gets the checksum; other bytes of the same size are another asset. Here
// the older release had kept x under IMG_3.jpg and skipped b, which has the same name and size;
// b comes first, so it is compared with x's bytes, and lands at ~1. The next run reads nothing.
func TestOldSidecarWithoutChecksum(t *testing.T) {
	t.Setenv("IMMICHGO_TEMPDIR", t.TempDir())
	bodies := map[string]string{
		"a": "the first asset", "c": "size not known yet",
		"b": "the other asset", "x": "same size as b!",
	}
	if len(bodies["x"]) != len(bodies["b"]) {
		t.Fatal("x and b must have the same size")
	}
	srcDir, dst := t.TempDir(), t.TempDir()
	writeSources(t, srcDir, bodies)
	src := &countingFS{FS: os.DirFS(srcDir), opens: map[string]int{}}
	for name, file := range map[string]string{"IMG_1.jpg": "a", "IMG_2.jpg": "c", "IMG_3.jpg": "x"} {
		writeArchived(t, dst, "2021/2021-07/"+name, bodies[file], "")
	}
	stamps := ageFiles(t, dst)
	run := func(archive fs.FS) {
		writeAll(t, archive,
			newImmichAsset(src, "a", "IMG_1.jpg", len(bodies["a"]), sha1Base64(bodies["a"])),
			newImmichAsset(src, "c", "IMG_2.jpg", 0, sha1Base64(bodies["c"])),
			newImmichAsset(src, "b", "IMG_3.jpg", len(bodies["b"]), sha1Base64(bodies["b"])),
			newImmichAsset(src, "x", "IMG_3.jpg", len(bodies["x"]), sha1Base64(bodies["x"])),
		)
	}

	run(osfs.DirFS(dst))
	if src.opens["a"] != 0 || src.opens["c"] != 0 || src.opens["x"] != 0 || src.opens["b"] != 1 {
		t.Fatalf("opens %v, want b only, once", src.opens)
	}
	want := []string{
		"2021/2021-07/IMG_1.jpg", "2021/2021-07/IMG_1.jpg.JSON", "2021/2021-07/IMG_2.jpg", "2021/2021-07/IMG_2.jpg.JSON",
		"2021/2021-07/IMG_3.jpg", "2021/2021-07/IMG_3.jpg.JSON", "2021/2021-07/IMG_3~1.jpg", "2021/2021-07/IMG_3~1.jpg.JSON",
	}
	if got := listFiles(t, dst); !equal(got, want) {
		t.Fatalf("wrote %v, want %v", got, want)
	}
	for name, file := range map[string]string{"IMG_1.jpg": "a", "IMG_2.jpg": "c", "IMG_3.jpg": "x", "IMG_3~1.jpg": "b"} {
		name = "2021/2021-07/" + name
		if b, _ := os.ReadFile(filepath.Join(dst, name)); string(b) != bodies[file] {
			t.Fatalf("%s holds %q, want %q", name, b, bodies[file])
		}
		if got := sidecarChecksum(t, dst, name); got != sha1Base64(bodies[file]) {
			t.Fatalf("%s sidecar checksum %q, want %q", name, got, sha1Base64(bodies[file]))
		}
	}
	checkUnchanged(t, dst, stamps, "2021/2021-07/IMG_1.jpg.JSON", "2021/2021-07/IMG_2.jpg.JSON", "2021/2021-07/IMG_3.jpg.JSON")

	stamps = ageFiles(t, dst)
	archive := newArchiveFS(dst)
	run(archive)
	if len(archive.reads) != 0 {
		t.Fatalf("the next run read archived originals: %v", archive.reads)
	}
	if src.opens["a"] != 0 || src.opens["c"] != 0 || src.opens["x"] != 0 || src.opens["b"] != 1 {
		t.Fatalf("the next run downloaded: opens %v", src.opens)
	}
	checkUnchanged(t, dst, stamps)
}

// Other content under the asset's name, whose sidecar has another checksum (and here another
// size), is another file: the asset goes to ~1, and that file and its sidecar are left as they
// are. The sidecar decides; the file is not read.
func TestChangedContentGoesToNextName(t *testing.T) {
	t.Setenv("IMMICHGO_TEMPDIR", t.TempDir())
	body := "the asset, longer than the file there"
	srcDir, dst := t.TempDir(), t.TempDir()
	writeSources(t, srcDir, map[string]string{"n": body})
	src := &countingFS{FS: os.DirFS(srcDir), opens: map[string]int{}}
	writeArchived(t, dst, "2021/2021-07/IMG_1.jpg", "other content", sha1Base64("other content"))
	stamps := ageFiles(t, dst)

	archive := newArchiveFS(dst)
	writeAll(t, archive, newImmichAsset(src, "n", "IMG_1.jpg", len(body), sha1Base64(body)))
	if src.opens["n"] != 1 {
		t.Fatalf("opens %v, want n once", src.opens)
	}
	if len(archive.reads) != 0 {
		t.Fatalf("read archived originals: %v", archive.reads)
	}
	want := []string{"2021/2021-07/IMG_1.jpg", "2021/2021-07/IMG_1.jpg.JSON", "2021/2021-07/IMG_1~1.jpg", "2021/2021-07/IMG_1~1.jpg.JSON"}
	if got := listFiles(t, dst); !equal(got, want) {
		t.Fatalf("wrote %v, want %v", got, want)
	}
	if b, _ := os.ReadFile(filepath.Join(dst, "2021/2021-07/IMG_1~1.jpg")); string(b) != body {
		t.Fatalf("IMG_1~1.jpg holds %q", b)
	}
	if got := sidecarChecksum(t, dst, "2021/2021-07/IMG_1~1.jpg"); got != sha1Base64(body) {
		t.Fatalf("IMG_1~1.jpg sidecar checksum %q, want %q", got, sha1Base64(body))
	}
	if b, _ := os.ReadFile(filepath.Join(dst, "2021/2021-07/IMG_1.jpg")); string(b) != "other content" {
		t.Fatalf("IMG_1.jpg holds %q", b)
	}
	checkUnchanged(t, dst, stamps)
}

// A download whose SHA-1 is not the asset's checksum is refused with an error, and nothing is
// left: no original, no .partial, no sidecar. A download with the right SHA-1 is written. The
// SHA-1 is the cache reader's for a streamed source; a local file opened in place is hashed.
func TestDownloadChecksumMismatch(t *testing.T) {
	for _, plain := range []bool{false, true} {
		t.Run(map[bool]string{false: "streamed", true: "local"}[plain], func(t *testing.T) {
			t.Setenv("IMMICHGO_TEMPDIR", t.TempDir())
			body := "bytes that are not the asset"
			srcDir, dst := t.TempDir(), t.TempDir()
			writeSources(t, srcDir, map[string]string{"m": body})
			src := &countingFS{FS: os.DirFS(srcDir), opens: map[string]int{}, plain: plain}
			w, err := NewLocalAssetWriter(osfs.DirFS(dst), ".")
			if err != nil {
				t.Fatal(err)
			}

			a := newImmichAsset(src, "m", "IMG_1.jpg", len(body), sha1Base64("the asset's bytes"))
			err = w.WriteAsset(context.Background(), a)
			_ = a.Close()
			if err == nil || !strings.Contains(err.Error(), "SHA-1") {
				t.Fatalf("WriteAsset: %v; want an error about the SHA-1", err)
			}
			if files := listFiles(t, dst); len(files) != 0 {
				t.Fatalf("files %q left; want none (no original, no .partial, no sidecar)", files)
			}

			a = newImmichAsset(src, "m", "IMG_2.jpg", len(body), sha1Base64(body))
			err = w.WriteAsset(context.Background(), a)
			_ = a.Close()
			want := []string{"2021/2021-07/IMG_2.jpg", "2021/2021-07/IMG_2.jpg.JSON"}
			if got := listFiles(t, dst); err != nil || !equal(got, want) {
				t.Fatalf("WriteAsset: %v, files %v; want %v", err, got, want)
			}
		})
	}
}
