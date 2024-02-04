package files

import (
	"context"
	"io/fs"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/simulot/immich-go/browser"
	"github.com/simulot/immich-go/helpers/fshelper"
	"github.com/simulot/immich-go/immich/metadata"
	"github.com/simulot/immich-go/logger"
)

type LocalAssetBrowser struct {
	fsyss  []fs.FS
	albums map[string]string
	log    *logger.Journal
}

func NewLocalFiles(ctx context.Context, log *logger.Journal, fsyss ...fs.FS) (*LocalAssetBrowser, error) {
	return &LocalAssetBrowser{
		fsyss:  fsyss,
		albums: map[string]string{},
		log:    log,
	}, nil
}

var toOldDate = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)

func (la *LocalAssetBrowser) Browse(ctx context.Context) chan *browser.LocalAssetFile {
	fileChan := make(chan *browser.LocalAssetFile)
	// Browse all given FS to collect the list of files
	go func(ctx context.Context) {
		defer close(fileChan)
		for _, fsys := range la.fsyss {
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
						if d.IsDir() {
							return la.handleFolder(ctx, fsys, fileChan, name)
						}
					}
					return nil
				})
			if err != nil {
				// Check if the context has been cancelled before sending the error
				select {
				case <-ctx.Done():
					// If the context has been cancelled, return immediately
					return
				case fileChan <- &browser.LocalAssetFile{
					Err: err,
				}:
				}
			}
		}
	}(ctx)

	return fileChan
}

func (la *LocalAssetBrowser) handleFolder(ctx context.Context, fsys fs.FS, fileChan chan *browser.LocalAssetFile, folder string) error {
	entries, err := fs.ReadDir(fsys, folder)
	if err != nil {
		return err
	}

	// fileMap := map[string][]fs.DirEntry{}
	// for _, e := range entries {
	// 	if e.IsDir() {
	// 		continue
	// 	}
	// 	ext := path.Ext(e.Name())
	// 	_, err := fshelper.MimeFromExt(ext)
	// 	if strings.ToLower(ext) == ".xmp" || err == nil {
	// 		base := strings.TrimSuffix(e.Name(), ext)
	// 		fileMap[base] = append(fileMap[base], e)
	// 	}
	// }

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		fileName := path.Join(folder, e.Name())
		la.log.AddEntry(fileName, logger.DISCOVERED_FILE, "")
		name := e.Name()
		ext := strings.ToLower(path.Ext(name))
		if fshelper.IsMetadataExt(ext) {
			la.log.AddEntry(name, logger.METADATA, "")
			continue
		} else if fshelper.IsIgnoredExt(ext) {
			la.log.AddEntry(fileName, logger.UNSUPPORTED, "")
			continue
		}
		m, err := fshelper.MimeFromExt(strings.ToLower(ext))
		if err != nil {
			la.log.AddEntry(fileName, logger.UNSUPPORTED, "")
			continue
		}
		ss := strings.Split(m[0], "/")
		if ss[0] == "image" {
			la.log.AddEntry(name, logger.SCANNED_IMAGE, "")
		} else {
			la.log.AddEntry(name, logger.SCANNED_VIDEO, "")
		}

		f := browser.LocalAssetFile{
			FSys:      fsys,
			FileName:  path.Join(folder, name),
			Title:     path.Base(name),
			FileSize:  0,
			Err:       err,
			DateTaken: metadata.TakeTimeFromName(filepath.Base(name)),
		}

		s, err := e.Info()
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
			la.checkSidecar(fsys, &f, entries, folder, name)
		}
		// Check if the context has been cancelled
		select {
		case <-ctx.Done():
			// If the context has been cancelled, return immediately
			return ctx.Err()
		default:
			fileChan <- &f
		}
	}
	return nil
}

func (la *LocalAssetBrowser) checkSidecar(fsys fs.FS, f *browser.LocalAssetFile, entries []fs.DirEntry, dir, name string) bool {
	assetBase := baseNames(name)

	for _, name := range assetBase {
		xmp := name + ".[xX][mM][pP]"
		for _, e := range entries {
			m, err := path.Match(xmp, e.Name())
			if err != nil {
				panic(err)
			}
			if m {
				f.SideCar = &metadata.SideCar{
					FileName: path.Join(dir, e.Name()),
					OnFSsys:  true,
				}
				la.log.AddEntry(name, logger.ASSOCIATED_META, "")
				return true
			}
		}
	}
	return false
}

func baseNames(n string) []string {
	n = escapeName(n)
	names := []string{n}
	ext := path.Ext(n)
	for {
		if ext == "" {
			return names
		}
		_, err := fshelper.MimeFromExt(ext)
		if err != nil {
			return names
		}
		n = strings.TrimSuffix(n, ext)
		names = append(names, n, n+".*")
		ext = path.Ext(n)
	}
}

func escapeName(n string) string {
	b := strings.Builder{}
	for _, c := range n {
		switch c {
		case '*', '?', '[', ']', '^':
			b.WriteRune('\\')
		case '\\':
			if runtime.GOOS != "windows" {
				b.WriteRune('\\')
			}
		}
		b.WriteRune(c)
	}
	return b.String()
}

func (la *LocalAssetBrowser) addAlbum(dir string) {
	base := path.Base(dir)
	la.albums[dir] = base
}

func (la *LocalAssetBrowser) ReadMetadataFromFile(a *browser.LocalAssetFile) error {
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
