package folder

import (
	"context"
	"errors"
	"io/fs"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/simulot/immich-go/adapters"
	"github.com/simulot/immich-go/helpers/gen"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/simulot/immich-go/internal/metadata"
	"golang.org/x/exp/constraints"
)

type fileLinks struct {
	image   string
	video   string
	sidecar string
}

type LocalAssetBrowser struct {
	fsyss    []fs.FS
	albums   map[string]string
	catalogs map[fs.FS]map[string][]string
	log      *fileevent.Recorder
	flags    *ImportFolderOptions
	exiftool *metadata.ExifTool
}

func NewLocalFiles(ctx context.Context, l *fileevent.Recorder, flags *ImportFolderOptions, fsyss ...fs.FS) (*LocalAssetBrowser, error) {
	if flags.ImportIntoAlbum != "" && flags.UsePathAsAlbumName != FolderModeNone {
		return nil, errors.New("cannot use both --into-album and --folder-as-album")
	}

	la := LocalAssetBrowser{
		fsyss:    fsyss,
		albums:   map[string]string{},
		catalogs: map[fs.FS]map[string][]string{},
		flags:    flags,
		log:      l,
	}

	if flags.ExifToolFlags.UseExifTool {
		et, err := metadata.NewExifTool(&flags.ExifToolFlags)
		if err != nil {
			return nil, err
		}
		la.exiftool = et
	}

	return &la, nil
}

func (la *LocalAssetBrowser) Browse(ctx context.Context) (chan *adapters.AssetGroup, error) {
	for _, fsys := range la.fsyss {
		err := la.passOneFsWalk(ctx, fsys)
		if err != nil {
			return nil, err
		}
	}
	return la.passTwo(ctx), nil
}

func (la *LocalAssetBrowser) passOneFsWalk(ctx context.Context, fsys fs.FS) error {
	la.catalogs[fsys] = map[string][]string{}
	err := fs.WalkDir(fsys, ".",
		func(name string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				if !la.flags.Recursive && name != "." {
					return fs.SkipDir
				}
				if la.flags.BannedFiles.Match(name) {
					la.log.Record(ctx, fileevent.DiscoveredDiscarded, fileevent.AsFileAndName(fsys, name), "reason", "banned folder")
					return fs.SkipDir
				}
				la.catalogs[fsys][name] = []string{}
				return nil
			}
			select {
			case <-ctx.Done():
				// If the context has been cancelled, return immediately
				return ctx.Err()
			default:
				if la.flags.BannedFiles.Match(name) {
					la.log.Record(ctx, fileevent.DiscoveredDiscarded, fileevent.AsFileAndName(fsys, name), "reason", "banned file")
					return nil
				}

				dir, base := filepath.Split(name)
				dir = strings.TrimSuffix(dir, "/")
				if dir == "" {
					dir = "."
				}
				ext := filepath.Ext(base)
				mediaType := la.flags.SupportedMedia.TypeFromExt(ext)

				if mediaType == metadata.TypeUnknown {
					la.log.Record(ctx, fileevent.DiscoveredUnsupported, fileevent.AsFileAndName(fsys, name), "reason", "unsupported file type")
					return nil
				}

				cat := la.catalogs[fsys][dir]

				switch mediaType {
				case metadata.TypeImage:
					la.log.Record(ctx, fileevent.DiscoveredImage, fileevent.AsFileAndName(fsys, name))
				case metadata.TypeVideo:
					la.log.Record(ctx, fileevent.DiscoveredVideo, fileevent.AsFileAndName(fsys, name))
				case metadata.TypeSidecar:
					la.log.Record(ctx, fileevent.DiscoveredSidecar, fileevent.AsFileAndName(fsys, name))
					if la.flags.IgnoreSideCarFiles {
						la.log.Record(ctx, fileevent.DiscoveredDiscarded, fileevent.AsFileAndName(fsys, name), "reason", "sidecar file ignored")
						return nil
					}
				}

				if !la.flags.InclusionFlags.IncludedExtensions.Include(ext) {
					la.log.Record(ctx, fileevent.DiscoveredDiscarded, fileevent.AsFileAndName(fsys, name), "reason", "extension not included")
					return nil
				}

				if la.flags.InclusionFlags.ExcludedExtensions.Exclude(ext) {
					la.log.Record(ctx, fileevent.DiscoveredDiscarded, fileevent.AsFileAndName(fsys, name), "reason", "extension excluded")
					return nil
				}

				la.catalogs[fsys][dir] = append(cat, name)
			}
			return nil
		})
	return err
}

func (la *LocalAssetBrowser) passTwo(ctx context.Context) chan *adapters.AssetGroup {
	fileChan := make(chan *adapters.AssetGroup)
	// Browse all given FS to collect the list of files
	go func(ctx context.Context) {
		defer close(fileChan)
		var err error
		if la.exiftool != nil {
			defer la.exiftool.Close()
		}

		errFn := func(name fileevent.FileAndName, err error) {
			if err != nil {
				la.log.Record(ctx, fileevent.Error, name, "error", err.Error())
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
					if la.flags.SupportedMedia.TypeFromExt(ext) == metadata.TypeImage {
						base := strings.TrimSuffix(file, ext)
						linked := links[base]
						linked.image = file
						links[base] = linked
					}
				}

				// Link videos and XMP to the image when needed
				for _, file := range files {
					ext := path.Ext(file)
					t := la.flags.SupportedMedia.TypeFromExt(ext)
					if t == metadata.TypeImage {
						continue
					}

					base := strings.TrimSuffix(file, ext)
					switch t {
					case metadata.TypeSidecar:
						if image, ok := links[file]; ok {
							// file.ext.XMP -> file.exp
							image.sidecar = file
							links[file] = image
							continue
						}
						if image, ok := links[base]; ok {
							// file.XMP -> file.JPG
							image.sidecar = file
							links[base] = image
							continue
						}

					case metadata.TypeVideo:
						if image, ok := links[file]; ok {
							// file.MP.jpg -> file.MP
							image.video = file
							links[file] = image
							continue
						}
						if image, ok := links[base]; ok {
							// file.MP4 -> file.JPG
							image.video = file
							links[base] = image
							continue
						}
						// Unlinked video
						links[file] = fileLinks{video: file}
					}
				}

				files = gen.MapKeys(links)
				sort.Strings(files)

				for _, file := range files {
					var a *adapters.LocalAssetFile
					var g *adapters.AssetGroup
					linked := links[file]

					switch {
					case linked.image != "" && linked.video != "":
						kind := adapters.GroupKindMotionPhoto
						v, err := la.assetFromFile(ctx, fsys, linked.video)
						if err != nil {
							// If the video file is not found, remove the video's link
							errFn(fileevent.AsFileAndName(fsys, linked.video), err)
							linked.video = ""
							kind = adapters.GroupKindNone
							if v != nil {
								v.Close()
							}
							v = nil
						}

						a, err = la.assetFromFile(ctx, fsys, linked.image)
						if err != nil {
							// If the image file is not found, remove the photo's link
							linked.image = ""
							errFn(fileevent.AsFileAndName(fsys, linked.image), err)
							kind = adapters.GroupKindNone

							if a != nil {
								a.Close()
								a = nil
							}
						}
						if a == nil && v == nil {
							continue
						}
						if a != nil && linked.sidecar != "" {
							la.log.Record(ctx, fileevent.AnalysisAssociatedMetadata, a, "sidecar", linked.sidecar)
							a.SideCar = metadata.SideCarFile{
								FSys:     fsys,
								FileName: linked.sidecar,
							}
						}

						// The video must be the first asset in the group
						g = adapters.NewAssetGroup(kind, v, a)

					case linked.image != "" && linked.video == "":
						// image alone
						a, err = la.assetFromFile(ctx, fsys, linked.image)
						if err != nil {
							errFn(fileevent.AsFileAndName(fsys, linked.image), err)
							continue
						}
						if linked.sidecar != "" {
							la.log.Record(ctx, fileevent.AnalysisAssociatedMetadata, a, "sidecar", linked.sidecar)
							a.SideCar = metadata.SideCarFile{
								FSys:     fsys,
								FileName: linked.sidecar,
							}
						}
						g = adapters.NewAssetGroup(adapters.GroupKindNone, a)

					case linked.video != "" && linked.image == "":
						// video alone
						a, err = la.assetFromFile(ctx, fsys, linked.video)
						if err != nil {
							errFn(fileevent.AsFileAndName(fsys, linked.video), err)
							continue
						}
						if linked.sidecar != "" {
							la.log.Record(ctx, fileevent.AnalysisAssociatedMetadata, a, "sidecar", linked.sidecar)
							a.SideCar = metadata.SideCarFile{
								FSys:     fsys,
								FileName: linked.sidecar,
							}
						}
						g = adapters.NewAssetGroup(adapters.GroupKindNone, a)
					}

					// If the asset is not found, skip it
					if g == nil || g.Validate() != nil {
						continue
					}

					// manage album options
					if la.flags.ImportIntoAlbum != "" {
						g.Albums = append(g.Albums, adapters.LocalAlbum{
							Path:  a.FileName,
							Title: la.flags.ImportIntoAlbum,
						})
					} else if la.flags.UsePathAsAlbumName != FolderModeNone {
						switch la.flags.UsePathAsAlbumName {
						case FolderModeFolder:
							title := filepath.Base(filepath.Dir(a.FileName))
							if title == "." {
								if fsys, ok := fsys.(fshelper.NameFS); ok {
									title = fsys.Name()
								}
							}
							if title != "" {
								g.Albums = append(g.Albums, adapters.LocalAlbum{
									Path:  a.FileName,
									Title: title,
								})
							}
						case FolderModePath:
							parts := []string{}
							if fsys, ok := fsys.(fshelper.NameFS); ok {
								parts = append(parts, fsys.Name())
							}
							path := filepath.Dir(a.FileName)
							if path != "." {
								parts = append(parts, strings.Split(path, "/")...) // TODO: Check on windows
							}
							Title := strings.Join(parts, la.flags.AlbumNamePathSeparator)
							g.Albums = append(g.Albums, adapters.LocalAlbum{
								Path:  filepath.Dir(a.FileName),
								Title: Title,
							})
						}
					}

					gs := []*adapters.AssetGroup{g}

					// Check if all assets share the same capture date
					if g.Kind == adapters.GroupKindMotionPhoto {
						baseDate := g.Assets[0].CaptureDate
						if baseDate.IsZero() {
							baseDate = g.Assets[0].FileDate
						}
						for i, a := range g.Assets[1:] {
							aDate := a.CaptureDate
							if aDate.IsZero() {
								aDate = a.FileDate
							}
							if abs(baseDate.Sub(aDate)) > 1*time.Minute {
								// take this asset out of the group
								g.Assets = append(g.Assets[:i+1], g.Assets[i+2:]...)
								// create a group for this assed
								g2 := adapters.NewAssetGroup(adapters.GroupKindNone, a)
								g2.Albums = g.Albums
								gs = append(gs, g2)
							}
						}
						if len(g.Assets) == 1 {
							g.Kind = adapters.GroupKindNone
						}
					}
					for _, g := range gs {
						select {
						case <-ctx.Done():
							return
						default:
							if len(g.Assets) > 0 {
								fileChan <- g
							}
						}
					}
				}
			}
		}
	}(ctx)

	return fileChan
}

func (la *LocalAssetBrowser) assetFromFile(ctx context.Context, fsys fs.FS, name string) (*adapters.LocalAssetFile, error) {
	a := &adapters.LocalAssetFile{
		FileName: name,
		Title:    filepath.Base(name),
		FSys:     fsys,
	}

	md, err := a.ReadMetadata(la.flags.DateHandlingFlags.Method, adapters.ReadMetadataOptions{
		ExifTool:         la.exiftool,
		ExiftoolTimezone: la.flags.ExifToolFlags.Timezone.Location(),
		FilenameTimeZone: la.flags.DateHandlingFlags.FilenameTimeZone.Location(),
	})
	if err != nil {
		a.Close()
		return nil, err
	}

	i, err := fs.Stat(fsys, name)
	if err != nil {
		a.Close()
		return nil, err
	}
	a.FileSize = int(i.Size())
	a.FileDate = i.ModTime()

	if md != nil {
		a.CaptureDate = md.DateTaken
	}

	if la.flags.InclusionFlags.DateRange.IsSet() && !la.flags.InclusionFlags.DateRange.InRange(a.CaptureDate) {
		a.Close()
		la.log.Record(ctx, fileevent.DiscoveredDiscarded, a, "reason", "asset outside date range")
		return nil, nil
	}
	return a, nil
}

func abs[T constraints.Integer](a T) T {
	if a < 0 {
		return -a
	}
	return a
}
