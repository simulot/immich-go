package folder

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/exif/sidecars"
	"github.com/simulot/immich-go/internal/exif/sidecars/jsonsidecar"
	"github.com/simulot/immich-go/internal/exif/sidecars/xmpsidecar"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/simulot/immich-go/internal/fshelper/debugfiles"
)

// type minimalFSWriter interface {
// 	fs.FS
// 	fshelper.FSCanWrite
// }

type closer interface {
	Close() error
}
type LocalAssetWriter struct {
	WriteToFS     fs.FS
	createdDir    map[string]struct{}
	SidecarFormat sidecars.SidecarFormat
	Log           *slog.Logger
}

func NewLocalAssetWriter(fsys fs.FS, writeToPath string, format sidecars.SidecarFormat, log *slog.Logger) (*LocalAssetWriter, error) {
	if _, ok := fsys.(fshelper.FSCanWrite); !ok {
		return nil, errors.New("FS does not support writing")
	}
	return &LocalAssetWriter{
		WriteToFS:     fsys,
		createdDir:    make(map[string]struct{}),
		SidecarFormat: format,
		Log:           log,
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
		r, err := a.OpenFile()
		if err != nil {
			return err
		}
		defer r.Close()

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Add an index to the file name if it already exists, or the XMP or JSON
			index := 0
			ext := path.Ext(base)
			radical := base[:len(base)-len(ext)]
			for {
				if index > 0 {
					base = fmt.Sprintf("%s~%d%s", radical, index, path.Ext(base))
				}
				_, err := fs.Stat(w.WriteToFS, path.Join(dir, base))
				if err == nil {
					index++
					continue
				}
				_, err = fs.Stat(w.WriteToFS, path.Join(dir, base+".XMP"))
				if err == nil {
					index++
					continue
				}
				_, err = fs.Stat(w.WriteToFS, path.Join(dir, base+".JSON"))
				if err == nil {
					index++
					continue
				}
				break
			}

			// write the asset
			err = fshelper.WriteFile(w.WriteToFS, path.Join(dir, base), r)
			if err != nil {
				return err
			}

			// Warn about data loss when using XMP-only format
			if w.SidecarFormat == sidecars.FormatXMP && a.FromApplication != nil {
				var lostFields []string
				if a.FromApplication.Trashed {
					lostFields = append(lostFields, "Trashed")
				}
				if a.FromApplication.Archived {
					lostFields = append(lostFields, "Archived")
				}
				if a.FromApplication.FromPartner {
					lostFields = append(lostFields, "FromPartner")
				}
				if len(lostFields) > 0 && w.Log != nil {
					w.Log.Warn("XMP format cannot preserve some metadata",
						"file", a.OriginalFileName,
						"lost_fields", lostFields)
				}
			}

			// Write XMP sidecar if format requires it
			if w.SidecarFormat.CreatesXMP() {
				if a.FromSideCar != nil && a.FromSideCar.File.FS() != nil {
					// Existing XMP sidecar file - copy it
					var scr fs.File
					scr, err = a.FromSideCar.File.Open()
					if err != nil {
						return err
					}
					debugfiles.TrackOpenFile(scr, a.FromSideCar.File.Name())
					defer scr.Close()
					defer debugfiles.TrackCloseFile(scr)
					var scw fshelper.WFile
					scw, err = fshelper.OpenFile(w.WriteToFS, path.Join(dir, base+".xmp"), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
					if err != nil {
						return err
					}
					_, err = io.Copy(scw, scr)
					scw.Close()
				} else if a.FromApplication != nil {
					// Generate XMP from FromApplication metadata
					var scw fshelper.WFile
					scw, err = fshelper.OpenFile(w.WriteToFS, path.Join(dir, base+".xmp"), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
					if err != nil {
						return err
					}
					err = xmpsidecar.Write(a.FromApplication, scw)
					scw.Close()
				}
			}

			// Write JSON sidecar if format requires it
			if w.SidecarFormat.CreatesJSON() && a.FromApplication != nil {
				var scw fshelper.WFile
				scw, err = fshelper.OpenFile(w.WriteToFS, path.Join(dir, base+".JSON"), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
				if err != nil {
					return err
				}
				err = jsonsidecar.Write(a.FromApplication, scw)
				scw.Close()
			}

			return err
		}
	}
}

func (w *LocalAssetWriter) pathOfAsset(a *assets.Asset) string {
	d := a.CaptureDate
	if d.IsZero() {
		return "no-date"
	}
	p := path.Join(fmt.Sprintf("%04d", d.Year()), fmt.Sprintf("%04d-%02d", d.Year(), d.Month()))
	return p
}
