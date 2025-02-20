package folder

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/simulot/immich-go/coverageTester"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/exif"
	"github.com/simulot/immich-go/internal/exif/sidecars/jsonsidecar"
	"github.com/simulot/immich-go/internal/exif/sidecars/xmpsidecar"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/filetypes"
	"github.com/simulot/immich-go/internal/filters"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/simulot/immich-go/internal/gen"
	"github.com/simulot/immich-go/internal/groups"
	"github.com/simulot/immich-go/internal/groups/burst"
	"github.com/simulot/immich-go/internal/groups/epsonfastfoto"
	"github.com/simulot/immich-go/internal/groups/series"
	"github.com/simulot/immich-go/internal/worker"
)

type LocalAssetBrowser struct {
	fsyss                   []fs.FS
	log                     *fileevent.Recorder
	flags                   *ImportFolderOptions
	pool                    *worker.Pool
	wg                      sync.WaitGroup
	groupers                []groups.Grouper
	requiresDateInformation bool                              // true if we need to read the date from the file for the options
	picasaAlbums            *gen.SyncMap[string, PicasaAlbum] // ap[string]PicasaAlbum
}

func NewLocalFiles(ctx context.Context, l *fileevent.Recorder, flags *ImportFolderOptions, fsyss ...fs.FS) (*LocalAssetBrowser, error) {


	coverageTester.WriteUniqueLine("NewLocalFiles - Branch 0/7 Covered")

	if flags.ImportIntoAlbum != "" && flags.UsePathAsAlbumName != FolderModeNone {

		coverageTester.WriteUniqueLine("NewLocalFiles - Branch 1/7 Covered")
		return nil, errors.New("cannot use both --into-album and --folder-as-album")
	}

	la := LocalAssetBrowser{
		fsyss: fsyss,
		flags: flags,
		log:   l,
		pool:  worker.NewPool(10), // TODO: Make this configurable
		requiresDateInformation: flags.InclusionFlags.DateRange.IsSet() ||
			flags.TakeDateFromFilename || flags.StackBurstPhotos ||
			flags.ManageHEICJPG != filters.HeicJpgNothing || flags.ManageRawJPG != filters.RawJPGNothing,
	}

	if flags.PicasaAlbum {
		coverageTester.WriteUniqueLine("NewLocalFiles - Branch 2/7 Covered")
		la.picasaAlbums = gen.NewSyncMap[string, PicasaAlbum]() // make(map[string]PicasaAlbum)
	}

	if flags.InfoCollector == nil {
		coverageTester.WriteUniqueLine("NewLocalFiles - Branch 3/7 Covered")
		flags.InfoCollector = filenames.NewInfoCollector(flags.TZ, flags.SupportedMedia)
	}

	if flags.InclusionFlags.DateRange.IsSet() {
		coverageTester.WriteUniqueLine("NewLocalFiles - Branch 4/7 Covered")
		flags.InclusionFlags.DateRange.SetTZ(flags.TZ)
	}

	if flags.SessionTag {
		coverageTester.WriteUniqueLine("NewLocalFiles - Branch 5/7 Covered")
		flags.session = fmt.Sprintf("{immich-go}/%s", time.Now().Format("2006-01-02 15:04:05"))
	}

	// if flags.ExifToolFlags.UseExifTool {
	// 	err := exif.NewExifTool(&flags.ExifToolFlags)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// }

	if flags.ManageEpsonFastFoto {
		coverageTester.WriteUniqueLine("NewLocalFiles - Branch 6/7 Covered")
		g := epsonfastfoto.Group{}
		la.groupers = append(la.groupers, g.Group)
	}
	if flags.ManageBurst != filters.BurstNothing {
		coverageTester.WriteUniqueLine("NewLocalFiles - Branch 7/7 Covered")
		la.groupers = append(la.groupers, burst.Group)
	}
	la.groupers = append(la.groupers, series.Group)

	return &la, nil
}

func (la *LocalAssetBrowser) Browse(ctx context.Context) chan *assets.Group {
	gOut := make(chan *assets.Group)
	go func() {
		defer close(gOut)
		for _, fsys := range la.fsyss {
			la.concurrentParseDir(ctx, fsys, ".", gOut)
		}
		la.wg.Wait()
		la.pool.Stop()
	}()
	return gOut
}

func (la *LocalAssetBrowser) concurrentParseDir(ctx context.Context, fsys fs.FS, dir string, gOut chan *assets.Group) {
	la.wg.Add(1)
	ctx, cancel := context.WithCancelCause(ctx)
	go la.pool.Submit(func() {
		defer la.wg.Done()
		err := la.parseDir(ctx, fsys, dir, gOut)
		if err != nil {
			la.log.Log().Error(err.Error())
			cancel(err)
		}
	})
}

func (la *LocalAssetBrowser) parseDir(ctx context.Context, fsys fs.FS, dir string, gOut chan *assets.Group) error {
	coverageTester.WriteUniqueLine("parseDir - Branch 0/53 Covered")

	fsName := ""
	if fsys, ok := fsys.(interface{ Name() string }); ok {
		coverageTester.WriteUniqueLine("parseDir - Branch 1/53 Covered")
		fsName = fsys.Name()
	}

	var as []*assets.Asset
	var entries []fs.DirEntry
	var err error

	select {
	case <-ctx.Done():
		coverageTester.WriteUniqueLine("parseDir - Branch 2/53 Covered")
		return ctx.Err()
	default:
		entries, err = fs.ReadDir(fsys, dir)
		if err != nil {
			coverageTester.WriteUniqueLine("parseDir - Branch 3/53 Covered")
			return err
		}
	}

	for _, entry := range entries {
		base := entry.Name()
		name := path.Join(dir, base)
		if entry.IsDir() {
			coverageTester.WriteUniqueLine("parseDir - Branch 4/53 Covered")
			continue
		}

		if la.flags.BannedFiles.Match(name) {
			coverageTester.WriteUniqueLine("parseDir - Branch 5/53 Covered")
			la.log.Record(ctx, fileevent.DiscoveredDiscarded, fshelper.FSName(fsys, entry.Name()), "reason", "banned file")
			continue
		}

		if la.flags.SupportedMedia.IsUseLess(name) {
			coverageTester.WriteUniqueLine("parseDir - Branch 6/53 Covered")
			la.log.Record(ctx, fileevent.DiscoveredUseless, fshelper.FSName(fsys, entry.Name()))
			continue
		}

		if la.flags.PicasaAlbum && (strings.ToLower(base) == ".picasa.ini" || strings.ToLower(base) == "picasa.ini") {
			coverageTester.WriteUniqueLine("parseDir - Branch 7/53 Covered")
			a, err := ReadPicasaIni(fsys, name)
			if err != nil {
				coverageTester.WriteUniqueLine("parseDir - Branch 8/53 Covered")
				la.log.Record(ctx, fileevent.Error, fshelper.FSName(fsys, name), "error", err.Error())
			} else {
				coverageTester.WriteUniqueLine("parseDir - Branch 9/53 Covered")
				la.picasaAlbums.Store(dir, a) // la.picasaAlbums[dir] = a
				la.log.Log().Info("Picasa album detected", "file", fshelper.FSName(fsys, path.Join(dir, name)), "album", a.Name)
			}
			continue
		}

		ext := filepath.Ext(base)
		mediaType := la.flags.SupportedMedia.TypeFromExt(ext)

		if mediaType == filetypes.TypeUnknown {
			coverageTester.WriteUniqueLine("parseDir - Branch 10/53 Covered")
			la.log.Record(ctx, fileevent.DiscoveredUnsupported, fshelper.FSName(fsys, name), "reason", "unsupported file type")
			continue
		}

		switch mediaType {
		case filetypes.TypeUseless:
			coverageTester.WriteUniqueLine("parseDir - Branch 11/53 Covered")
			la.log.Record(ctx, fileevent.DiscoveredUseless, fshelper.FSName(fsys, name))
			continue
		case filetypes.TypeImage:
			coverageTester.WriteUniqueLine("parseDir - Branch 12/53 Covered")
			la.log.Record(ctx, fileevent.DiscoveredImage, fshelper.FSName(fsys, name))
		case filetypes.TypeVideo:
			coverageTester.WriteUniqueLine("parseDir - Branch 13/53 Covered")
			la.log.Record(ctx, fileevent.DiscoveredVideo, fshelper.FSName(fsys, name))
		case filetypes.TypeSidecar:
			coverageTester.WriteUniqueLine("parseDir - Branch 14/53 Covered")
			if la.flags.IgnoreSideCarFiles {
				coverageTester.WriteUniqueLine("parseDir - Branch 15/53 Covered")
				la.log.Record(ctx, fileevent.DiscoveredDiscarded, fshelper.FSName(fsys, name), "reason", "sidecar file ignored")
				continue
			}
			la.log.Record(ctx, fileevent.DiscoveredSidecar, fshelper.FSName(fsys, name))
			continue
		}

		if !la.flags.InclusionFlags.IncludedExtensions.Include(ext) {
			coverageTester.WriteUniqueLine("parseDir - Branch 16/53 Covered")
			la.log.Record(ctx, fileevent.DiscoveredDiscarded, fshelper.FSName(fsys, name), "reason", "extension not included")
			continue
		}

		if la.flags.InclusionFlags.ExcludedExtensions.Exclude(ext) {
			coverageTester.WriteUniqueLine("parseDir - Branch 17/53 Covered")
			la.log.Record(ctx, fileevent.DiscoveredDiscarded, fshelper.FSName(fsys, name), "reason", "extension excluded")
			continue
		}

		select {
		case <-ctx.Done():
			coverageTester.WriteUniqueLine("parseDir - Branch 18/53 Covered")
			return ctx.Err()
		default:
			// we have a file to process
			a, err := la.assetFromFile(ctx, fsys, name)
			if err != nil {
				coverageTester.WriteUniqueLine("parseDir - Branch 19/53 Covered")
				la.log.Record(ctx, fileevent.Error, fshelper.FSName(fsys, name), "error", err.Error())
				return err
			}
			if a != nil {
				coverageTester.WriteUniqueLine("parseDir - Branch 20/53 Covered")
				as = append(as, a)
			}
		}
	}

	// process the left over dirs
	for _, entry := range entries {
		base := entry.Name()
		name := path.Join(dir, base)
		if entry.IsDir() {
			coverageTester.WriteUniqueLine("parseDir - Branch 21/53 Covered")
			if la.flags.BannedFiles.Match(name) {
				coverageTester.WriteUniqueLine("parseDir - Branch 22/53 Covered")
				la.log.Record(ctx, fileevent.DiscoveredDiscarded, fshelper.FSName(fsys, name), "reason", "banned folder")
				continue // Skip this folder, no error
			}
			if la.flags.Recursive && entry.Name() != "." {
				coverageTester.WriteUniqueLine("parseDir - Branch 23/53 Covered")
				la.concurrentParseDir(ctx, fsys, name, gOut)
			}
			continue
		}
	}

	in := make(chan *assets.Asset)
	go func() {
		defer close(in)

		sort.Slice(as, func(i, j int) bool {
			// Sort by radical first
			radicalI := as[i].Radical
			radicalJ := as[j].Radical
			if radicalI != radicalJ {
				coverageTester.WriteUniqueLine("parseDir - Branch 24/53 Covered")
				return radicalI < radicalJ
			}
			// If radicals are the same, sort by date
			return as[i].CaptureDate.Before(as[j].CaptureDate)
		})

		for _, a := range as {
			// check the presence of a JSON file
			jsonName, err := checkExistSideCar(fsys, a.File.Name(), ".json")
			if err == nil && jsonName != "" {
				coverageTester.WriteUniqueLine("parseDir - Branch 25/53 Covered")
				buf, err := fs.ReadFile(fsys, jsonName)
				if err != nil {
					coverageTester.WriteUniqueLine("parseDir - Branch 26/53 Covered")
					la.log.Record(ctx, fileevent.Error, nil, "error", err.Error())
				} else {
					if bytes.Contains(buf, []byte("immich-go version")) {
						coverageTester.WriteUniqueLine("parseDir - Branch 27/53 Covered")
						md := &assets.Metadata{}
						err = jsonsidecar.Read(bytes.NewReader(buf), md)
						if err != nil {
							coverageTester.WriteUniqueLine("parseDir - Branch 28/53 Covered")
							la.log.Record(ctx, fileevent.Error, nil, "error", err.Error())
						} else {
							md.File = fshelper.FSName(fsys, jsonName)
							a.FromApplication = a.UseMetadata(md) // Force the use of the metadata coming from immich export
							a.OriginalFileName = md.FileName      // Force the name of the file to be the one from the JSON file
						}
					} else {
						la.log.Log().Warn("JSON file detected but not from immich-go", "file", fshelper.FSName(fsys, jsonName))
					}
				}
			}
			// check the presence of a XMP file
			xmpName, err := checkExistSideCar(fsys, a.File.Name(), ".xmp")
			if err == nil && xmpName != "" {
				coverageTester.WriteUniqueLine("parseDir - Branch 29/53 Covered")
				buf, err := fs.ReadFile(fsys, xmpName)
				if err != nil {
					coverageTester.WriteUniqueLine("parseDir - Branch 30/53 Covered")
					la.log.Record(ctx, fileevent.Error, nil, "error", err.Error())
				} else {
					md := &assets.Metadata{}
					err = xmpsidecar.ReadXMP(bytes.NewReader(buf), md)
					if err != nil {
						coverageTester.WriteUniqueLine("parseDir - Branch 31/53 Covered")
						la.log.Record(ctx, fileevent.Error, nil, "error", err.Error())
					} else {
						md.File = fshelper.FSName(fsys, xmpName)
						a.FromSideCar = a.UseMetadata(md)
					}
				}
			}

			// Read metadata from the file only id needed (date range or take date from filename)
			if la.requiresDateInformation {
				coverageTester.WriteUniqueLine("parseDir - Branch 32/53 Covered")
				if a.CaptureDate.IsZero() {
					coverageTester.WriteUniqueLine("parseDir - Branch 33/53 Covered")
					// no date in XMP, JSON, try reading the metadata
					f, err := a.OpenFile()
					if err == nil {
						coverageTester.WriteUniqueLine("parseDir - Branch 34/53 Covered")
						md, err := exif.GetMetaData(f, a.Ext, la.flags.TZ)
						if err != nil {
							coverageTester.WriteUniqueLine("parseDir - Branch 35/53 Covered")
							la.log.Record(ctx, fileevent.INFO, a.File, "warning", err.Error())
						} else {
							a.FromSourceFile = a.UseMetadata(md)
						}
						if (md == nil || md.DateTaken.IsZero()) && !a.NameInfo.Taken.IsZero() && la.flags.TakeDateFromFilename {
							coverageTester.WriteUniqueLine("parseDir - Branch 36/53 Covered")
							// no exif, but we have a date in the filename and the TakeDateFromFilename is set
							a.FromApplication = &assets.Metadata{
								DateTaken: a.NameInfo.Taken,
							}
							a.CaptureDate = a.FromApplication.DateTaken
						}
						f.Close()
					}
				}
			}

			if !la.flags.InclusionFlags.DateRange.InRange(a.CaptureDate) {
				coverageTester.WriteUniqueLine("parseDir - Branch 37/53 Covered")
				a.Close()
				la.log.Record(ctx, fileevent.DiscoveredDiscarded, a.File, "reason", "asset outside date range")
				continue
			}

			// Add tags
			if len(la.flags.Tags) > 0 {
				coverageTester.WriteUniqueLine("parseDir - Branch 38/53 Covered")
				for _, t := range la.flags.Tags {
					a.AddTag(t)
				}
			}

			// Add folder as tags
			if la.flags.FolderAsTags {
				coverageTester.WriteUniqueLine("parseDir - Branch 39/53 Covered")
				t := fsName
				if dir != "." {
					coverageTester.WriteUniqueLine("parseDir - Branch 40/53 Covered")
					t = path.Join(t, dir)
				}
				if t != "" {
					coverageTester.WriteUniqueLine("parseDir - Branch 41/53 Covered")
					a.AddTag(t)
				}
			}

			// Manage albums
			if la.flags.ImportIntoAlbum != "" {
				coverageTester.WriteUniqueLine("parseDir - Branch 42/53 Covered")
				a.Albums = []assets.Album{{Title: la.flags.ImportIntoAlbum}}
			} else {
				done := false
				if la.flags.PicasaAlbum {
					coverageTester.WriteUniqueLine("parseDir - Branch 43/53 Covered")
					if album, ok := la.picasaAlbums.Load(dir); ok {
						coverageTester.WriteUniqueLine("parseDir - Branch 44/53 Covered")
						a.Albums = []assets.Album{{Title: album.Name, Description: album.Description}}
						done = true
					}
				}
				if !done && la.flags.UsePathAsAlbumName != FolderModeNone && la.flags.UsePathAsAlbumName != "" {
					coverageTester.WriteUniqueLine("parseDir - Branch 45/53 Covered")
					Album := ""
					switch la.flags.UsePathAsAlbumName {
					case FolderModeFolder:
						coverageTester.WriteUniqueLine("parseDir - Branch 46/53 Covered")
						if dir == "." {
							coverageTester.WriteUniqueLine("parseDir - Branch 47/53 Covered")
							Album = fsName
						} else {
							Album = filepath.Base(dir)
						}
					case FolderModePath:
						coverageTester.WriteUniqueLine("parseDir - Branch 48/53 Covered")
						parts := []string{}
						if fsName != "" {
							coverageTester.WriteUniqueLine("parseDir - Branch 49/53 Covered")
							parts = append(parts, fsName)
						}
						if dir != "." {
							coverageTester.WriteUniqueLine("parseDir - Branch 50/53 Covered")
							parts = append(parts, strings.Split(dir, "/")...)
							// parts = append(parts, strings.Split(dir, string(filepath.Separator))...)
						}
						Album = strings.Join(parts, la.flags.AlbumNamePathSeparator)
					}
					a.Albums = []assets.Album{{Title: Album}}
				}
			}

			if la.flags.SessionTag {
				coverageTester.WriteUniqueLine("parseDir - Branch 51/53 Covered")
				a.AddTag(la.flags.session)
			}
			select {
			case in <- a:
			case <-ctx.Done():
				coverageTester.WriteUniqueLine("parseDir - Branch 52/53 Covered")
				return
			}
		}
	}()

	gs := groups.NewGrouperPipeline(ctx, la.groupers...).PipeGrouper(ctx, in)
	for g := range gs {
		select {
		case gOut <- g:
		case <-ctx.Done():
			coverageTester.WriteUniqueLine("parseDir - Branch 53/53 Covered")
			return ctx.Err()
		}
	}
	return nil
}

func checkExistSideCar(fsys fs.FS, name string, ext string) (string, error) {
	ext2 := ""
	for _, r := range ext {
		if r == '.' {
			ext2 += "."
			continue
		}
		ext2 += "[" + strings.ToLower(string(r)) + strings.ToUpper(string(r)) + "]"
	}

	base := name
	l, err := fs.Glob(fsys, base+ext2)
	if err != nil {
		return "", err
	}
	if len(l) > 0 {
		return l[0], nil
	}

	ext = path.Ext(base)
	if !filetypes.DefaultSupportedMedia.IsMedia(ext) {
		return "", nil
	}
	base = strings.TrimSuffix(base, ext)

	l, err = fs.Glob(fsys, base+ext2)
	if err != nil {
		return "", err
	}
	if len(l) > 0 {
		return l[0], nil
	}
	return "", nil
}

func (la *LocalAssetBrowser) assetFromFile(_ context.Context, fsys fs.FS, name string) (*assets.Asset, error) {
	a := &assets.Asset{
		File:             fshelper.FSName(fsys, name),
		OriginalFileName: filepath.Base(name),
	}
	i, err := fs.Stat(fsys, name)
	if err != nil {
		a.Close()
		return nil, err
	}
	a.FileSize = int(i.Size())
	a.FileDate = i.ModTime()

	n := path.Dir(name) + "/" + a.OriginalFileName
	if fsys, ok := fsys.(interface{ Name() string }); ok {
		n = path.Join(fsys.Name(), n)
	}

	a.SetNameInfo(la.flags.InfoCollector.GetInfo(n))
	return a, nil
}
