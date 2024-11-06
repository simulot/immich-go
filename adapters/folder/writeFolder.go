package folder

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/simulot/immich-go/internal/xmp"
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
	base := a.NameInfo().Base
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
		r, err := a.Open()
		if err != nil {
			return err
		}
		defer r.Close()

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			err = fshelper.WriteFile(w.WriteToFS, path.Join(dir, base), r)
			if err == nil && !a.SideCar.IsSet() {
				// No sidecar file, write XMP data
				var f fshelper.WFile
				f, err = fshelper.OpenFile(w.WriteToFS, path.Join(dir, base)+".xmp", os.O_RDWR|os.O_CREATE, 0o644)
				if err != nil {
					return err
				}
				defer f.Close()
				err = xmp.WriteXMP(a, f)
				if err != nil {
					return err
				}
			} else if a.SideCar.IsSet() {
				// Sidecar file is set, copy it
				var scr fs.File
				scr, err = a.FSys.Open(a.SideCar.FileName)
				if err != nil {
					return err
				}
				defer scr.Close()
				var scw fshelper.WFile
				scw, err = fshelper.OpenFile(w.WriteToFS, path.Join(dir, path.Base(a.SideCar.FileName)), os.O_RDWR|os.O_CREATE, 0o644)
				if err != nil {
					return err
				}
				defer scw.Close()
				_, err = io.Copy(scw, scr)
			}
			return err
		}
	}
}

func (w *LocalAssetWriter) pathOfAsset(a *assets.Asset) string {
	d := a.CaptureDate
	if d.IsZero() {
		d = a.NameInfo().Taken
		if d.IsZero() {
			d = a.ModTime()
		}
	}
	p := path.Join(fmt.Sprintf("%04d", d.Year()), fmt.Sprintf("%04d-%02d", d.Year(), d.Month()))
	return p
}
