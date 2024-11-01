package gp

import (
	"context"
	"io/fs"
	"log/slog"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/simulot/immich-go/helpers/gen"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/simulot/immich-go/internal/groups"
	"github.com/simulot/immich-go/internal/groups/burst"
	"github.com/simulot/immich-go/internal/groups/series"
	"github.com/simulot/immich-go/internal/metadata"
)

type Takeout struct {
	fsyss       []fs.FS
	catalogs    map[string]directoryCatalog     // file catalogs by directory in the set of the all takeout parts
	albums      map[string]assets.Album         // track album names by folder
	fileTracker map[fileKeyTracker]trackingInfo // key is base name + file size,  value is list of file paths
	// debugLinkedFiles []linkedFiles
	log      *fileevent.Recorder
	flags    *ImportFlags // command-line flags
	exiftool *metadata.ExifTool
}

type fileKeyTracker struct {
	baseName string
	size     int64
}

type trackingInfo struct {
	paths    []string
	count    int
	metadata *metadata.Metadata
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
	jsons          map[string]*metadata.Metadata // metadata in the catalog by base name
	unMatchedFiles map[string]*assetFile         // files to be matched map  by base name
	matchedFiles   map[string]*assets.Asset      // files matched by base name
}

// assetFile keep information collected during pass one
type assetFile struct {
	fsys   fs.FS              // Remember in which part of the archive the file is located
	base   string             // Remember the original file name
	length int                // file length in bytes
	date   time.Time          // file modification date
	md     *metadata.Metadata // will point to the associated metadata
}

// Implement slog.LogValuer for assetFile
func (af assetFile) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("base", af.base),
		slog.Int("length", af.length),
		slog.Time("date", af.date),
	)
}

func NewTakeout(ctx context.Context, l *fileevent.Recorder, flags *ImportFlags, fsyss ...fs.FS) (*Takeout, error) {
	to := Takeout{
		fsyss:       fsyss,
		catalogs:    map[string]directoryCatalog{},
		albums:      map[string]assets.Album{},
		fileTracker: map[fileKeyTracker]trackingInfo{},
		log:         l,
		flags:       flags,
	}
	if flags.InfoCollector == nil {
		flags.InfoCollector = filenames.NewInfoCollector(flags.DateHandlingFlags.FilenameTimeZone.Location(), flags.SupportedMedia)
	}
	if flags.ExifToolFlags.UseExifTool {
		et, err := metadata.NewExifTool(&flags.ExifToolFlags)
		if err != nil {
			return nil, err
		}
		to.exiftool = et
	}

	return &to, nil
}

// Prepare scans all files in all walker to build the file catalog of the archive
// metadata files content is read and kept
// return a channel of asset groups after the puzzle is solved

func (to *Takeout) Browse(ctx context.Context) chan *assets.Group {
	ctx, cancel := context.WithCancelCause(ctx)
	gOut := make(chan *assets.Group)
	go func() {
		defer close(gOut)

		for _, w := range to.fsyss {
			err := to.passOneFsWalk(ctx, w)
			if err != nil {
				cancel(err)
				return
			}
		}
		err := to.solvePuzzle(ctx)
		if err != nil {
			cancel(err)
			return
		}
		err = to.passTwo(ctx, gOut)
		cancel(err)
	}()
	return gOut
}

func (to *Takeout) passOneFsWalk(ctx context.Context, w fs.FS) error {
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

			dirCatalog, ok := to.catalogs[dir]
			if !ok {
				dirCatalog.jsons = map[string]*metadata.Metadata{}
				dirCatalog.unMatchedFiles = map[string]*assetFile{}
				dirCatalog.matchedFiles = map[string]*assets.Asset{}
			}
			finfo, err := d.Info()
			if err != nil {
				to.log.Record(ctx, fileevent.Error, fileevent.AsFileAndName(w, name), "error", err.Error())
				return err
			}
			switch ext {
			case ".json":
				md, err := fshelper.ReadJSON[GoogleMetaData](w, name)
				if err == nil {
					switch {
					case md.isAsset():
						dirCatalog.jsons[base] = md.AsMetadata() // Keep metadata
						to.log.Log().Debug("Asset JSON", "metadata", md)
						to.log.Record(ctx, fileevent.DiscoveredSidecar, fileevent.AsFileAndName(w, name), "type", "asset metadata", "title", md.Title)
					case md.isAlbum():
						to.log.Log().Debug("Album JSON", "metadata", md)
						if !to.flags.KeepUntitled && md.Title == "" {
							to.log.Record(ctx, fileevent.DiscoveredUnsupported, fileevent.AsFileAndName(w, name), "reason", "discard untitled album")
							return nil
						}
						a := to.albums[dir]
						a.Title = md.Title
						if a.Title == "" {
							a.Title = filepath.Base(dir)
						}
						if e := md.Enrichments; e != nil {
							a.Description = e.Text
							a.Latitude = e.Latitude
							a.Longitude = e.Longitude
						}
						to.albums[dir] = a
						to.log.Record(ctx, fileevent.DiscoveredSidecar, fileevent.AsFileAndName(w, name), "type", "album metadata", "title", md.Title)
					default:
						to.log.Record(ctx, fileevent.DiscoveredUnsupported, fileevent.AsFileAndName(w, name), "reason", "unknown JSONfile")
						return nil
					}
				} else {
					to.log.Record(ctx, fileevent.DiscoveredUnsupported, fileevent.AsFileAndName(w, name), "reason", "unknown JSONfile")
					return nil
				}
			default:

				if to.flags.BannedFiles.Match(name) {
					to.log.Record(ctx, fileevent.DiscoveredDiscarded, fileevent.AsFileAndName(w, name), "reason", "banned file")
					return nil
				}

				if !to.flags.InclusionFlags.IncludedExtensions.Include(ext) {
					to.log.Record(ctx, fileevent.DiscoveredDiscarded, fileevent.AsFileAndName(w, name), "reason", "file extension not selected")
					return nil
				}
				if to.flags.InclusionFlags.ExcludedExtensions.Exclude(ext) {
					to.log.Record(ctx, fileevent.DiscoveredDiscarded, fileevent.AsFileAndName(w, name), "reason", "file extension not allowed")
					return nil
				}
				t := to.flags.SupportedMedia.TypeFromExt(ext)
				switch t {
				case metadata.TypeUnknown:
					to.log.Record(ctx, fileevent.DiscoveredUnsupported, fileevent.AsFileAndName(w, name), "reason", "unsupported file type")
					return nil
				case metadata.TypeVideo:
					to.log.Record(ctx, fileevent.DiscoveredVideo, fileevent.AsFileAndName(w, name))
					if strings.Contains(name, "Failed Videos") {
						to.log.Record(ctx, fileevent.DiscoveredDiscarded, fileevent.AsFileAndName(w, name), "reason", "can't upload failed videos")
						return nil
					}
				case metadata.TypeImage:
					to.log.Record(ctx, fileevent.DiscoveredImage, fileevent.AsFileAndName(w, name))
				}

				key := fileKeyTracker{
					baseName: base,
					size:     finfo.Size(),
				}

				tracking := to.fileTracker[key]
				tracking.paths = append(tracking.paths, dir)
				tracking.count++
				to.fileTracker[key] = tracking

				if a, ok := dirCatalog.unMatchedFiles[base]; ok {
					to.logMessage(ctx, fileevent.AnalysisLocalDuplicate, a, "duplicated in the directory")
					return nil
				}

				dirCatalog.unMatchedFiles[base] = &assetFile{
					fsys:   w,
					base:   base,
					length: int(finfo.Size()),
					date:   finfo.ModTime(),
				}
			}
			to.catalogs[dir] = dirCatalog
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

type matcherFn func(jsonName string, fileName string, sm metadata.SupportedMedia) bool

// matchers is a list of matcherFn from the most likely to be used to the least one
var matchers = []struct {
	name string
	fn   matcherFn
}{
	{name: "normalMatch", fn: normalMatch},
	{name: "livePhotoMatch", fn: livePhotoMatch},
	{name: "matchWithOneCharOmitted", fn: matchWithOneCharOmitted},
	{name: "matchVeryLongNameWithNumber", fn: matchVeryLongNameWithNumber},
	{name: "matchDuplicateInYear", fn: matchDuplicateInYear},
	{name: "matchEditedName", fn: matchEditedName},
	{name: "matchForgottenDuplicates", fn: matchForgottenDuplicates},
}

func (to *Takeout) solvePuzzle(ctx context.Context) error {
	dirs := gen.MapKeysSorted(to.catalogs)
	for _, dir := range dirs {
		cat := to.catalogs[dir]
		jsons := gen.MapKeysSorted(cat.jsons)
		for _, matcher := range matchers {
			for _, json := range jsons {
				md := cat.jsons[json]
				for f := range cat.unMatchedFiles {
					select {
					case <-ctx.Done():
						return ctx.Err()
					default:
						if matcher.fn(json, f, to.flags.SupportedMedia) {
							i := cat.unMatchedFiles[f]
							i.md = md
							a := to.makeAsset(ctx, dir, i, md)
							cat.matchedFiles[f] = a
							to.log.Record(ctx, fileevent.AnalysisAssociatedMetadata, fileevent.AsFileAndName(i.fsys, path.Join(dir, i.base)), "json", json, "matcher", matcher.name)
							delete(cat.unMatchedFiles, f)
						}
					}
				}
			}
		}
		to.catalogs[dir] = cat
		if len(cat.unMatchedFiles) > 0 {
			files := gen.MapKeys(cat.unMatchedFiles)
			sort.Strings(files)
			for _, f := range files {
				i := cat.unMatchedFiles[f]
				to.log.Record(ctx, fileevent.AnalysisMissingAssociatedMetadata, fileevent.AsFileAndName(i.fsys, path.Join(dir, i.base)))
				if to.flags.KeepJSONLess {
					a := to.makeAsset(ctx, dir, i, nil)
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

func (to *Takeout) passTwo(ctx context.Context, gOut chan *assets.Group) error {
	dirs := gen.MapKeys(to.catalogs)
	sort.Strings(dirs)
	for _, dir := range dirs {
		if len(to.catalogs[dir].matchedFiles) > 0 {
			err := to.handleDir(ctx, dir, gOut)
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

func (to *Takeout) handleDir(ctx context.Context, dir string, gOut chan *assets.Group) error {
	catalog := to.catalogs[dir]

	dirEntries := make([]*assets.Asset, 0, len(catalog.matchedFiles))

	for name := range catalog.matchedFiles {
		a := catalog.matchedFiles[name]
		key := fileKeyTracker{baseName: name, size: int64(a.FileSize)}
		track := to.fileTracker[key]
		if track.status == fileevent.Uploaded {
			a.Close()
			to.logMessage(ctx, fileevent.AnalysisLocalDuplicate, fileevent.AsFileAndName(a.FSys, path.Join(dir, a.FileName)), "local duplicate")
			continue
		}

		// Filter on metadata
		if code := to.filterOnMetadata(ctx, a); code != fileevent.Code(0) {
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
			radicalI := dirEntries[i].NameInfo().Radical
			radicalJ := dirEntries[j].NameInfo().Radical
			if radicalI != radicalJ {
				return radicalI < radicalJ
			}
			// If radicals are the same, sort by date
			return dirEntries[i].CaptureDate.Before(dirEntries[j].CaptureDate)
		})

		for _, a := range dirEntries {
			select {
			case in <- a:
			case <-ctx.Done():
				return
			}
		}
	}()

	gs := groups.NewGrouperPipeline(ctx, burst.Group, series.Group).PipeGrouper(ctx, in)
	for g := range gs {
		// Manage albums
		for _, a := range g.Assets {
			if to.flags.CreateAlbums {
				if to.flags.ImportIntoAlbum != "" {
					// Force this album
					g.Albums = []assets.Album{{Title: to.flags.ImportIntoAlbum}}
				} else {
					// check if its duplicates are in some albums, and push them all at once
					key := fileKeyTracker{baseName: filepath.Base(a.FileName), size: int64(a.FileSize)}
					track := to.fileTracker[key]
					for _, p := range track.paths {
						if album, ok := to.albums[p]; ok {
							title := album.Title
							if title == "" {
								if !to.flags.KeepUntitled {
									continue
								}
								title = filepath.Base(p)
							}
							g.AddAlbum(assets.Album{
								Title:       title,
								Description: album.Description,
								Latitude:    album.Latitude,
								Longitude:   album.Longitude,
							})
						}
					}
				}

				// Force this album for partners photos
				if to.flags.PartnerSharedAlbum != "" && a.FromPartner {
					g.Albums = append(g.Albums, assets.Album{Title: to.flags.PartnerSharedAlbum})
				}
			}
			// If the asset has no GPS information, but the album has, use the album's location
			if a.Latitude == 0 && a.Longitude == 0 {
				for _, album := range g.Albums {
					if album.Latitude != 0 || album.Longitude != 0 {
						// when there isn't GPS information on the photo, but the album has a location,  use that location
						a.Latitude = album.Latitude
						a.Longitude = album.Longitude
						break
					}
				}
			}
		}

		select {
		case gOut <- g:
			for _, a := range g.Assets {
				key := fileKeyTracker{
					baseName: path.Base(a.FileName),
					size:     int64(a.FileSize),
				}
				track := to.fileTracker[key]
				track.status = fileevent.Uploaded
				to.fileTracker[key] = track
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

// makeAsset makes a localAssetFile based on the google metadata
func (to *Takeout) makeAsset(_ context.Context, dir string, f *assetFile, md *metadata.Metadata) *assets.Asset {
	file := filepath.Join(dir, f.base)
	a := &assets.Asset{
		FileName: file, // File as named in the archive
		FileSize: f.length,
		Title:    f.base,
		FSys:     f.fsys,
		FileDate: f.date,
	}

	// get the original file name from metadata
	if md != nil && md.FileName != "" {
		title := md.FileName

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
		// md.FileName = title
		a.Title = title
		a.CaptureDate = md.DateTaken
		a.Archived = md.Archived
		a.Favorite = md.Favorited
		a.Trashed = md.Trashed
		a.FromPartner = md.FromPartner
		a.Latitude = md.Latitude
		a.Longitude = md.Longitude
	}
	a.SetNameInfo(to.flags.InfoCollector.GetInfo(a.Title))
	if a.CaptureDate.IsZero() {
		a.CaptureDate = a.NameInfo().Taken
	}
	return a
}

func (to *Takeout) filterOnMetadata(ctx context.Context, a *assets.Asset) fileevent.Code {
	if !to.flags.KeepArchived && a.Archived {
		to.logMessage(ctx, fileevent.DiscoveredDiscarded, a, "discarding archived file")
		a.Close()
		return fileevent.DiscoveredDiscarded
	}
	if !to.flags.KeepPartner && a.FromPartner {
		to.logMessage(ctx, fileevent.DiscoveredDiscarded, a, "discarding partner file")
		a.Close()
		return fileevent.DiscoveredDiscarded
	}
	if !to.flags.KeepTrashed && a.Trashed {
		to.logMessage(ctx, fileevent.DiscoveredDiscarded, a, "discarding trashed file")
		a.Close()
		return fileevent.DiscoveredDiscarded
	}
	if a.DateTaken().IsZero() {
		// wasn't associated
		md, err := a.ReadMetadata(to.flags.DateHandlingFlags.Method, assets.ReadMetadataOptions{
			ExifTool:         to.exiftool,
			ExiftoolTimezone: to.flags.ExifToolFlags.Timezone.Location(),
			FilenameTimeZone: to.flags.DateHandlingFlags.FilenameTimeZone.Location(),
		})
		if err != nil {
			to.logMessage(ctx, fileevent.Error, a, err.Error())
			a.Close()
			return fileevent.Error
		}
		if md != nil {
			a.CaptureDate = md.DateTaken
		}
	}
	if to.flags.InclusionFlags.DateRange.IsSet() && !to.flags.InclusionFlags.DateRange.InRange(a.CaptureDate) {
		to.logMessage(ctx, fileevent.DiscoveredDiscarded, a, "discarding files out of date range")
		a.Close()
		return fileevent.DiscoveredDiscarded
	}
	if to.flags.ImportFromAlbum != "" {
		keep := false
		dir := path.Dir(a.FileName)
		if album, ok := to.albums[dir]; ok {
			keep = keep || album.Title == to.flags.ImportFromAlbum
		}
		if !keep {
			to.logMessage(ctx, fileevent.DiscoveredDiscarded, a, "discarding files not in the specified album")
			a.Close()
			return fileevent.DiscoveredDiscarded
		}
	}
	return fileevent.Code(0)
}
