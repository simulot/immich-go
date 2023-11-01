package files

import (
	"context"
	"immich-go/assets"
	"immich-go/immich/metadata"
	"io/fs"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type LocalAssetBrowser struct {
	fsys   fs.FS
	albums map[string]string
}

func NewLocalFiles(ctx context.Context, fsys fs.FS) (*LocalAssetBrowser, error) {
	return &LocalAssetBrowser{
		fsys:   fsys,
		albums: map[string]string{},
	}, nil
}

// func (la *LocalAssetBrowser) Stat(name string) (fs.FileInfo, error) {
// 	return fs.Stat(fsys.FS, name)
// }

var toOldDate = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)

func (la *LocalAssetBrowser) Browse(ctx context.Context) chan *assets.LocalAssetFile {
	fileChan := make(chan *assets.LocalAssetFile)
	// Browse all given FS to collect the list of files
	go func(ctx context.Context) {
		defer close(fileChan)
		err := fs.WalkDir(la.fsys, ".",
			func(name string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}

				// Check if the context has been cancelled
				select {
				case <-ctx.Done():
					// If the context has been cancelled, return immediately
					return ctx.Err()
				default:
				}
				if d.IsDir() {
					return nil
				}

				f := assets.LocalAssetFile{
					FSys:     la.fsys,
					FileName: name,
					Title:    path.Base(name),

					FileSize:  0,
					Err:       err,
					DateTaken: metadata.TakeTimeFromName(filepath.Base(name)),
				}

				s, err := d.Info()
				if err != nil {
					f.Err = err
				} else {
					f.FileSize = int(s.Size())
					if f.DateTaken.IsZero() {
						err = la.ReadMetadataFromFile(&f)
						_ = err
						if f.DateTaken.Before(toOldDate) {
							f.DateTaken = time.Now()
						}
					}
					_, err = fs.Stat(la.fsys, name+".xmp")
					if err == nil {
						f.SideCar = &metadata.SideCar{
							FileName: name + ".xmp",
							OnFSsys:  true,
						}
					}
				}
				fileChan <- &f
				return nil
			})
		if err != nil {
			// Check if the context has been cancelled before sending the error
			select {
			case <-ctx.Done():
				// If the context has been cancelled, return immediately
				return
			case fileChan <- &assets.LocalAssetFile{
				Err: err,
			}:
			}
		}

	}(ctx)

	return fileChan
}

func (la *LocalAssetBrowser) addAlbum(dir string) {
	base := path.Base(dir)
	la.albums[dir] = base
}

func (la *LocalAssetBrowser) ReadMetadataFromFile(a *assets.LocalAssetFile) error {
	ext := strings.ToLower(path.Ext(a.FileName))

	// Open the file
	r, err := a.PartialSourceReader()

	if err != nil {
		return err
	}
	m, err := metadata.GetFromReader(r, ext)
	if err == nil {
		a.DateTaken = m.DateTaken
	}
	return err
}
