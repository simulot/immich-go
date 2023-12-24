package files

import (
	"context"
	"io/fs"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/simulot/immich-go/browser"
	"github.com/simulot/immich-go/helpers/fshelper"
	"github.com/simulot/immich-go/immich/metadata"
	"github.com/simulot/immich-go/journal"
	"github.com/simulot/immich-go/logger"
)

type LocalAssetBrowser struct {
	fsyss  []fs.FS
	albums map[string]string
	log    logger.Logger
	conf   *browser.Configuration
}

func NewLocalFiles(ctx context.Context, log logger.Logger, conf *browser.Configuration, fsyss ...fs.FS) (*LocalAssetBrowser, error) {
	return &LocalAssetBrowser{
		fsyss:  fsyss,
		albums: map[string]string{},
		log:    log,
		conf:   conf,
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

	fileMap := map[string][]fs.DirEntry{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := path.Ext(e.Name())
		_, err := fshelper.MimeFromExt(ext)
		if strings.ToLower(ext) == ".xmp" || err == nil {
			base := strings.TrimSuffix(e.Name(), ext)
			fileMap[base] = append(fileMap[base], e)
		}
	}

	for _, es := range fileMap {

		for _, e := range es {
			fileName := path.Join(folder, e.Name())
			la.conf.Journal.AddEntry(fileName, journal.DISCOVERED_FILE, "")
			name := e.Name()
			ext := strings.ToLower(path.Ext(name))
			if fshelper.IsIgnoredExt(ext) {

			}
			if _, err := fshelper.MimeFromExt(strings.ToLower(ext)); err != nil {
				la.conf.Journal.AddEntry(fileName, journal.UNSUPPORTED, "")
				continue
			}
			if !la.conf.SelectExtensions.Include(ext) {
				la.conf.Journal.AddEntry(fileName, journal.DISCARDED, "because of select-type option")
				continue
			}
			if la.conf.ExcludeExtensions.Exclude(ext) {
				la.conf.Journal.AddEntry(fileName, journal.DISCARDED, "because of exclude-type option")
				continue
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
				if !la.checkSidecar(fsys, &f, name+".xmp") {
					la.checkSidecar(fsys, &f, strings.TrimSuffix(name, ext)+".xmp")
				}
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
	}
	return nil
}

func (la *LocalAssetBrowser) checkSidecar(fsys fs.FS, f *browser.LocalAssetFile, name string) bool {
	_, err := fs.Stat(fsys, name+".xmp")
	if err == nil {
		la.log.Debug("  found sidecar: '%s'", name)
		f.SideCar = &metadata.SideCar{
			FileName: name + ".xmp",
			OnFSsys:  true,
		}
		return true
	}
	return false
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
