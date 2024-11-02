package folder

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/fshelper"
)

// type minimalFSWriter interface {
// 	fs.FS
// 	fshelper.FSCanWrite
// }

type closer interface {
	Close() error
}
type LocalAssetWriter struct {
	WriteToFS fs.FS
}

func NewLocalAssetWriter(fsys fs.FS, writeToPath string) (*LocalAssetWriter, error) {
	if _, ok := fsys.(fshelper.FSCanWrite); !ok {
		return nil, errors.New("FS does not support writing")
	}
	return &LocalAssetWriter{
		WriteToFS: fsys,
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
	err := fshelper.MkdirAll(w.WriteToFS, dir, 0o755)
	if err != nil {
		return err
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
			return fshelper.WriteFile(w.WriteToFS, path.Join(dir, base), r)
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
