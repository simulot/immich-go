package folder

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math/rand/v2"
	"os"
	"path"
	"strconv"
	"sync"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/exif/sidecars/jsonsidecar"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/simulot/immich-go/internal/fshelper/debugfiles"
	"github.com/simulot/immich-go/internal/fshelper/osfs"
)

// type minimalFSWriter interface {
// 	fs.FS
// 	fshelper.FSCanWrite
// }

// LocalAssetWriter writes assets into the folder WriteToFS, one folder per month. WriteAsset can be
// called from several goroutines for assets of different names. Two assets that can get the same
// name must be written one after the other: each takes the first name that is free.
type LocalAssetWriter struct {
	WriteToFS fs.FS

	mu         sync.Mutex          // guards createdDir and hashed
	createdDir map[string]struct{} // the folders made in this run
	hashed     map[string]string   // SHA-1, base64, of the archived files read in this run, by path
	// hashed grows by one entry per archived file read (see isArchived): at most one per file.
}

func NewLocalAssetWriter(fsys fs.FS) (*LocalAssetWriter, error) {
	if _, ok := fsys.(fshelper.FSCanWrite); !ok {
		return nil, errors.New("FS does not support writing")
	}
	return &LocalAssetWriter{
		WriteToFS:  fsys,
		createdDir: make(map[string]struct{}),
		hashed:     make(map[string]string),
	}, nil
}

func (w *LocalAssetWriter) WriteAsset(ctx context.Context, a *assets.Asset) error {
	base := a.Base
	dir := w.pathOfAsset(a)
	if err := w.makeDir(dir); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// What tells the asset apart from the files already in the archive, as its source gives it.
	// archive from-immich gives the asset's Immich checksum, on the asset and in the metadata its
	// sidecar is written from: for a timeline asset, the SHA-1 of its original, base64; for an
	// external-library asset, a hash of its path, kept as the path checksum. Other sources give
	// neither: only the SHA-1 of the asset's bytes tells it apart, and the asset is read for it.
	src := &source{a: a}
	defer src.Close()
	id := identity{size: int64(a.FileSize), src: src}
	if a.FromApplication != nil && a.Checksum != "" {
		switch a.Checksum {
		case a.FromApplication.Checksum:
			id.checksum = a.Checksum
		case a.FromApplication.PathChecksum:
			id.pathChecksum = a.Checksum
		}
	}

	// Choose the file name before the asset is opened: opening it is what downloads it. The name is
	// the first of base, radical~1.ext, radical~2.ext, ... that is free, or that already holds this
	// asset, written by an earlier run: then it is not written again and no ~N copy is made. A file
	// is this asset only when its content is proven to be, never on its name and size alone (see
	// isArchived). Other content under the same name is another file, and the asset goes to the
	// next free ~N: the archive never deletes or overwrites a file, and never rewrites the sidecar
	// of another file. An asset's checksum never changes: Immich edits are non-destructive,
	// /original returns the original bytes, and the checksum is the original's.
	//
	// archived is the SHA-1 of the archived file, base64, once it is known.
	ext := path.Ext(base)
	radical := base[:len(base)-len(ext)]
	name, archived := base, ""
	for index := 0; ; index++ {
		if index > 0 {
			name = fmt.Sprintf("%s~%d%s", radical, index, ext)
		}
		p := path.Join(dir, name)
		if fi, err := fs.Stat(w.WriteToFS, p); err == nil {
			if fi.Mode().IsRegular() {
				archived, err = w.isArchived(p, fi, id)
				if err != nil {
					return err
				}
				if archived != "" {
					break
				}
			}
			continue
		}
		if _, err := fs.Stat(w.WriteToFS, p+".XMP"); err == nil {
			continue
		}
		if _, err := fs.Stat(w.WriteToFS, p+".JSON"); err == nil {
			continue
		}
		break
	}

	if archived == "" {
		if err := src.open(); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		// Under a temporary name first: a download cut short never sits under the asset's name,
		// where the next run would take it for another asset. With the size known, the bytes must
		// be exactly that many, whatever the transport said; with a timeline asset's checksum,
		// their SHA-1 must be it: the download must be the asset, or nothing is written.
		var err error
		archived, err = writeAtomic(w.WriteToFS, path.Join(dir, name), src.r, id.size, id.checksum)
		if err != nil {
			return err
		}
	}

	// XMP? Sidecar file is set, copy it. Written only when its content changed, through a
	// temporary file, as the .JSON below.
	if a.FromSideCar != nil {
		scr, err := a.FromSideCar.File.Open()
		if err != nil {
			return err
		}
		debugfiles.TrackOpenFile(scr, a.FromSideCar.File.Name())
		defer scr.Close()
		defer debugfiles.TrackCloseFile(scr)
		xmp, err := io.ReadAll(scr)
		if err != nil {
			return err
		}
		xmpName := path.Join(dir, name+".XMP")
		old, err := fs.ReadFile(w.WriteToFS, xmpName)
		if err != nil || !bytes.Equal(old, xmp) {
			if _, err := writeAtomic(w.WriteToFS, xmpName, bytes.NewReader(xmp), 0, ""); err != nil {
				return err
			}
		}
	}

	// Having metadata from an Application or immich-go JSON? Written only when its content
	// changed, through a temporary file. It records what tells the asset apart in the archive:
	// the SHA-1 of the archived bytes, whatever the source, and an external-library asset's path
	// checksum.
	if a.FromApplication != nil {
		md := *a.FromApplication
		md.Checksum, md.PathChecksum = archived, id.pathChecksum
		var buf bytes.Buffer
		err := jsonsidecar.Write(&md, &buf)
		if err != nil {
			return err
		}
		jsonName := path.Join(dir, name+".JSON")
		old, err := fs.ReadFile(w.WriteToFS, jsonName)
		if err != nil || !bytes.Equal(old, buf.Bytes()) {
			_, err = writeAtomic(w.WriteToFS, jsonName, &buf, 0, "")
			return err
		}
	}
	return nil
}

// identity is what tells an asset apart from the files already in the archive.
type identity struct {
	size         int64   // the asset's size; 0 when not known
	checksum     string  // a timeline asset's Immich checksum: the SHA-1 of its original, base64
	pathChecksum string  // an external-library asset's Immich checksum: a hash of its path
	src          *source // the asset, read when the SHA-1 of its bytes is needed
}

// isArchived tells whether the regular file p (fi) is the asset, and returns then the SHA-1 of its
// bytes, base64; it returns "" when p is another file. A file is never taken for the asset on its
// name and size alone:
//
//   - With the asset's size known, a file of another size is not the asset, whatever its sidecar
//     says.
//   - A sidecar with a path checksum is that external-library asset's: no other asset is taken for
//     its file.
//   - A sidecar with a checksum (the SHA-1 of the archived file) decides, and p is not read: an
//     external-library asset is there when the sidecar's path checksum is its own, a timeline asset
//     when the checksum is its own, an asset of another source when the checksum is the SHA-1 of the
//     asset's bytes, read from the source.
//   - A sidecar without a checksum (written by an older release), or none: p is read, once per run,
//     and its SHA-1 compared with the timeline asset's checksum, or with the SHA-1 of the asset's
//     bytes, downloaded once for an external-library asset, read from the source for another. When
//     they match, the sidecar written next records them.
func (w *LocalAssetWriter) isArchived(p string, fi fs.FileInfo, id identity) (string, error) {
	if id.size > 0 && fi.Size() != id.size {
		return "", nil
	}
	var md assets.Metadata
	if b, err := fs.ReadFile(w.WriteToFS, p+".JSON"); err == nil {
		if jsonsidecar.Read(bytes.NewReader(b), &md) != nil {
			md = assets.Metadata{} // unreadable: as no sidecar
		}
	}
	if md.PathChecksum != "" && md.PathChecksum != id.pathChecksum {
		return "", nil
	}
	if md.Checksum != "" {
		var match bool
		switch {
		case id.pathChecksum != "":
			match = md.PathChecksum == id.pathChecksum
		case id.checksum != "":
			match = md.Checksum == id.checksum
		default:
			sum, err := id.src.sha1()
			if err != nil {
				return "", err
			}
			match = md.Checksum == sum
		}
		if !match {
			return "", nil
		}
		return md.Checksum, nil
	}
	got, err := w.sha1OfArchived(p)
	if err != nil {
		return "", err
	}
	want := id.checksum
	if want == "" {
		if want, err = id.src.sha1(); err != nil {
			return "", err
		}
	}
	if got != want {
		return "", nil
	}
	return got, nil
}

// makeDir makes the folder dir, once per run.
func (w *LocalAssetWriter) makeDir(dir string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if _, ok := w.createdDir[dir]; ok {
		return nil
	}
	if err := fshelper.MkdirAll(w.WriteToFS, dir, 0o755); err != nil {
		return err
	}
	w.createdDir[dir] = struct{}{}
	return nil
}

// sha1OfArchived returns the SHA-1 of the archived file p, base64. p is read once per run.
func (w *LocalAssetWriter) sha1OfArchived(p string) (string, error) {
	w.mu.Lock()
	sum, ok := w.hashed[p]
	w.mu.Unlock()
	if ok {
		return sum, nil
	}
	f, err := w.WriteToFS.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()
	b, err := sha1Of(f)
	if err != nil {
		return "", err
	}
	sum = base64.StdEncoding.EncodeToString(b)
	w.mu.Lock()
	w.hashed[p] = sum
	w.mu.Unlock()
	return sum, nil
}

// source is the asset being written. It is opened at most once, and only when needed: opening an
// Immich asset is what downloads it.
type source struct {
	a   *assets.Asset
	r   osfs.OSFS
	sum string // the SHA-1 of the asset's bytes, base64, once known
}

func (s *source) open() error {
	if s.r != nil {
		return nil
	}
	r, sum, err := s.a.OpenFile()
	if err != nil {
		return err
	}
	s.r, s.sum = r, sum
	return nil
}

// sha1 returns the SHA-1 of the asset's bytes, base64: the one the cache reader computed while it
// copied a streamed source (an Immich download), or else the bytes are read, and r is rewound.
func (s *source) sha1() (string, error) {
	if err := s.open(); err != nil {
		return "", err
	}
	if s.sum == "" {
		sum, err := sha1Of(s.r)
		if err != nil {
			return "", err
		}
		s.sum = base64.StdEncoding.EncodeToString(sum)
	}
	return s.sum, nil
}

func (s *source) Close() error {
	if s.r == nil {
		return nil
	}
	return s.r.Close()
}

func sha1Of(r io.Reader) ([]byte, error) {
	h := sha1.New()
	if _, err := io.Copy(h, r); err != nil {
		return nil, err
	}
	if s, ok := r.(io.Seeker); ok {
		if _, err := s.Seek(0, io.SeekStart); err != nil {
			return nil, err
		}
	}
	return h.Sum(nil), nil
}

// writeAtomic writes r to a new ".<name>.<random>.partial" file beside name, syncs it, then
// renames it to name, and returns the SHA-1 of the bytes it wrote, base64. Each call writes its
// own .partial file: two runs that write the same name never write into the same file. With
// size > 0, r must give exactly size bytes, and with checksum set, their SHA-1 must be checksum.
// On any error, the rename's included, the .partial file is removed and name is not written.
func writeAtomic(fsys fs.FS, name string, r io.Reader, size int64, checksum string) (string, error) {
	f, tmp, err := createPartial(fsys, name)
	if err != nil {
		return "", err
	}
	h := sha1.New()
	n, err := io.Copy(f, io.TeeReader(r, h))
	sum := base64.StdEncoding.EncodeToString(h.Sum(nil))
	if err == nil && size > 0 && n != size {
		err = fmt.Errorf("%s: %d bytes written, the asset has %d", name, n, size)
	}
	if err == nil && checksum != "" && sum != checksum {
		err = fmt.Errorf("%s: the downloaded bytes have the SHA-1 %s, the asset's checksum is %s", name, sum, checksum)
	}
	if err == nil {
		if s, ok := f.(interface{ Sync() error }); ok {
			err = s.Sync()
		}
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = fshelper.Rename(fsys, tmp, name)
	}
	if err != nil {
		_ = fshelper.Remove(fsys, tmp)
		return "", err
	}
	return sum, nil
}

// createPartial creates the .partial file of name, ".<name>.<random>.partial" in name's folder,
// under a name no file has yet, as os.CreateTemp does, and returns it with its name.
func createPartial(fsys fs.FS, name string) (fshelper.WFile, string, error) {
	dir, base := path.Split(name)
	for range 10000 {
		tmp := path.Join(dir, "."+base+"."+strconv.FormatUint(uint64(rand.Uint32()), 10)+".partial")
		f, err := fshelper.OpenFile(fsys, tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if !errors.Is(err, fs.ErrExist) {
			return f, tmp, err
		}
	}
	return nil, "", fmt.Errorf("%s: no free name for its .partial file", name)
}

func (w *LocalAssetWriter) pathOfAsset(a *assets.Asset) string {
	d := a.CaptureDate
	if d.IsZero() {
		return "no-date"
	}
	p := path.Join(fmt.Sprintf("%04d", d.Year()), fmt.Sprintf("%04d-%02d", d.Year(), d.Month()))
	return p
}
