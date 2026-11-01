package gp

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/filetypes"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/simulot/immich-go/internal/fshelper/hash"
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

// fileKey names a file or a sidecar in a directory catalog.
//
// Google Takeout splits a large export into series of zip parts
// (takeout-<date>-1-001.zip, ..., takeout-<date>-2-001.zip, ...) and numbers name clashes
// "(1)", "(2)"... separately in each series. The same name can therefore belong to two
// different files, one per series, each with a sidecar of the same name in its own series.
type fileKey struct {
	series string // see takeoutSeries
	base   string
}

// _reTakeoutPart matches the name fshelper gives a takeout zip part: takeout-<date>[-<series>]-<part>
var _reTakeoutPart = regexp.MustCompile(`^takeout-\d{8}T\d{6}Z(-\d+)?-\d{3}$`)

// takeoutSeries returns the series of a takeout part: its name without the part number.
// "takeout-20251016T092000Z-2-001" gives "takeout-20251016T092000Z-2". Any other name gives "":
// such parts share one series, and their files are matched together as before.
func takeoutSeries(fsys fs.FS) string {
	if n, ok := fsys.(fshelper.NameFS); ok {
		if name := path.Base(n.Name()); _reTakeoutPart.MatchString(name) {
			return _re3digits.ReplaceAllString(name, "")
		}
	}
	return ""
}

func sortedFileKeys[V any](m map[fileKey]V) []fileKey {
	keys := gen.MapKeys(m)
	slices.SortFunc(keys, func(a, b fileKey) int {
		if c := strings.Compare(a.base, b.base); c != 0 {
			return c
		}
		return strings.Compare(a.series, b.series)
	})
	return keys
}

// directoryCatalog captures all files in a given directory
type directoryCatalog struct {
	jsons          map[fileKey]*assets.Metadata // metadata in the catalog by series and base name
	unMatchedFiles map[fileKey]*assetFile       // files to be matched by series and base name
	matchedFiles   map[fileKey]*assets.Asset    // files matched by series and base name
	series         []string                     // series of the files found in the directory during pass one
}

// seriesPerName counts, for each name of a file or a sidecar, the series of the directory that hold it.
func (cat directoryCatalog) seriesPerName() map[string]int {
	n := map[string]int{}
	for k := range cat.jsons {
		n[k.base]++
	}
	for k := range cat.unMatchedFiles {
		n[k.base]++
	}
	return n
}

// borrowable tells whether the sidecar j can describe the file f of another series. Takeout numbers name
// clashes "(1)", "(2)"... within a series, so an indexed name means nothing in another series, and a name
// found in several series is ambiguous.
func borrowable(j, f fileKey, seriesPerName map[string]int) bool {
	if _, index := getFileIndex(j.base); index != "" {
		return false
	}
	if _, index := getFileIndex(f.base); index != "" {
		return false
	}
	return seriesPerName[j.base] == 1 && seriesPerName[f.base] == 1
}

// holdsIdenticalFile tells whether the directory dir already holds a file with the name, the size and the content
// of af. Contents are read only for files of the same name and size, and kept on the files so that each is read once.
// A file that can't be read is identical to none: it is kept, which is the safe side.
func (cat directoryCatalog) holdsIdenticalFile(dir string, af *assetFile) bool {
	sum := func(f *assetFile) []byte { // nil if the file can't be read, then tried again next time
		if f.sum == nil {
			f.sum, _ = hash.FileSHA1Hash(f.fsys, path.Join(dir, f.base))
		}
		return f.sum
	}
	for _, s := range cat.series {
		other, ok := cat.unMatchedFiles[fileKey{series: s, base: af.base}]
		if !ok || other.length != af.length {
			continue
		}
		if h := sum(other); h != nil && bytes.Equal(h, sum(af)) {
			return true
		}
	}
	return false
}

// holdsSeveralSeries tells whether the directory holds files or sidecars of more than one series.
func (cat directoryCatalog) holdsSeveralSeries() bool {
	series := map[string]bool{}
	for k := range cat.jsons {
		series[k.series] = true
	}
	for k := range cat.unMatchedFiles {
		series[k.series] = true
	}
	return len(series) > 1
}

// assetFile keep information collected during pass one
type assetFile struct {
	fsys    fs.FS            // Remember in which part of the archive the file is located
	base    string           // Remember the original file name
	length  int              // file length in bytes
	date    time.Time        // file modification date
	md      *assets.Metadata // will point to the associated metadata
	matcher string           // name of the matcher that associated the file with its metadata
	sum     []byte           // SHA1 of the content, read by holdsIdenticalFile when a file of the same name and size shows up
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

func (toc *TakeoutCmd) Browse(ctx context.Context) chan *assets.Group {
	ctx, cancel := context.WithCancelCause(ctx)
	gOut := make(chan *assets.Group)
	go func() {
		defer close(gOut)

		for _, w := range toc.fsyss {
			err := toc.passOneFsWalk(ctx, w)
			if err != nil {
				cancel(err)
				return
			}
		}
		err := toc.solvePuzzle(ctx)
		if err != nil {
			cancel(err)
			return
		}
		toc.findNamesakes()
		err = toc.passTwo(ctx, gOut)
		cancel(err)
	}()
	return gOut
}

func (toc *TakeoutCmd) passOneFsWalk(ctx context.Context, w fs.FS) error {
	series := takeoutSeries(w)
	err := fs.WalkDir(w, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				// This branch can happen when you have symbolic links.
				return nil
			}

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
			finfo, err := fs.Stat(w, name)
			if err != nil {
				toc.processor.RecordNonAsset(ctx, fshelper.FSName(w, name), 0, fileevent.ErrorFileAccess, "error", err.Error())
				return nil
			}

			// Exclude files to be ignored before processing
			if toc.BannedFiles.Match(name) {
				toc.processor.RecordNonAsset(ctx, fshelper.FSName(w, name), 0, fileevent.DiscoveredBanned, "reason", "banned file")
				return nil
			}

			if toc.supportedMedia.IsUseLess(name) {
				toc.processor.RecordNonAsset(ctx, fshelper.FSName(w, name), 0, fileevent.DiscoveredUnknown, "reason", "useless file")
				return nil
			}

			if !toc.InclusionFlags.IncludedExtensions.Include(ext) {
				toc.processor.RecordAssetDiscardedImmediately(ctx, fshelper.FSName(w, name), finfo.Size(), fileevent.DiscardedFiltered, "extension not included")

				return nil
			}
			if toc.InclusionFlags.ExcludedExtensions.Exclude(ext) {
				toc.processor.RecordAssetDiscardedImmediately(ctx, fshelper.FSName(w, name), finfo.Size(), fileevent.DiscardedFiltered, "extension excluded")

				return nil
			}

			dirCatalog, ok := toc.catalogs[dir]
			if !ok {
				dirCatalog.jsons = map[fileKey]*assets.Metadata{}
				dirCatalog.unMatchedFiles = map[fileKey]*assetFile{}
				dirCatalog.matchedFiles = map[fileKey]*assets.Asset{}
			}
			switch ext {
			case ".json":
				var md *assets.Metadata
				b, err := fs.ReadFile(w, name)
				if err != nil {
					err = toc.app.ProcessError(err)
					return err
				}
				if bytes.Contains(b, []byte("immich-go version:")) {
					md, err = assets.UnMarshalMetadata(b)
					if err != nil {
						toc.processor.RecordNonAsset(ctx, fshelper.FSName(w, name), int64(len(b)), fileevent.DiscoveredUnsupported, "reason", "unknown JSONfile")
						return nil
					}
					md.FileName = base
					toc.processor.RecordNonAsset(ctx, fshelper.FSName(w, name), int64(len(b)), fileevent.DiscoveredSidecar, "type", "immich-go metadata", "title", md.FileName)
					md.File = fshelper.FSName(w, name)
				} else {
					md, err := fshelper.UnmarshalJSON[GoogleMetaData](b)
					if err == nil {
						switch {
						case md.isAsset():
							md := md.AsMetadata(fshelper.FSName(w, name), toc.PeopleTag) // Keep metadata
							dirCatalog.jsons[fileKey{series: series, base: base}] = md
							toc.app.Log().Debug("Asset JSON", "metadata", md)
							toc.processor.RecordNonAsset(ctx, fshelper.FSName(w, name), int64(len(b)), fileevent.DiscoveredSidecar, "type", "asset metadata", "title", md.FileName, "date", md.DateTaken)
						case md.isAlbum():
							toc.app.Log().Debug("Album JSON", "metadata", md)
							if !toc.KeepUntitled && md.Title == "" {
								toc.processor.RecordNonAsset(ctx, fshelper.FSName(w, name), int64(len(b)), fileevent.DiscoveredUnsupported, "reason", "discard untitled album")
								return nil
							}
							a := toc.albums[dir]
							a.Title = md.Title
							if a.Title == "" {
								a.Title = filepath.Base(dir)
							}
							if e := md.Enrichments; e != nil {
								a.Description = e.Text
								a.Latitude = e.Latitude
								a.Longitude = e.Longitude
							}
							toc.albums[dir] = a
							toc.processor.RecordNonAsset(ctx, fshelper.FSName(w, name), int64(len(b)), fileevent.DiscoveredSidecar, "type", "album metadata", "title", md.Title)
						default:
							toc.processor.RecordNonAsset(ctx, fshelper.FSName(w, name), int64(len(b)), fileevent.DiscoveredUnsupported, "reason", "unknown JSONfile")
							return nil
						}
					} else {
						toc.processor.RecordNonAsset(ctx, fshelper.FSName(w, name), int64(len(b)), fileevent.DiscoveredUnsupported, "reason", "unknown JSONfile")
						return nil
					}
				}
			default:

				t := toc.supportedMedia.TypeFromExt(ext)
				switch t {
				case filetypes.TypeUseless:
					toc.processor.RecordNonAsset(ctx, fshelper.FSName(w, name), finfo.Size(), fileevent.DiscoveredUnknown, "reason", "useless file")
					return nil
				case filetypes.TypeUnknown:
					toc.processor.RecordNonAsset(ctx, fshelper.FSName(w, name), finfo.Size(), fileevent.DiscoveredUnsupported, "reason", "unsupported file type")
					return nil
				case filetypes.TypeVideo:
					if strings.Contains(name, "Failed Videos") {
						toc.processor.RecordAssetDiscardedImmediately(ctx, fshelper.FSName(w, name), finfo.Size(), fileevent.DiscardedFiltered, "can't upload failed videos")
						return nil
					} else {
						toc.processor.RecordAssetDiscovered(ctx, fshelper.FSName(w, name), finfo.Size(), fileevent.DiscoveredVideo)
					}
				case filetypes.TypeImage:
					toc.processor.RecordAssetDiscovered(ctx, fshelper.FSName(w, name), finfo.Size(), fileevent.DiscoveredImage)
				}

				key := fileKeyTracker{
					baseName: base,
					size:     finfo.Size(),
				}

				// TODO: remove debugging code
				tracking, _ := toc.fileTracker.Load(key) // tracking := to.fileTracker[key]
				tracking.paths = append(tracking.paths, dir)
				tracking.count++
				toc.fileTracker.Store(key, tracking) // to.fileTracker[key] = tracking

				af := &assetFile{
					fsys:   w,
					base:   base,
					length: int(finfo.Size()),
					date:   finfo.ModTime(),
				}

				// A file of the same name in this series, or the same file from another series, is a duplicate.
				// The same name and size alone is not proof: the content is compared.
				if _, ok := dirCatalog.unMatchedFiles[fileKey{series: series, base: base}]; ok || dirCatalog.holdsIdenticalFile(dir, af) {
					toc.processor.RecordAssetDiscardedImmediately(ctx, fshelper.FSName(w, name), finfo.Size(), fileevent.DiscardedLocalDuplicate, "duplicated in the directory")
					return nil
				}

				if !slices.Contains(dirCatalog.series, series) {
					dirCatalog.series = append(dirCatalog.series, series)
				}
				dirCatalog.unMatchedFiles[fileKey{series: series, base: base}] = af
			}
			toc.catalogs[dir] = dirCatalog
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
// Each matcher tries the sidecars of the file's own takeout series first, then a sidecar of another series
// when neither name is indexed nor found in several series (see fileKey and borrowable).

type matcherFn func(jsonName string, fileName string, sm filetypes.SupportedMedia) bool

// seriesPass is a round of a matcher over a directory: sidecars of the file's own series first, then, when the
// directory holds several series, a borrowable sidecar of another series.
type seriesPass int

const (
	ownSeries seriesPass = iota
	otherSeries
)

// considers tells whether the pass looks at the sidecar j for the file f.
func (p seriesPass) considers(j, f fileKey) bool {
	return (j.series == f.series) == (p == ownSeries)
}

// matchers is a list of matcherFn from the most likely to be used to the least one
var matchers = []struct {
	name string
	fn   matcherFn
}{
	{name: "matchFastTrack", fn: matchFastTrack},
	{name: "matchNormal", fn: matchNormal},
	{name: "matchLivePhotoVideo", fn: matchLivePhotoVideo},
	{name: "matchForgottenDuplicates", fn: matchForgottenDuplicates},
	{name: "matchEditedName", fn: matchEditedName},
}

func (toc *TakeoutCmd) solvePuzzle(ctx context.Context) error {
	dirs := gen.MapKeysSorted(toc.catalogs)
	for _, dir := range dirs {
		cat := toc.catalogs[dir]
		jsons := sortedFileKeys(cat.jsons)
		passes := []seriesPass{ownSeries}
		if cat.holdsSeveralSeries() {
			passes = append(passes, otherSeries)
		}
		seriesPerName := cat.seriesPerName()
		for _, matcher := range matchers {
			for _, pass := range passes {
				for _, json := range jsons {
					md := cat.jsons[json]
					for f := range cat.unMatchedFiles {
						if !pass.considers(json, f) {
							continue
						}
						select {
						case <-ctx.Done():
							return ctx.Err()
						default:
							if matcher.fn(json.base, f.base, toc.supportedMedia) && (pass == ownSeries || borrowable(json, f, seriesPerName)) {
								i := cat.unMatchedFiles[f]
								i.matcher = matcher.name
								i.md = md
								a := toc.makeAsset(ctx, dir, i, md)
								cat.matchedFiles[f] = a
								toc.processor.RecordNonAsset(ctx, fshelper.FSName(i.fsys, path.Join(dir, i.base)), 0, fileevent.ProcessedAssociatedMetadata, "json", md.File, "matcher", matcher.name)
								delete(cat.unMatchedFiles, f)
							}
						}
					}
				}
			}
		}
		toc.catalogs[dir] = cat
		if len(cat.unMatchedFiles) > 0 {
			for _, f := range sortedFileKeys(cat.unMatchedFiles) {
				i := cat.unMatchedFiles[f]
				toc.processor.RecordNonAsset(ctx, fshelper.FSName(i.fsys, path.Join(dir, i.base)), 0, fileevent.ProcessedMissingMetadata)
				if toc.KeepJSONLess {
					a := toc.makeAsset(ctx, dir, i, nil)
					cat.matchedFiles[f] = a
					delete(cat.unMatchedFiles, f)
				}
			}
		}
	}
	return nil
}

// findNamesakes collects the names and the titles of the files that a directory keeps in several series. They are
// different files, as pass one dropped the identical ones, and the upload can't tell them apart by name and date.
// Their copies in other directories (albums) have the same name, or at least the same title.
func (toc *TakeoutCmd) findNamesakes() {
	toc.namesakes = map[string]bool{}
	for _, cat := range toc.catalogs {
		if len(cat.series) < 2 {
			continue // one series holds one file per name
		}
		byName := map[string][]*assets.Asset{}
		for k, a := range cat.matchedFiles {
			byName[k.base] = append(byName[k.base], a)
		}
		for base, as := range byName {
			if len(as) < 2 {
				continue
			}
			toc.namesakes[base] = true
			for _, a := range as {
				toc.namesakes[a.OriginalFileName] = true
			}
		}
	}
}

// Browse return a channel of assets
// Each asset is a group of files that are associated with each other

func (toc *TakeoutCmd) passTwo(ctx context.Context, gOut chan *assets.Group) error {
	dirs := gen.MapKeys(toc.catalogs)
	sort.Strings(dirs)
	for _, dir := range dirs {
		if len(toc.catalogs[dir].matchedFiles) > 0 {
			err := toc.handleDir(ctx, dir, gOut)
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

func (toc *TakeoutCmd) handleDir(ctx context.Context, dir string, gOut chan *assets.Group) error {
	catalog := toc.catalogs[dir]

	dirEntries := make([]*assets.Asset, 0, len(catalog.matchedFiles))
	seriesOf := map[*assets.Asset]string{}
	namesakeSeries := map[string]bool{} // series of the directory's files with namesakes, before any is dropped

	// Filter and sort the files
	for _, name := range sortedFileKeys(catalog.matchedFiles) {
		a := catalog.matchedFiles[name]
		a.HasNamesake = toc.namesakes[name.base] || toc.namesakes[a.OriginalFileName]
		seriesOf[a] = groupingSeries(name.series, a)
		if a.HasNamesake {
			namesakeSeries[seriesOf[a]] = true
		}
		key := fileKeyTracker{baseName: name.base, size: int64(a.FileSize)}
		track, _ := toc.fileTracker.Load(key) // track := to.fileTracker[key]
		if track.status == fileevent.ProcessedUploadSuccess {
			a.Close()
			toc.processor.RecordAssetDiscarded(ctx, a.File, int64(a.FileSize), fileevent.DiscardedLocalDuplicate, "local duplicate")
			continue
		}

		// Filter on metadata
		if code := toc.filterOnMetadata(ctx, a); code != fileevent.Code(0) {
			continue
		}
		dirEntries = append(dirEntries, a)
	}

	for _, entries := range partitionBySeries(dirEntries, seriesOf, len(namesakeSeries) > 1) {
		if len(entries) == 0 {
			continue
		}
		if err := toc.sendGroups(ctx, entries, gOut); err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	return nil
}

// groupingSeries is the series a file is grouped in: that of its sidecar, as an edit or a Live Photo half can take
// the sidecar of its original in another series, else its own.
func groupingSeries(own string, a *assets.Asset) string {
	if md := a.FromApplication; md != nil && md.File.FS() != nil {
		return takeoutSeries(md.File.FS())
	}
	return own
}

// partitionBySeries splits the files of a directory for grouping. When the directory holds namesakes (#1473) of
// several series (split), different photos of one name, its files are grouped series by series: no group, hence no
// stack, mixes two photos, and every grouper keeps a file with its own companions (edited copy, RAW, Live Photo half,
// scan copies, burst frames), which share its sidecar's series in real exports. Any other directory is grouped as a
// whole, as before.
func partitionBySeries(entries []*assets.Asset, seriesOf map[*assets.Asset]string, split bool) [][]*assets.Asset {
	if !split {
		return [][]*assets.Asset{entries}
	}
	bySeries := map[string][]*assets.Asset{}
	for _, a := range entries {
		bySeries[seriesOf[a]] = append(bySeries[seriesOf[a]], a)
	}
	parts := make([][]*assets.Asset, 0, len(bySeries))
	for _, s := range gen.MapKeysSorted(bySeries) {
		parts = append(parts, bySeries[s])
	}
	return parts
}

// sendGroups groups the files of a directory and sends the groups to gOut.
func (toc *TakeoutCmd) sendGroups(ctx context.Context, dirEntries []*assets.Asset, gOut chan *assets.Group) error {
	// the go routine push all asset to the in chanel for been grouped
	in := make(chan *assets.Asset)
	go func() {
		defer close(in)

		sort.SliceStable(dirEntries, func(i, j int) bool {
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
			if toc.CreateAlbums {
				if toc.ImportIntoAlbum != "" {
					// Force this album
					a.Albums = []assets.Album{{Title: toc.ImportIntoAlbum}}
				} else {
					// check if its duplicates are in some albums, and push them all at once
					key := fileKeyTracker{baseName: filepath.Base(a.File.Name()), size: int64(a.FileSize)}
					track, _ := toc.fileTracker.Load(key) // track := to.fileTracker[key]
					for _, p := range track.paths {
						if album, ok := toc.albums[p]; ok {
							title := album.Title
							if title == "" {
								if !toc.KeepUntitled {
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
				if toc.PartnerSharedAlbum != "" && a.FromPartner {
					a.Albums = append(a.Albums, assets.Album{Title: toc.PartnerSharedAlbum})
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

			if toc.TakeoutTag {
				a.AddTag(toc.TakeoutName)
			}

			select {
			case in <- a:
			case <-ctx.Done():
				return
			}
		}
	}()

	// pull assets from the in chan, and pass them to the pipeline
	gs := groups.NewGrouperPipeline(ctx, toc.groupers...).PipeGrouper(ctx, in)
	for g := range gs {
		select {
		case gOut <- g:
			for _, a := range g.Assets {
				key := fileKeyTracker{
					baseName: path.Base(a.File.Name()),
					size:     int64(a.FileSize),
				}
				track, _ := toc.fileTracker.Load(key) // track := to.fileTracker[key]
				track.status = fileevent.ProcessedUploadSuccess
				toc.fileTracker.Store(key, track) // to.fileTracker[key] = track
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

// makeAsset makes a localAssetFile based on the google metadata
func (toc *TakeoutCmd) makeAsset(_ context.Context, dir string, f *assetFile, md *assets.Metadata) *assets.Asset {
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
	a.SetNameInfo(toc.infoCollector.GetInfo(a.OriginalFileName))
	// Google Takeout names an edited copy with a localized, possibly truncated suffix
	// (e.g. "-edited", "-modifié") and matches it to the original's sidecar through
	// matchEditedName. Mark it so the upload layer stacks it over the original and
	// shows it as the cover.
	if f.matcher == "matchEditedName" {
		a.Kind = assets.KindEdited
		a.IsModified = true
		a.IsCover = true
	}
	return a
}

// filterOnMetadata, log discared files and closes the asset
func (toc *TakeoutCmd) filterOnMetadata(ctx context.Context, a *assets.Asset) fileevent.Code {
	if !toc.KeepArchived && a.Visibility == assets.VisibilityArchive {
		toc.processor.RecordAssetDiscarded(ctx, a.File, int64(a.FileSize), fileevent.DiscardedFiltered, "discarding archived file")
		a.Close()
		return fileevent.DiscardedFiltered
	}
	if !toc.KeepPartner && a.FromPartner {
		toc.processor.RecordAssetDiscarded(ctx, a.File, int64(a.FileSize), fileevent.DiscardedFiltered, "discarding partner file")
		a.Close()
		return fileevent.DiscardedFiltered
	}
	if !toc.KeepTrashed && a.Trashed {
		toc.processor.RecordAssetDiscarded(ctx, a.File, int64(a.FileSize), fileevent.DiscardedFiltered, "discarding trashed file")
		a.Close()
		return fileevent.DiscardedFiltered
	}

	if toc.InclusionFlags.DateRange.IsSet() && !toc.InclusionFlags.DateRange.InRange(a.CaptureDate) {
		toc.processor.RecordAssetDiscarded(ctx, a.File, int64(a.FileSize), fileevent.DiscardedFiltered, "discarding files out of date range")
		a.Close()
		return fileevent.DiscardedFiltered
	}
	if toc.ImportFromAlbum != "" {
		keep := false
		dir := path.Dir(a.File.Name())
		if dir == "." {
			dir = ""
		}
		if album, ok := toc.albums[dir]; ok {
			keep = keep || album.Title == toc.ImportFromAlbum
		}
		if !keep {
			toc.processor.RecordAssetDiscarded(ctx, a.File, int64(a.FileSize), fileevent.DiscardedFiltered, "discarding files not in the specified album")
			a.Close()
			return fileevent.DiscardedFiltered
		}
	}
	return fileevent.Code(0)
}
