package files

import (
	"context"
	"io/fs"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/simulot/immich-go/browser"
	"github.com/simulot/immich-go/helpers/fileevent"
	"github.com/simulot/immich-go/helpers/fshelper"
	"github.com/simulot/immich-go/helpers/gen"
	"github.com/simulot/immich-go/helpers/namematcher"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/immich/metadata"
)

type fileLinks struct {
	image   string
	video   string
	sidecar string
}

type LocalAssetBrowser struct {
	fsyss       []fs.FS
	albums      map[string]string
	catalogs    map[fs.FS]map[string]map[string]fileLinks // per FS, DIR and base name
	log         *fileevent.Recorder
	sm          immich.SupportedMedia
	bannedFiles namematcher.List // list of file pattern to be exclude
	whenNoDate  string
}

func NewLocalFiles(ctx context.Context, l *fileevent.Recorder, fsyss ...fs.FS) (*LocalAssetBrowser, error) {
	return &LocalAssetBrowser{
		fsyss:      fsyss,
		albums:     map[string]string{},
		catalogs:   map[fs.FS]map[string]map[string]fileLinks{},
		log:        l,
		whenNoDate: "FILE",
	}, nil
}

func (la *LocalAssetBrowser) SetSupportedMedia(sm immich.SupportedMedia) *LocalAssetBrowser {
	la.sm = sm
	return la
}

func (la *LocalAssetBrowser) SetBannedFiles(banned namematcher.List) *LocalAssetBrowser {
	la.bannedFiles = banned
	return la
}

func (la *LocalAssetBrowser) SetWhenNoDate(opt string) *LocalAssetBrowser {
	la.whenNoDate = opt
	return la
}

func (la *LocalAssetBrowser) Prepare(ctx context.Context) error {
	for _, fsys := range la.fsyss {
		err := la.passOneFsWalk(ctx, fsys)
		if err != nil {
			return err
		}
	}

	return nil
}

func (la *LocalAssetBrowser) passOneFsWalk(ctx context.Context, fsys fs.FS) error {
	fsCatalog := map[string]map[string]fileLinks{}
	err := fs.WalkDir(fsys, ".",
		func(name string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				fsCatalog[name] = map[string]fileLinks{}
				return nil
			}

			select {
			case <-ctx.Done():
				// If the context has been cancelled, return immediately
				return ctx.Err()
			default:
				dir, base := filepath.Split(name)
				dir = strings.TrimSuffix(dir, "/")
				if dir == "" {
					dir = "."
				}
				ext := filepath.Ext(base)
				mediaType := la.sm.TypeFromExt(ext)

				if mediaType == immich.TypeUnknown {
					la.log.Record(ctx, fileevent.DiscoveredUnsupported, nil, name, "reason", "unsupported file type")
					return nil
				}

				linkBase := strings.TrimSuffix(base, ext)
				for {
					e := path.Ext(linkBase)
					if la.sm.IsMedia(e) {
						linkBase = strings.TrimSuffix(linkBase, e)
						continue
					}
					break
				}
				dirLinks := fsCatalog[dir]
				links := dirLinks[linkBase]

				switch mediaType {
				case immich.TypeImage:
					links.image = name
					la.log.Record(ctx, fileevent.DiscoveredImage, nil, name)
				case immich.TypeVideo:
					links.video = name
					la.log.Record(ctx, fileevent.DiscoveredVideo, nil, name)
				case immich.TypeSidecar:
					links.sidecar = name
					la.log.Record(ctx, fileevent.DiscoveredSidecar, nil, name)
				}

				if la.bannedFiles.Match(name) {
					la.log.Record(ctx, fileevent.DiscoveredDiscarded, nil, name, "reason", "banned file")
					return nil
				}
				dirLinks[linkBase] = links
				fsCatalog[dir] = dirLinks
			}
			return nil
		})
	la.catalogs[fsys] = fsCatalog
	return err
}

func (la *LocalAssetBrowser) Browse(ctx context.Context) chan *browser.LocalAssetFile {
	fileChan := make(chan *browser.LocalAssetFile)
	// Browse all given FS to collect the list of files
	go func(ctx context.Context) {
		defer close(fileChan)
		var err error

		errFn := func(name string, err error) {
			if err != nil {
				la.log.Record(ctx, fileevent.Error, nil, name, "error", err.Error())
			}
		}
		for _, fsys := range la.fsyss {
			dirLinks := la.catalogs[fsys]
			dirKeys := gen.MapKeys(dirLinks)
			sort.Strings(dirKeys)
			for _, d := range dirKeys {
				linksList := la.catalogs[fsys][d]
				linksKeys := gen.MapKeys(linksList)
				sort.Strings(linksKeys)
				for _, l := range linksKeys {
					var a *browser.LocalAssetFile
					links := linksList[l]

					if links.image != "" {
						a, err = la.assetFromFile(fsys, links.image)
						if err != nil {
							errFn(links.image, err)
							return
						}
						if links.video != "" {
							a.LivePhoto, err = la.assetFromFile(fsys, links.video)
							if err != nil {
								errFn(links.video, err)
								return
							}
						}
					} else if links.video != "" {
						a, err = la.assetFromFile(fsys, links.video)
						if err != nil {
							errFn(links.video, err)
							return
						}
					}

					if a != nil && links.sidecar != "" {
						a.SideCar = metadata.SideCarFile{
							FSys:     fsys,
							FileName: links.sidecar,
						}
						la.log.Record(ctx, fileevent.AnalysisAssociatedMetadata, nil, links.sidecar, "main", a.FileName)
					}

					select {
					case <-ctx.Done():
						return
					default:
						if a != nil {
							fileChan <- a
						}
					}
				}
			}
		}
	}(ctx)

	return fileChan
}

var toOldDate = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)

func (la *LocalAssetBrowser) assetFromFile(fsys fs.FS, name string) (*browser.LocalAssetFile, error) {
	a := &browser.LocalAssetFile{
		FileName: name,
		Title:    filepath.Base(name),
		FSys:     fsys,
	}

	fullPath := name
	if fsys, ok := fsys.(fshelper.NameFS); ok {
		fullPath = filepath.Join(fsys.Name(), name)
	}
	a.Metadata.DateTaken = metadata.TakeTimeFromPath(fullPath)

	i, err := fs.Stat(fsys, name)
	if err != nil {
		return nil, err
	}
	a.FileSize = int(i.Size())
	if a.Metadata.DateTaken.IsZero() {
		err = la.ReadMetadataFromFile(a)
		if err != nil {
			return nil, err
		}
		if a.Metadata.DateTaken.Before(toOldDate) {
			switch la.whenNoDate {
			case "FILE":
				a.Metadata.DateTaken = i.ModTime()
			case "NOW":
				a.Metadata.DateTaken = time.Now()
			}
		}
	}
	return a, nil
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
		a.Metadata.DateTaken = m.DateTaken
	}
	return nil
}
