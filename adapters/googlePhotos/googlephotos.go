package gp

import (
	"bytes"
	"context"
	"io/fs"
	"log/slog"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/filetypes"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/simulot/immich-go/internal/gen"
	"github.com/simulot/immich-go/internal/groups"
)

type fileKeyTracker struct {
	baseName string
	size     int64
}

type trackingInfo struct {
	paths    []string
	count    int
	metadata *assets.Metadata
	status   fileevent.Code
}

func trackerKeySortFunc(a, b fileKeyTracker) int {
	cmp := strings.Compare(a.baseName, b.baseName)
	if cmp != 0 {
		return cmp
	}
	return int(a.size) - int(b.size)
}

// directoryCatalog captures all files in a given directory
type directoryCatalog struct {
	jsons          map[string]*assets.Metadata // metadata in the catalog by base name
	unMatchedFiles map[string]*assetFile       // files to be matched map  by base name
	matchedFiles   map[string]*assets.Asset    // files matched by base name
}

// assetFile keep information collected during pass one
type assetFile struct {
	fsys   fs.FS            // Remember in which part of the archive the file is located
	base   string           // Remember the original file name
	length int              // file length in bytes
	date   time.Time        // file modification date
	md     *assets.Metadata // will point to the associated metadata
}

// Implement slog.LogValuer for assetFile
func (af assetFile) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("base", af.base),
		slog.Int("length", af.length),
		slog.Time("date", af.date),
	)
}

// Prepare scans all files in all walker to build the file catalog of the archive
// metadata files content is read and kept
// return a channel of asset groups after the puzzle is solved

func (toC *TakeoutCmd) Browse(ctx context.Context) chan *assets.Group {
	ctx, cancel := context.WithCancelCause(ctx)
	gOut := make(chan *assets.Group)
	go func() {
		defer close(gOut)

		for _, w := range toC.fsyss {
			err := toC.passOneFsWalk(ctx, w)
			if err != nil {
				cancel(err)
				return
			}
		}
		err := toC.solvePuzzle(ctx)
		if err != nil {
			cancel(err)
			return
		}
		err = toC.passTwo(ctx, gOut)
		cancel(err)
	}()
	return gOut
}

func (toC *TakeoutCmd) passOneFsWalk(ctx context.Context, w fs.FS) error {
	err := fs.WalkDir(w, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:

			if d.IsDir() {
				return nil
			}
			dir, base := path.Split(name)
			dir = strings.TrimSuffix(dir, "/")
			ext := strings.ToLower(path.Ext(base))

			// Exclude files to be ignored before processing
			if toC.BannedFiles.Match(name) {
				toC.log.Record(ctx, fileevent.DiscoveredDiscarded, fshelper.FSName(w, name), "reason", "banned file")
				return nil
			}

			if toC.SupportedMedia.IsUseLess(name) {
				toC.log.Record(ctx, fileevent.DiscoveredUseless, fshelper.FSName(w, name))
				return nil
			}

			if !toC.InclusionFlags.IncludedExtensions.Include(ext) {
				toC.log.Record(ctx, fileevent.DiscoveredDiscarded, fshelper.FSName(w, name), "reason", "file extension not selected")
				return nil
			}
			if toC.InclusionFlags.ExcludedExtensions.Exclude(ext) {
				toC.log.Record(ctx, fileevent.DiscoveredDiscarded, fshelper.FSName(w, name), "reason", "file extension not allowed")
				return nil
			}

			dirCatalog, ok := toC.catalogs[dir]
			if !ok {
				dirCatalog.jsons = map[string]*assets.Metadata{}
				dirCatalog.unMatchedFiles = map[string]*assetFile{}
				dirCatalog.matchedFiles = map[string]*assets.Asset{}
			}
			finfo, err := d.Info()
			if err != nil {
				toC.log.Record(ctx, fileevent.Error, fshelper.FSName(w, name), "error", err.Error())
				return err
			}
			switch ext {
			case ".json":
				var md *assets.Metadata
				b, err := fs.ReadFile(w, name)
				if err != nil {
					toC.log.Record(ctx, fileevent.Error, fshelper.FSName(w, name), "error", err.Error())
					return nil
				}
				if bytes.Contains(b, []byte("immich-go version:")) {
					md, err = assets.UnMarshalMetadata(b)
					if err != nil {
						toC.log.Record(ctx, fileevent.DiscoveredUnsupported, fshelper.FSName(w, name), "reason", "unknown JSONfile")
					}
					md.FileName = base
					toC.log.Record(ctx, fileevent.DiscoveredSidecar, fshelper.FSName(w, name), "type", "immich-go metadata", "title", md.FileName)
					md.File = fshelper.FSName(w, name)
				} else {
					md, err := fshelper.UnmarshalJSON[GoogleMetaData](b)
					if err == nil {
						switch {
						case md.isAsset():
							md := md.AsMetadata(fshelper.FSName(w, name), toC.PeopleTag) // Keep metadata
							dirCatalog.jsons[base] = md
							toC.log.Log().Debug("Asset JSON", "metadata", md)
							toC.log.Record(ctx, fileevent.DiscoveredSidecar, fshelper.FSName(w, name), "type", "asset metadata", "title", md.FileName, "date", md.DateTaken)
						case md.isAlbum():
							toC.log.Log().Debug("Album JSON", "metadata", md)
							if !toC.KeepUntitled && md.Title == "" {
								toC.log.Record(ctx, fileevent.DiscoveredUnsupported, fshelper.FSName(w, name), "reason", "discard untitled album")
								return nil
							}
							a := toC.albums[dir]
							a.Title = md.Title
							if a.Title == "" {
								a.Title = filepath.Base(dir)
							}
							if e := md.Enrichments; e != nil {
								a.Description = e.Text
								a.Latitude = e.Latitude
								a.Longitude = e.Longitude
							}
							toC.albums[dir] = a
							toC.log.Record(ctx, fileevent.DiscoveredSidecar, fshelper.FSName(w, name), "type", "album metadata", "title", md.Title)
						default:
							toC.log.Record(ctx, fileevent.DiscoveredUnsupported, fshelper.FSName(w, name), "reason", "unknown JSONfile")
							return nil
						}
					} else {
						toC.log.Record(ctx, fileevent.DiscoveredUnsupported, fshelper.FSName(w, name), "reason", "unknown JSONfile")
						return nil
					}
				}
			default:

				t := toC.SupportedMedia.TypeFromExt(ext)
				switch t {
				case filetypes.TypeUseless:
					toC.log.Record(ctx, fileevent.DiscoveredUseless, fshelper.FSName(w, name), "reason", "useless file")
					return nil
				case filetypes.TypeUnknown:
					toC.log.Record(ctx, fileevent.DiscoveredUnsupported, fshelper.FSName(w, name), "reason", "unsupported file type")
					return nil
				case filetypes.TypeVideo:
					toC.log.Record(ctx, fileevent.DiscoveredVideo, fshelper.FSName(w, name))
					if strings.Contains(name, "Failed Videos") {
						toC.log.Record(ctx, fileevent.DiscoveredDiscarded, fshelper.FSName(w, name), "reason", "can't upload failed videos")
						return nil
					}
				case filetypes.TypeImage:
					toC.log.Record(ctx, fileevent.DiscoveredImage, fshelper.FSName(w, name))
				}

				key := fileKeyTracker{
					baseName: base,
					size:     finfo.Size(),
				}

				// TODO: remove debugging code
				tracking, _ := toC.fileTracker.Load(key) // tracking := to.fileTracker[key]
				tracking.paths = append(tracking.paths, dir)
				tracking.count++
				toC.fileTracker.Store(key, tracking) // to.fileTracker[key] = tracking

				if a, ok := dirCatalog.unMatchedFiles[base]; ok {
					toC.logMessage(ctx, fileevent.AnalysisLocalDuplicate, a, "duplicated in the directory")
					return nil
				}

				dirCatalog.unMatchedFiles[base] = &assetFile{
					fsys:   w,
					base:   base,
					length: int(finfo.Size()),
					date:   finfo.ModTime(),
				}
			}
			toC.catalogs[dir] = dirCatalog
			return nil
		}
	})
	return err
}

// solvePuzzle prepares metadata with information collected during pass one for each accepted files
//
// JSON files give important information about the relative photos / movies:
//   - The original name (useful when it as been truncated)
//   - The date of capture (useful when the files doesn't have this date)
//   - The GPS coordinates (will be useful in a future release)
//
// Each JSON is checked. JSON is duplicated in albums folder.
// --Associated files with the JSON can be found in the JSON's folder, or in the Year photos.--
// ++JSON and files are located in the same folder
///
// Once associated and sent to the main program, files are tagged for not been associated with an other one JSON.
// Association is done with the help of a set of matcher functions. Each one implement a rule
//
// 1 JSON can be associated with 1+ files that have a part of their name in common.
// -   the file is named after the JSON name
// -   the file name can be 1 UTF-16 char shorter (🤯) than the JSON name
// -   the file name is longer than 46 UTF-16 chars (🤯) is truncated. But the truncation can creates duplicates, then a number is added.
// -   if there are several files with same original name, the first instance kept as it is, the next has a sequence number.
//       File is renamed as IMG_1234(1).JPG and the JSON is renamed as IMG_1234.JPG(1).JSON
// -   of course those rules are likely to collide. They have to be applied from the most common to the least one.
// -   sometimes the file isn't in the same folder than the json... It can be found in Year's photos folder
//
// --The duplicates files (same name, same length in bytes) found in the local source are discarded before been presented to the immich server.
// ++ Duplicates are presented to the next layer to allow the album handling
//
// To solve the puzzle, each directory is checked with all matchers in the order of the most common to the least.

type matcherFn func(jsonName string, fileName string, sm filetypes.SupportedMedia) bool

// matchers is a list of matcherFn from the most likely to be used to the least one
var matchers = []struct {
	name string
	fn   matcherFn
}{
	{name: "matchFastTrack", fn: matchFastTrack},
	{name: "matchNormal", fn: matchNormal},
	{name: "matchForgottenDuplicates", fn: matchForgottenDuplicates},
	{name: "matchEditedName", fn: matchEditedName},
}

func (toC *TakeoutCmd) solvePuzzle(ctx context.Context) error {
	dirs := gen.MapKeysSorted(toC.catalogs)
	for _, dir := range dirs {
		cat := toC.catalogs[dir]
		jsons := gen.MapKeysSorted(cat.jsons)
		for _, matcher := range matchers {
			for _, json := range jsons {
				md := cat.jsons[json]
				for f := range cat.unMatchedFiles {
					select {
					case <-ctx.Done():
						return ctx.Err()
					default:
						if matcher.fn(json, f, toC.SupportedMedia) {
							i := cat.unMatchedFiles[f]
							i.md = md
							a := toC.makeAsset(ctx, dir, i, md)
							cat.matchedFiles[f] = a
							toC.log.Record(ctx, fileevent.AnalysisAssociatedMetadata, fshelper.FSName(i.fsys, path.Join(dir, i.base)), "json", json, "matcher", matcher.name)
							delete(cat.unMatchedFiles, f)
						}
					}
				}
			}
		}
		toC.catalogs[dir] = cat
		if len(cat.unMatchedFiles) > 0 {
			files := gen.MapKeys(cat.unMatchedFiles)
			sort.Strings(files)
			for _, f := range files {
				i := cat.unMatchedFiles[f]
				toC.log.Record(ctx, fileevent.AnalysisMissingAssociatedMetadata, fshelper.FSName(i.fsys, path.Join(dir, i.base)))
				if toC.KeepJSONLess {
					a := toC.makeAsset(ctx, dir, i, nil)
					cat.matchedFiles[f] = a
					delete(cat.unMatchedFiles, f)
				}
			}
		}
	}
	return nil
}

// Browse return a channel of assets
// Each asset is a group of files that are associated with each other

func (toC *TakeoutCmd) passTwo(ctx context.Context, gOut chan *assets.Group) error {
	dirs := gen.MapKeys(toC.catalogs)
	sort.Strings(dirs)
	for _, dir := range dirs {
		if len(toC.catalogs[dir].matchedFiles) > 0 {
			err := toC.handleDir(ctx, dir, gOut)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// type linkedFiles struct {
// 	dir   string
// 	base  string
// 	video *assetFile
// 	image *assetFile
// }

func (toC *TakeoutCmd) handleDir(ctx context.Context, dir string, gOut chan *assets.Group) error {
	catalog := toC.catalogs[dir]

	dirEntries := make([]*assets.Asset, 0, len(catalog.matchedFiles))

	for name := range catalog.matchedFiles {
		a := catalog.matchedFiles[name]
		key := fileKeyTracker{baseName: name, size: int64(a.FileSize)}
		track, _ := toC.fileTracker.Load(key) // track := to.fileTracker[key]
		if track.status == fileevent.Uploaded {
			a.Close()
			toC.logMessage(ctx, fileevent.AnalysisLocalDuplicate, a.File, "local duplicate")
			continue
		}

		// Filter on metadata
		if code := toC.filterOnMetadata(ctx, a); code != fileevent.Code(0) {
			a.Close()
			continue
		}
		dirEntries = append(dirEntries, a)
	}

	in := make(chan *assets.Asset)
	go func() {
		defer close(in)

		sort.Slice(dirEntries, func(i, j int) bool {
			// Sort by radical first
			radicalI := dirEntries[i].Radical
			radicalJ := dirEntries[j].Radical
			if radicalI != radicalJ {
				return radicalI < radicalJ
			}
			// If radicals are the same, sort by date
			return dirEntries[i].CaptureDate.Before(dirEntries[j].CaptureDate)
		})

		for _, a := range dirEntries {
			if toC.CreateAlbums {
				if toC.ImportIntoAlbum != "" {
					// Force this album
					a.Albums = []assets.Album{{Title: toC.ImportIntoAlbum}}
				} else {
					// check if its duplicates are in some albums, and push them all at once
					key := fileKeyTracker{baseName: filepath.Base(a.File.Name()), size: int64(a.FileSize)}
					track, _ := toC.fileTracker.Load(key) // track := to.fileTracker[key]
					for _, p := range track.paths {
						if album, ok := toC.albums[p]; ok {
							title := album.Title
							if title == "" {
								if !toC.KeepUntitled {
									continue
								}
								title = filepath.Base(p)
							}
							a.Albums = append(a.Albums, assets.Album{
								Title:       title,
								Description: album.Description,
								Latitude:    album.Latitude,
								Longitude:   album.Longitude,
							})
						}
					}
				}

				// Force this album for partners photos
				if toC.PartnerSharedAlbum != "" && a.FromPartner {
					a.Albums = append(a.Albums, assets.Album{Title: toC.PartnerSharedAlbum})
				}
				if a.FromApplication != nil {
					a.FromApplication.Albums = a.Albums
				}
			}
			// If the asset has no GPS information, but the album has, use the album's location
			if a.Latitude == 0 && a.Longitude == 0 {
				for _, album := range a.Albums {
					if album.Latitude != 0 || album.Longitude != 0 {
						// when there isn't GPS information on the photo, but the album has a location,  use that location
						a.Latitude = album.Latitude
						a.Longitude = album.Longitude
						break
					}
				}
			}

			if toC.TakeoutTag {
				a.AddTag(toC.TakeoutName)
			}

			select {
			case in <- a:
			case <-ctx.Done():
				return
			}
		}
	}()

	gs := groups.NewGrouperPipeline(ctx, toC.groupers...).PipeGrouper(ctx, in)
	for g := range gs {
		select {
		case gOut <- g:
			for _, a := range g.Assets {
				key := fileKeyTracker{
					baseName: path.Base(a.File.Name()),
					size:     int64(a.FileSize),
				}
				track, _ := toC.fileTracker.Load(key) // track := to.fileTracker[key]
				track.status = fileevent.Uploaded
				toC.fileTracker.Store(key, track) // to.fileTracker[key] = track
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

// makeAsset makes a localAssetFile based on the google metadata
func (toC *TakeoutCmd) makeAsset(_ context.Context, dir string, f *assetFile, md *assets.Metadata) *assets.Asset {
	file := path.Join(dir, f.base)
	a := &assets.Asset{
		File:             fshelper.FSName(f.fsys, file), // File as named in the archive
		FileSize:         f.length,
		OriginalFileName: f.base,
		FileDate:         f.date,
	}

	// get the original file name from metadata
	if md != nil && md.FileName != "" {
		title := md.FileName

		// a.OriginalFileName = md.FileName
		// title := md.FileName

		// trim superfluous extensions
		titleExt := path.Ext(title)
		fileExt := path.Ext(file)

		if titleExt != fileExt {
			title = strings.TrimSuffix(title, titleExt)
			titleExt = path.Ext(title)
			if titleExt != fileExt {
				title = strings.TrimSuffix(title, titleExt) + fileExt
			}
		}
		a.FromApplication = a.UseMetadata(md)
		a.OriginalFileName = title
	}
	a.SetNameInfo(toC.InfoCollector.GetInfo(a.OriginalFileName))
	return a
}

func (toC *TakeoutCmd) filterOnMetadata(ctx context.Context, a *assets.Asset) fileevent.Code {
	if !toC.KeepArchived && a.Visibility == assets.VisibilityArchive {
		toC.logMessage(ctx, fileevent.DiscoveredDiscarded, a, "discarding archived file")
		a.Close()
		return fileevent.DiscoveredDiscarded
	}
	if !toC.KeepPartner && a.FromPartner {
		toC.logMessage(ctx, fileevent.DiscoveredDiscarded, a, "discarding partner file")
		a.Close()
		return fileevent.DiscoveredDiscarded
	}
	if !toC.KeepTrashed && a.Trashed {
		toC.logMessage(ctx, fileevent.DiscoveredDiscarded, a, "discarding trashed file")
		a.Close()
		return fileevent.DiscoveredDiscarded
	}

	if toC.InclusionFlags.DateRange.IsSet() && !toC.InclusionFlags.DateRange.InRange(a.CaptureDate) {
		toC.logMessage(ctx, fileevent.DiscoveredDiscarded, a, "discarding files out of date range")
		a.Close()
		return fileevent.DiscoveredDiscarded
	}
	if toC.ImportFromAlbum != "" {
		keep := false
		dir := path.Dir(a.File.Name())
		if dir == "." {
			dir = ""
		}
		if album, ok := toC.albums[dir]; ok {
			keep = keep || album.Title == toC.ImportFromAlbum
		}
		if !keep {
			toC.logMessage(ctx, fileevent.DiscoveredDiscarded, a, "discarding files not in the specified album")
			a.Close()
			return fileevent.DiscoveredDiscarded
		}
	}
	return fileevent.Code(0)
}
