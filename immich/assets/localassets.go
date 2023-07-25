package assets

import (
	"context"
	"io/fs"
	"path"
	"strings"
)

func BrowseLocalAssets(ctx context.Context, fsys fs.FS) chan *LocalAssetFile {
	fileChan := make(chan *LocalAssetFile)
	// Browse all given FS to collect the list of files
	go func(ctx context.Context) {
		defer close(fileChan)
		err := fs.WalkDir(fsys, ".",
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
				ext := strings.ToLower(path.Ext(name))
				switch ext {
				case ".jpg", "jpeg", ".png", ".mp4", ".heic", ".mov", ".m4v", ".gif":

					s, err := d.Info()

					f := LocalAssetFile{
						srcFS:    fsys,
						FileName: name,
						size:     int(s.Size()),
						Album:    path.Base(path.Dir(name)),
						Err:      err,
					}
					// Check if the context has been cancelled before sending the file
					select {
					case <-ctx.Done():
						// If the context has been cancelled, return immediately
						return ctx.Err()
					case fileChan <- &f:
					}
					err = nil
				}
				return err
			})
		if err != nil {
			// Check if the context has been cancelled before sending the error
			select {
			case <-ctx.Done():
				// If the context has been cancelled, return immediately
				return
			case fileChan <- &LocalAssetFile{
				Err: err,
			}:
			}
		}

	}(ctx)

	return fileChan
}
