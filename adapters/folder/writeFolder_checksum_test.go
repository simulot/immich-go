package folder

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/exif/sidecars/jsonsidecar"
	"github.com/simulot/immich-go/internal/fshelper/osfs"
)

// asExternal makes a an asset of an Immich external library as archive from-immich gives it: its
// Immich checksum hashes its path, and is on the asset and, as the path checksum, in the metadata
// its sidecar is written from.
func asExternal(a *assets.Asset, pathChecksum string) *assets.Asset {
	a.Checksum = pathChecksum
	a.FromApplication.PathChecksum = pathChecksum
	return a
}

// sidecarOf is the metadata in the .JSON sidecar of name.
func sidecarOf(t *testing.T, root, name string) assets.Metadata {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)+".JSON"))
	if err != nil {
		t.Fatal(err)
	}
	var md assets.Metadata
	if err := jsonsidecar.Read(bytes.NewReader(b), &md); err != nil {
		t.Fatal(err)
	}
	return md
}

// checkHolds fails when a file of root does not hold the given bytes.
func checkHolds(t *testing.T, root string, bodies map[string]string) {
	t.Helper()
	for name, body := range bodies {
		if b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name))); err != nil || string(b) != body {
			t.Fatalf("%s holds %q (%v), want %q", name, b, err, body)
		}
	}
}

// simulot's reproduction on #1478: an archive from-folder source of 17 bytes, and another file
// of 17 bytes already under the name, with a sidecar from an older release (no checksum) or none.
// The size says nothing: the asset goes to ~1, and that file and its sidecar are left as they are.
func TestFromFolderSameNameSameSizeOtherFile(t *testing.T) {
	body, other := "the asset's bytes", "another 17 bytes!"
	if len(body) != 17 || len(other) != 17 {
		t.Fatal("both files must have 17 bytes")
	}
	for _, tc := range []struct {
		name    string
		sidecar bool
	}{{"older sidecar", true}, {"no sidecar", false}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("IMMICHGO_TEMPDIR", t.TempDir())
			srcDir, dst := t.TempDir(), t.TempDir()
			writeSources(t, srcDir, map[string]string{"s": body})
			src := &countingFS{FS: os.DirFS(srcDir), opens: map[string]int{}, plain: true}
			want := []string{"2021/2021-07/IMG_1.jpg"}
			if tc.sidecar {
				writeArchived(t, dst, "2021/2021-07/IMG_1.jpg", other, "")
				want = append(want, "2021/2021-07/IMG_1.jpg.JSON")
			} else {
				if err := os.MkdirAll(filepath.Join(dst, "2021", "2021-07"), 0o755); err != nil {
					t.Fatal(err)
				}
				writeSources(t, filepath.Join(dst, "2021", "2021-07"), map[string]string{"IMG_1.jpg": other})
			}
			stamps := ageFiles(t, dst)

			writeAll(t, osfs.DirFS(dst), newAsset(src, "s", "IMG_1.jpg", len(body), "a description"))
			want = append(want, "2021/2021-07/IMG_1~1.jpg", "2021/2021-07/IMG_1~1.jpg.JSON")
			if got := listFiles(t, dst); !slices.Equal(got, want) {
				t.Fatalf("files %v, want %v", got, want)
			}
			checkHolds(t, dst, map[string]string{"2021/2021-07/IMG_1.jpg": other, "2021/2021-07/IMG_1~1.jpg": body})
			checkUnchanged(t, dst, stamps)
			if md := sidecarOf(t, dst, "2021/2021-07/IMG_1~1.jpg"); md.Checksum != sha1Base64(body) || md.Description != "a description" {
				t.Fatalf("IMG_1~1.jpg sidecar: checksum %q, description %q; want %q, %q", md.Checksum, md.Description, sha1Base64(body), "a description")
			}
		})
	}
}

// archive from-folder writes in each sidecar the SHA-1 of the archived bytes. A rerun reads each
// source (a local read) to compare its SHA-1 with the sidecar's, reads no archived original and
// writes nothing. a and b have the same name and the same size: b is at ~1.
func TestFromFolderRerunWritesNothing(t *testing.T) {
	t.Setenv("IMMICHGO_TEMPDIR", t.TempDir())
	bodies := map[string]string{"a": "the first asset", "b": "the other asset", "c": "a third one"}
	if len(bodies["a"]) != len(bodies["b"]) {
		t.Fatal("a and b must have the same size")
	}
	srcDir, dst := t.TempDir(), t.TempDir()
	writeSources(t, srcDir, bodies)
	src := &countingFS{FS: os.DirFS(srcDir), opens: map[string]int{}, plain: true}
	run := func(archive fs.FS) {
		writeAll(t, archive,
			newAsset(src, "a", "IMG_1.jpg", len(bodies["a"]), ""),
			newAsset(src, "b", "IMG_1.jpg", len(bodies["b"]), ""),
			newAsset(src, "c", "IMG_2.jpg", len(bodies["c"]), ""),
		)
	}

	run(osfs.DirFS(dst))
	want := []string{
		"2021/2021-07/IMG_1.jpg", "2021/2021-07/IMG_1.jpg.JSON", "2021/2021-07/IMG_1~1.jpg", "2021/2021-07/IMG_1~1.jpg.JSON",
		"2021/2021-07/IMG_2.jpg", "2021/2021-07/IMG_2.jpg.JSON",
	}
	if got := listFiles(t, dst); !slices.Equal(got, want) {
		t.Fatalf("run 1 wrote %v, want %v", got, want)
	}
	for name, file := range map[string]string{"2021/2021-07/IMG_1.jpg": "a", "2021/2021-07/IMG_1~1.jpg": "b", "2021/2021-07/IMG_2.jpg": "c"} {
		checkHolds(t, dst, map[string]string{name: bodies[file]})
		if got := sidecarChecksum(t, dst, name); got != sha1Base64(bodies[file]) {
			t.Fatalf("%s sidecar checksum %q, want %q", name, got, sha1Base64(bodies[file]))
		}
	}
	stamps := ageFiles(t, dst)

	archive := newArchiveFS(dst)
	run(archive)
	if len(archive.reads) != 0 {
		t.Fatalf("run 2 read archived originals: %v", archive.reads)
	}
	if src.opens["a"] != 2 || src.opens["b"] != 2 || src.opens["c"] != 2 {
		t.Fatalf("opens %v, want each source read once per run", src.opens)
	}
	if got := listFiles(t, dst); !slices.Equal(got, want) {
		t.Fatalf("run 2 left %v, want %v", got, want)
	}
	checkUnchanged(t, dst, stamps)
}

// An external-library asset's sidecar has the SHA-1 of the archived bytes as its checksum, and its
// Immich checksum (a hash of its path) as its path checksum. A rerun recognizes the file by the
// path checksum: nothing downloaded, nothing read, nothing written. x and y have the same name and
// the same size: y is at ~1.
func TestExternalLibraryRerunNoDownload(t *testing.T) {
	t.Setenv("IMMICHGO_TEMPDIR", t.TempDir())
	bodies := map[string]string{"x": "the first asset", "y": "the other asset"}
	paths := map[string]string{"x": sha1Base64("path:/lib/x/IMG_1.jpg"), "y": sha1Base64("path:/lib/y/IMG_1.jpg")}
	srcDir, dst := t.TempDir(), t.TempDir()
	writeSources(t, srcDir, bodies)
	src := &countingFS{FS: os.DirFS(srcDir), opens: map[string]int{}}
	run := func(archive fs.FS) {
		writeAll(t, archive,
			asExternal(newAsset(src, "x", "IMG_1.jpg", len(bodies["x"]), ""), paths["x"]),
			asExternal(newAsset(src, "y", "IMG_1.jpg", len(bodies["y"]), ""), paths["y"]),
		)
	}

	run(osfs.DirFS(dst))
	want := []string{"2021/2021-07/IMG_1.jpg", "2021/2021-07/IMG_1.jpg.JSON", "2021/2021-07/IMG_1~1.jpg", "2021/2021-07/IMG_1~1.jpg.JSON"}
	if got := listFiles(t, dst); !slices.Equal(got, want) {
		t.Fatalf("run 1 wrote %v, want %v", got, want)
	}
	for name, file := range map[string]string{"2021/2021-07/IMG_1.jpg": "x", "2021/2021-07/IMG_1~1.jpg": "y"} {
		checkHolds(t, dst, map[string]string{name: bodies[file]})
		if md := sidecarOf(t, dst, name); md.Checksum != sha1Base64(bodies[file]) || md.PathChecksum != paths[file] {
			t.Fatalf("%s sidecar: checksum %q, path checksum %q; want %q, %q", name, md.Checksum, md.PathChecksum, sha1Base64(bodies[file]), paths[file])
		}
	}
	stamps := ageFiles(t, dst)

	archive := newArchiveFS(dst)
	run(archive)
	if src.opens["x"] != 1 || src.opens["y"] != 1 {
		t.Fatalf("run 2 downloaded: opens %v", src.opens)
	}
	if len(archive.reads) != 0 {
		t.Fatalf("run 2 read archived originals: %v", archive.reads)
	}
	if got := listFiles(t, dst); !slices.Equal(got, want) {
		t.Fatalf("run 2 left %v, want %v", got, want)
	}
	checkUnchanged(t, dst, stamps)
}

// An external-library file archived by an older release has a sidecar without a checksum. A rerun
// downloads the asset once and compares its SHA-1 with the archived file's, read once: here y,
// another asset of the same name and size, comes first, does not match, and is written at ~1 from
// that same download; x matches, is not written again, and its sidecar gets the checksum and the
// path checksum. The next run downloads and reads nothing.
func TestExternalLibraryOldSidecarDownloadsOnce(t *testing.T) {
	t.Setenv("IMMICHGO_TEMPDIR", t.TempDir())
	bodies := map[string]string{"x": "the first asset", "y": "the other asset"}
	paths := map[string]string{"x": sha1Base64("path:/lib/x/IMG_1.jpg"), "y": sha1Base64("path:/lib/y/IMG_1.jpg")}
	srcDir, dst := t.TempDir(), t.TempDir()
	writeSources(t, srcDir, bodies)
	src := &countingFS{FS: os.DirFS(srcDir), opens: map[string]int{}}
	writeArchived(t, dst, "2021/2021-07/IMG_1.jpg", bodies["x"], "")
	stamps := ageFiles(t, dst)
	run := func(archive fs.FS) {
		writeAll(t, archive,
			asExternal(newAsset(src, "y", "IMG_1.jpg", len(bodies["y"]), ""), paths["y"]),
			asExternal(newAsset(src, "x", "IMG_1.jpg", len(bodies["x"]), ""), paths["x"]),
		)
	}

	archive := newArchiveFS(dst)
	run(archive)
	if src.opens["x"] != 1 || src.opens["y"] != 1 {
		t.Fatalf("opens %v, want each asset downloaded once", src.opens)
	}
	if archive.reads["2021/2021-07/IMG_1.jpg"] != 1 || len(archive.reads) != 1 {
		t.Fatalf("archived originals read %v, want IMG_1.jpg once", archive.reads)
	}
	want := []string{"2021/2021-07/IMG_1.jpg", "2021/2021-07/IMG_1.jpg.JSON", "2021/2021-07/IMG_1~1.jpg", "2021/2021-07/IMG_1~1.jpg.JSON"}
	if got := listFiles(t, dst); !slices.Equal(got, want) {
		t.Fatalf("wrote %v, want %v", got, want)
	}
	checkHolds(t, dst, map[string]string{"2021/2021-07/IMG_1.jpg": bodies["x"], "2021/2021-07/IMG_1~1.jpg": bodies["y"]})
	for name, file := range map[string]string{"2021/2021-07/IMG_1.jpg": "x", "2021/2021-07/IMG_1~1.jpg": "y"} {
		if md := sidecarOf(t, dst, name); md.Checksum != sha1Base64(bodies[file]) || md.PathChecksum != paths[file] {
			t.Fatalf("%s sidecar: checksum %q, path checksum %q; want %q, %q", name, md.Checksum, md.PathChecksum, sha1Base64(bodies[file]), paths[file])
		}
	}
	checkUnchanged(t, dst, stamps, "2021/2021-07/IMG_1.jpg.JSON")

	stamps = ageFiles(t, dst)
	archive = newArchiveFS(dst)
	run(archive)
	if src.opens["x"] != 1 || src.opens["y"] != 1 {
		t.Fatalf("the next run downloaded: opens %v", src.opens)
	}
	if len(archive.reads) != 0 {
		t.Fatalf("the next run read archived originals: %v", archive.reads)
	}
	checkUnchanged(t, dst, stamps)
}

// A from-folder file archived by an older release has a sidecar without a checksum. A rerun reads
// it once and compares its SHA-1 with the source's: here b, another source of the same name and
// size, comes first, does not match, and is written at ~1; a matches, is not written again, and
// its sidecar gets the checksum. The next run reads no archived original.
func TestOldSidecarFromFolderHashedOnce(t *testing.T) {
	t.Setenv("IMMICHGO_TEMPDIR", t.TempDir())
	bodies := map[string]string{"a": "the first asset", "b": "the other asset"}
	srcDir, dst := t.TempDir(), t.TempDir()
	writeSources(t, srcDir, bodies)
	src := &countingFS{FS: os.DirFS(srcDir), opens: map[string]int{}, plain: true}
	writeArchived(t, dst, "2021/2021-07/IMG_1.jpg", bodies["a"], "")
	stamps := ageFiles(t, dst)
	run := func(archive fs.FS) {
		writeAll(t, archive,
			newAsset(src, "b", "IMG_1.jpg", len(bodies["b"]), ""),
			newAsset(src, "a", "IMG_1.jpg", len(bodies["a"]), ""),
		)
	}

	archive := newArchiveFS(dst)
	run(archive)
	if archive.reads["2021/2021-07/IMG_1.jpg"] != 1 || len(archive.reads) != 1 {
		t.Fatalf("archived originals read %v, want IMG_1.jpg once", archive.reads)
	}
	want := []string{"2021/2021-07/IMG_1.jpg", "2021/2021-07/IMG_1.jpg.JSON", "2021/2021-07/IMG_1~1.jpg", "2021/2021-07/IMG_1~1.jpg.JSON"}
	if got := listFiles(t, dst); !slices.Equal(got, want) {
		t.Fatalf("wrote %v, want %v", got, want)
	}
	checkHolds(t, dst, map[string]string{"2021/2021-07/IMG_1.jpg": bodies["a"], "2021/2021-07/IMG_1~1.jpg": bodies["b"]})
	for name, file := range map[string]string{"2021/2021-07/IMG_1.jpg": "a", "2021/2021-07/IMG_1~1.jpg": "b"} {
		if got := sidecarChecksum(t, dst, name); got != sha1Base64(bodies[file]) {
			t.Fatalf("%s sidecar checksum %q, want %q", name, got, sha1Base64(bodies[file]))
		}
	}
	checkUnchanged(t, dst, stamps, "2021/2021-07/IMG_1.jpg.JSON")

	stamps = ageFiles(t, dst)
	archive = newArchiveFS(dst)
	run(archive)
	if len(archive.reads) != 0 {
		t.Fatalf("the next run read archived originals: %v", archive.reads)
	}
	checkUnchanged(t, dst, stamps)
}

// A timeline asset and an external-library asset with the same name and the same bytes are two
// assets: each keeps its own file, whichever comes first. A sidecar with a path checksum is that
// external-library asset's, and a sidecar without one is not an external-library asset's.
func TestTimelineAndExternalLibrarySameBytes(t *testing.T) {
	body := "the same bytes"
	pathSum := sha1Base64("path:/lib/IMG_1.jpg")
	for _, externalFirst := range []bool{false, true} {
		t.Run(map[bool]string{false: "timeline first", true: "external first"}[externalFirst], func(t *testing.T) {
			t.Setenv("IMMICHGO_TEMPDIR", t.TempDir())
			srcDir, dst := t.TempDir(), t.TempDir()
			writeSources(t, srcDir, map[string]string{"t": body, "e": body})
			src := &countingFS{FS: os.DirFS(srcDir), opens: map[string]int{}}
			run := func(archive fs.FS) {
				as := []*assets.Asset{
					newImmichAsset(src, "t", "IMG_1.jpg", len(body), sha1Base64(body)),
					asExternal(newAsset(src, "e", "IMG_1.jpg", len(body), ""), pathSum),
				}
				if externalFirst {
					as[0], as[1] = as[1], as[0]
				}
				writeAll(t, archive, as...)
			}

			run(osfs.DirFS(dst))
			want := []string{"2021/2021-07/IMG_1.jpg", "2021/2021-07/IMG_1.jpg.JSON", "2021/2021-07/IMG_1~1.jpg", "2021/2021-07/IMG_1~1.jpg.JSON"}
			if got := listFiles(t, dst); !slices.Equal(got, want) {
				t.Fatalf("run 1 wrote %v, want %v", got, want)
			}
			external, timeline := "2021/2021-07/IMG_1~1.jpg", "2021/2021-07/IMG_1.jpg"
			if externalFirst {
				external, timeline = timeline, external
			}
			if md := sidecarOf(t, dst, external); md.Checksum != sha1Base64(body) || md.PathChecksum != pathSum {
				t.Fatalf("%s sidecar: checksum %q, path checksum %q; want %q, %q", external, md.Checksum, md.PathChecksum, sha1Base64(body), pathSum)
			}
			if md := sidecarOf(t, dst, timeline); md.Checksum != sha1Base64(body) || md.PathChecksum != "" {
				t.Fatalf("%s sidecar: checksum %q, path checksum %q; want %q, none", timeline, md.Checksum, md.PathChecksum, sha1Base64(body))
			}
			stamps := ageFiles(t, dst)

			archive := newArchiveFS(dst)
			run(archive)
			if src.opens["t"] != 1 || src.opens["e"] != 1 {
				t.Fatalf("run 2 downloaded: opens %v", src.opens)
			}
			if len(archive.reads) != 0 {
				t.Fatalf("run 2 read archived originals: %v", archive.reads)
			}
			checkUnchanged(t, dst, stamps)
		})
	}
}
