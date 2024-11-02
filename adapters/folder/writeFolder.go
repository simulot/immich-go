package folder

import (
	"errors"
	"fmt"
	"io/fs"
	"path"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/fshelper"
)

type minimalFSWriter interface {
	fs.FS
	fshelper.FSCanWrite
}

type closer interface {
	Close() error
}
type LocalAssetWriter struct {
	WriteToFS minimalFSWriter
}

func NewLocalAssetWriter(fsys fs.FS, writeToPath string) (*LocalAssetWriter, error) {
	if fsys, ok := fsys.(minimalFSWriter); ok {
		return &LocalAssetWriter{
			WriteToFS: fsys,
		}, nil
	}
	return nil, errors.New("FS does not support writing")
}

func (w *LocalAssetWriter) Write(group *assets.Group) error {
	var err error

	if fsys, ok := w.WriteToFS.(closer); ok {
		defer fsys.Close()
	}
	for _, a := range group.Assets {
		err = errors.Join(err, w.WriteAsset(a))
	}
	return err
}

func (w *LocalAssetWriter) WriteAsset(a *assets.Asset) error {
	base := a.NameInfo().Base
	dir := w.pathOfAsset(a)
	err := w.WriteToFS.Mkdir(dir, 0o755)
	if err != nil {
		return err
	}
	r, err := a.Open()
	if err != nil {
		return err
	}
	defer r.Close()

	err = fshelper.WriteFile(w.WriteToFS, path.Join(dir, base), r)
	return err
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
