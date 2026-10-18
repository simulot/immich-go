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
	"os"
	"path"

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

type closer interface {
	Close() error
}
type LocalAssetWriter struct {
	WriteToFS  fs.FS
	createdDir map[string]struct{}
}

func NewLocalAssetWriter(fsys fs.FS, writeToPath string) (*LocalAssetWriter, error) {
	if _, ok := fsys.(fshelper.FSCanWrite); !ok {
		return nil, errors.New("FS does not support writing")
	}
	return &LocalAssetWriter{
		WriteToFS:  fsys,
		createdDir: make(map[string]struct{}),
	}, nil
}

func (w *LocalAssetWriter) WriteGroup(ctx context.Context, group *assets.Group) error {
	var err error

	if fsys, ok := w.WriteToFS.(closer); ok {
		defer fsys.Close()
	}
	for _, a := range group.Assets {
		select {
		case <-ctx.Done():
			return errors.Join(err, ctx.Err())
		default:
			err = errors.Join(err, w.WriteAsset(ctx, a))
		}
	}
	return err
}

func (w *LocalAssetWriter) WriteAsset(ctx context.Context, a *assets.Asset) error {
	base := a.Base
	dir := w.pathOfAsset(a)
	if _, ok := w.createdDir[dir]; !ok {
		err := fshelper.MkdirAll(w.WriteToFS, dir, 0o755)
		if err != nil {
			return err
		}
		w.createdDir[dir] = struct{}{}
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Choose the file name before the asset is opened: opening it is what downloads it. The name is
	// the first of base, radical~1.ext, radical~2.ext, ... that is free, or that already holds a
	// regular file of the asset's size. Such a file is this asset, written by an earlier run, so it
	// is not downloaded again and no ~N copy is made. A different asset with the same name (another
	// size) still gets the next free ~N. Immich originals never change in place.
	// When the asset's size is not known (0: the server has not read the file yet), the files
	// already under those names are kept in taken, and compared with the download below.
	ext := path.Ext(base)
	radical := base[:len(base)-len(ext)]
	name, present := base, false
	var taken []string
	for index := 0; ; index++ {
		if index > 0 {
			name = fmt.Sprintf("%s~%d%s", radical, index, ext)
		}
		if fi, err := fs.Stat(w.WriteToFS, path.Join(dir, name)); err == nil {
			if fi.Mode().IsRegular() {
				if a.FileSize > 0 && fi.Size() == int64(a.FileSize) {
					present = true
					break
				}
				if a.FileSize <= 0 {
					taken = append(taken, name)
				}
			}
			continue
		}
		if _, err := fs.Stat(w.WriteToFS, path.Join(dir, name+".XMP")); err == nil {
			continue
		}
		if _, err := fs.Stat(w.WriteToFS, path.Join(dir, name+".JSON")); err == nil {
			continue
		}
		break
	}

	if !present {
		r, err := a.OpenFile()
		if err != nil {
			return err
		}
		defer r.Close()
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		// Size unknown: the asset is already here if one of the taken names holds the same bytes.
		// Only then is the download compared; the cost is one download per run until the size is
		// known, and never a ~N copy.
		if len(taken) > 0 {
			if match, err := sameContent(w.WriteToFS, dir, taken, r, a.Checksum); err != nil {
				return err
			} else if match != "" {
				name, present = match, true
			}
		}
		if !present {
			// Under a temporary name first: a download cut short never sits under the asset's
			// name, where the next run would take it for another asset. With the size known, the
			// bytes must be exactly that many, whatever the transport said.
			err = writeAtomic(w.WriteToFS, path.Join(dir, name), r, int64(a.FileSize))
			if err != nil {
				return err
			}
		}
	}

	// XMP?
	if a.FromSideCar != nil {
		// Sidecar file is set, copy it
		scr, err := a.FromSideCar.File.Open()
		if err != nil {
			return err
		}
		debugfiles.TrackOpenFile(scr, a.FromSideCar.File.Name())
		defer scr.Close()
		defer debugfiles.TrackCloseFile(scr)
		scw, err := fshelper.OpenFile(w.WriteToFS, path.Join(dir, name+".XMP"), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
		if err != nil {
			return err
		}
		_, err = io.Copy(scw, scr)
		scw.Close()
		if err != nil {
			return err
		}
	}

	// Having metadata from an Application or immich-go JSON? Written only when its content
	// changed, through a temporary file.
	if a.FromApplication != nil {
		var buf bytes.Buffer
		err := jsonsidecar.Write(a.FromApplication, &buf)
		if err != nil {
			return err
		}
		jsonName := path.Join(dir, name+".JSON")
		old, err := fs.ReadFile(w.WriteToFS, jsonName)
		if err != nil || !bytes.Equal(old, buf.Bytes()) {
			return writeAtomic(w.WriteToFS, jsonName, &buf, 0)
		}
	}
	return nil
}

// sameContent returns the first of names (in dir) whose bytes equal r's, or "". r is rewound.
// checksum is the asset's SHA-1, base64 (a.Checksum): after Asset.OpenFile it is the SHA-1 the
// cache reader computed while it copied the download to r, so r is not read again. Without a
// valid one (a local file opened in place, where OpenFile computes none), r is hashed here.
func sameContent(fsys fs.FS, dir string, names []string, r osfs.OSFS, checksum string) (string, error) {
	fi, err := r.Stat()
	if err != nil {
		return "", err
	}
	want, err := base64.StdEncoding.DecodeString(checksum)
	if err != nil || len(want) != sha1.Size {
		want = nil
	}
	for _, n := range names {
		cfi, err := fs.Stat(fsys, path.Join(dir, n))
		if err != nil || cfi.Size() != fi.Size() {
			continue
		}
		if want == nil {
			if want, err = sha1Of(r); err != nil {
				return "", err
			}
		}
		f, err := fsys.Open(path.Join(dir, n))
		if err != nil {
			continue
		}
		got, err := sha1Of(f)
		f.Close()
		if err == nil && bytes.Equal(got, want) {
			_, err = r.Seek(0, io.SeekStart)
			return n, err
		}
	}
	_, err = r.Seek(0, io.SeekStart)
	return "", err
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

// writeAtomic writes r to ".<name>.partial" beside name, syncs it, then renames it to name. With
// size > 0, r must give exactly size bytes; otherwise the .partial file is removed and name is not
// written.
func writeAtomic(fsys fs.FS, name string, r io.Reader, size int64) error {
	tmp := path.Join(path.Dir(name), "."+path.Base(name)+".partial")
	f, err := fshelper.OpenFile(fsys, tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	n, err := io.Copy(f, r)
	if err == nil && size > 0 && n != size {
		err = fmt.Errorf("%s: %d bytes written, the asset has %d", name, n, size)
	}
	if err == nil {
		if s, ok := f.(interface{ Sync() error }); ok {
			err = s.Sync()
		}
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = fshelper.Remove(fsys, tmp)
		return err
	}
	return fshelper.Rename(fsys, tmp, name)
}

func (w *LocalAssetWriter) pathOfAsset(a *assets.Asset) string {
	d := a.CaptureDate
	if d.IsZero() {
		return "no-date"
	}
	p := path.Join(fmt.Sprintf("%04d", d.Year()), fmt.Sprintf("%04d-%02d", d.Year(), d.Month()))
	return p
}
