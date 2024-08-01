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
	catalogs    map[fs.FS]map[string][]string
	log         *fileevent.Recorder
	sm          immich.SupportedMedia
	bannedFiles namematcher.List // list of file pattern to be exclude
	whenNoDate  string
}

func NewLocalFiles(ctx context.Context, l *fileevent.Recorder, fsyss ...fs.FS) (*LocalAssetBrowser, error) {
	return &LocalAssetBrowser{
		fsyss:      fsyss,
		albums:     map[string]string{},
		catalogs:   map[fs.FS]map[string][]string{},
		log:        l,
		whenNoDate: "FILE",
		sm:         immich.DefaultSupportedMedia,
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
	la.catalogs[fsys] = map[string][]string{}
	err := fs.WalkDir(fsys, ".",
		func(name string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				la.catalogs[fsys][name] = []string{}
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

				cat := la.catalogs[fsys][dir]

				switch mediaType {
				case immich.TypeImage:
					la.log.Record(ctx, fileevent.DiscoveredImage, nil, name)
				case immich.TypeVideo:
					la.log.Record(ctx, fileevent.DiscoveredVideo, nil, name)
				case immich.TypeSidecar:
					la.log.Record(ctx, fileevent.DiscoveredSidecar, nil, name)
				}

				if la.bannedFiles.Match(name) {
					la.log.Record(ctx, fileevent.DiscoveredDiscarded, nil, name, "reason", "banned file")
					return nil
				}
				la.catalogs[fsys][dir] = append(cat, name)
			}
			return nil
		})
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
			dirs := gen.MapKeys(la.catalogs[fsys])
			sort.Strings(dirs)
			for _, dir := range dirs {
				links := map[string]fileLinks{}
				files := la.catalogs[fsys][dir]

				if len(files) == 0 {
					continue
				}

				// Scan images first
				for _, file := range files {
					ext := path.Ext(file)
					if la.sm.TypeFromExt(ext) == immich.TypeImage {
						linked := links[file]
						linked.image = file
						links[file] = linked
					}
				}

			next:
				for _, file := range files {
					ext := path.Ext(file)
					t := la.sm.TypeFromExt(ext)
					if t == immich.TypeImage {
						continue next
					}

					base := strings.TrimSuffix(file, ext)
					switch t {
					case immich.TypeSidecar:
						if image, ok := links[base]; ok {
							// file.ext.XMP -> file.ext
							image.sidecar = file
							links[base] = image
							continue next
						}
						for f := range links {
							if strings.TrimSuffix(f, path.Ext(f)) == base {
								if image, ok := links[f]; ok {
									// base.XMP -> base.ext
									image.sidecar = file
									links[f] = image
									continue next
								}
							}
						}
					case immich.TypeVideo:
						if image, ok := links[base]; ok {
							// file.MP.ext -> file.ext
							image.sidecar = file
							links[base] = image
							continue next
						}
						for f := range links {
							if strings.TrimSuffix(f, path.Ext(f)) == base {
								if image, ok := links[f]; ok {
									// base.MP4 -> base.ext
									image.video = file
									links[f] = image
									continue next
								}
							}
							if strings.TrimSuffix(f, path.Ext(f)) == file {
								if image, ok := links[f]; ok {
									// base.MP4 -> base.ext
									image.video = file
									links[f] = image
									continue next
								}
							}
						}
						// Unlinked video
						links[file] = fileLinks{video: file}
					}
				}

				files = gen.MapKeys(links)
				sort.Strings(files)
				for _, file := range files {
					var a *browser.LocalAssetFile
					linked := links[file]

					if linked.image != "" {
						a, err = la.assetFromFile(fsys, linked.image)
						if err != nil {
							errFn(linked.image, err)
							return
						}
						if linked.video != "" {
							a.LivePhoto, err = la.assetFromFile(fsys, linked.video)
							if err != nil {
								errFn(linked.video, err)
								return
							}
						}
					} else if linked.video != "" {
						a, err = la.assetFromFile(fsys, linked.video)
						if err != nil {
							errFn(linked.video, err)
							return
						}
					}

					if a != nil && linked.sidecar != "" {
						a.SideCar = metadata.SideCarFile{
							FSys:     fsys,
							FileName: linked.sidecar,
						}
						la.log.Record(ctx, fileevent.AnalysisAssociatedMetadata, nil, linked.sidecar, "main", a.FileName)
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
