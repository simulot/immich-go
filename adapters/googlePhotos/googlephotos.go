package gp

import (
	"context"
	"io/fs"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/simulot/immich-go/adapters"
	"github.com/simulot/immich-go/helpers/gen"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/simulot/immich-go/internal/metadata"
)

type Takeout struct {
	fsyss            []fs.FS
	catalogs         map[string]directoryCatalog     // file catalogs by directory in the set of the all takeout parts
	albums           map[string]adapters.LocalAlbum  // track album names by folder
	fileTracker      map[fileKeyTracker]trackingInfo // key is base name + file size,  value is list of file paths
	debugLinkedFiles []linkedFiles
	log              *fileevent.Recorder
	flags            *ImportFlags // command-line flags
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
	matchedFiles   map[string]*assetFile         // files matched by base name
}

// assetFile keep information collected during pass one
type assetFile struct {
	fsys   fs.FS              // Remember in which part of the archive the the file
	base   string             // Remember the original file name
	length int                // file length in bytes
	md     *metadata.Metadata // will point to the associated metadata
}

func NewTakeout(ctx context.Context, l *fileevent.Recorder, flags *ImportFlags, fsyss ...fs.FS) (*Takeout, error) {
	to := Takeout{
		fsyss:       fsyss,
		catalogs:    map[string]directoryCatalog{},
		albums:      map[string]adapters.LocalAlbum{},
		fileTracker: map[fileKeyTracker]trackingInfo{},
		log:         l,
		flags:       flags,
	}

	return &to, nil
}

// Prepare scans all files in all walker to build the file catalog of the archive
// metadata files content is read and kept

func (to *Takeout) Browse(ctx context.Context) (chan *adapters.LocalAssetFile, error) {
	for _, w := range to.fsyss {
		err := to.passOneFsWalk(ctx, w)
		if err != nil {
			return nil, err
		}
	}
	err := to.solvePuzzle(ctx)
	if err != nil {
		return nil, err
	}
	return to.nextPass(ctx), nil
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
				dirCatalog.matchedFiles = map[string]*assetFile{}
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
						to.log.Record(ctx, fileevent.DiscoveredSidecar, fileevent.AsFileAndName(w, name), "type", "asset metadata", "title", md.Title)
					case md.isAlbum():
						if !to.flags.KeepUntitled && md.Title == "" {
							to.log.Record(ctx, fileevent.DiscoveredUnsupported, fileevent.AsFileAndName(w, name), "reason", "discard untitled album")
							return nil
						}
						a := to.albums[dir]
						a.Title = md.Title
						if a.Title == "" {
							a.Title = filepath.Base(dir)
						}
						a.Path = filepath.Base(dir)
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

				if _, ok := dirCatalog.unMatchedFiles[base]; ok {
					to.log.Record(ctx, fileevent.AnalysisLocalDuplicate, fileevent.AsFileAndName(w, name), "reason", "duplicated in the folder")
					return nil
				}

				dirCatalog.unMatchedFiles[base] = &assetFile{
					fsys:   w,
					base:   base,
					length: int(finfo.Size()),
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
							cat.matchedFiles[f] = i
							to.log.Record(ctx, fileevent.AnalysisAssociatedMetadata, fileevent.AsFileAndName(nil, filepath.Join(dir, f)), "json", json, "size", i.length, "matcher", matcher.name)
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
				to.log.Record(ctx, fileevent.AnalysisMissingAssociatedMetadata, fileevent.AsFileAndName(nil, filepath.Join(dir, f)), filepath.Join(dir, f))
				if to.flags.KeepJSONLess {
					cat.matchedFiles[f] = cat.unMatchedFiles[f]
					delete(cat.unMatchedFiles, f)
				}
			}
		}
	}
	return nil
}

// Browse return a channel of assets
//
// Walkers are rewind, and scanned again
// each file net yet sent to immich is sent with associated metadata

func (to *Takeout) nextPass(ctx context.Context) chan *adapters.LocalAssetFile {
	assetChan := make(chan *adapters.LocalAssetFile)

	go func() {
		defer close(assetChan)
		dirs := gen.MapKeys(to.catalogs)
		sort.Strings(dirs)
		for _, dir := range dirs {
			if len(to.catalogs[dir].matchedFiles) > 0 {
				err := to.passTwo(ctx, dir, assetChan)
				if err != nil {
					assetChan <- &adapters.LocalAssetFile{Err: err}
				}
			}
		}
	}()
	return assetChan
}

type linkedFiles struct {
	dir   string
	base  string
	video *assetFile
	image *assetFile
}

func (to *Takeout) passTwo(ctx context.Context, dir string, assetChan chan *adapters.LocalAssetFile) error {
	catalog := to.catalogs[dir]

	linkedFiles := map[string]linkedFiles{}
	matchedFiles := gen.MapKeysSorted(catalog.matchedFiles)

	// skip duplicates
	newMatchedFiles := []string{}
	for _, name := range matchedFiles {
		file := catalog.matchedFiles[name]
		key := fileKeyTracker{baseName: file.base, size: int64(file.length)}
		track := to.fileTracker[key]
		if track.status == fileevent.Uploaded {
			// track.count++
			// to.fileTracker[key] = track
			to.logMessage(ctx, fileevent.AnalysisLocalDuplicate, fileevent.AsFileAndName(file.fsys, path.Join(dir, name)), "local duplicate")
			continue
		}

		newMatchedFiles = append(newMatchedFiles, name)
		ext := filepath.Ext(name)
		if to.flags.SupportedMedia.TypeFromExt(ext) == metadata.TypeImage {
			linked := linkedFiles[name]
			linked.image = catalog.matchedFiles[name]
			linkedFiles[name] = linked
		}
	}
	matchedFiles = newMatchedFiles

	// Scan videos and try to detect motion pictures
nextVideo:
	for _, f := range matchedFiles {
		fExt := filepath.Ext(f)
		if to.flags.SupportedMedia.TypeFromExt(fExt) == metadata.TypeVideo {
			name := strings.TrimSuffix(f, fExt)
			//  Check if there is a matching image
			for i, linked := range linkedFiles {
				if linked.image == nil {
					// not an image... next
					continue
				}
				if linked.image != nil && linked.video != nil {
					// already associated ... next
					continue
				}

				p := linked.image.base
				ext := filepath.Ext(p)
				p = strings.TrimSuffix(p, ext)
				ext = filepath.Ext(p)
				// manage .MP motion picture files
				if strings.ToUpper(ext) == ".MP" || strings.HasPrefix(strings.ToUpper(ext), ".MP~") {
					if fExt != ext {
						continue
					}
					p = strings.TrimSuffix(p, ext)
				}
				// image and video files with the same base name. They are linked
				if p == name {
					linked.video = catalog.matchedFiles[f]
					linkedFiles[i] = linked
					continue nextVideo
				}
			}
			//  no matching image found, create a new linked file for the movie
			linked := linkedFiles[f]
			linked.video = catalog.matchedFiles[f]
			linkedFiles[f] = linked
		}
	}

	// Process files from the directory
	for _, base := range gen.MapKeysSorted(linkedFiles) {
		var (
			mainAsset *adapters.LocalAssetFile
			liveVideo *adapters.LocalAssetFile
		)

		linked := linkedFiles[base]
		linked.base = base
		linked.dir = dir
		to.debugLinkedFiles = append(to.debugLinkedFiles, linked)

		switch {
		case linked.image != nil && linked.video != nil:
			liveVideo = to.makeAsset(ctx, dir, linked.video, linked.video.md)
			mainAsset = to.makeAsset(ctx, dir, linked.image, linked.image.md)
		case linked.image != nil && linked.video == nil:
			mainAsset = to.makeAsset(ctx, dir, linked.image, linked.image.md)
		case linked.video != nil && linked.image == nil:
			mainAsset = to.makeAsset(ctx, dir, linked.video, linked.video.md)
		}

		if mainAsset == nil {
			continue
		}

		if liveVideo != nil {
			err := to.pushAsset(ctx, assetChan, liveVideo)
			if err != nil {
				return err
			}
			mainAsset.LivePhoto = liveVideo
		}
		err := to.pushAsset(ctx, assetChan, mainAsset)
		if err != nil {
			return err
		}
	}
	return nil
}

// pushAsset sends the asset to the next layer
func (to *Takeout) pushAsset(ctx context.Context, assetChan chan *adapters.LocalAssetFile, a *adapters.LocalAssetFile) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		assetChan <- a
		key := fileKeyTracker{
			baseName: path.Base(a.FileName),
			size:     int64(a.FileSize),
		}
		track := to.fileTracker[key]
		track.status = fileevent.Uploaded
		to.fileTracker[key] = track
	}
	return nil
}

// makeAsset makes a localAssetFile based on the google metadata
func (to *Takeout) makeAsset(ctx context.Context, dir string, f *assetFile, md *metadata.Metadata) *adapters.LocalAssetFile {
	key := fileKeyTracker{
		baseName: f.base,
		size:     int64(f.length),
	}
	track := to.fileTracker[key]
	track.metadata = md
	defer func() {
		to.fileTracker[key] = track
	}()

	file := path.Join(dir, f.base)
	i, err := fs.Stat(f.fsys, file)
	if err != nil {
		to.logMessage(ctx, fileevent.Error, fileevent.AsFileAndName(f.fsys, file), err.Error())
		track.status = fileevent.Error
		return nil
	}

	if md != nil {
		if !to.flags.KeepArchived && md.Archived {
			to.logMessage(ctx, fileevent.DiscoveredDiscarded, fileevent.AsFileAndName(f.fsys, file), "discarding archived file")
			track.status = fileevent.DiscoveredDiscarded
			return nil
		}
		if !to.flags.KeepPartner && md.FromPartner {
			to.logMessage(ctx, fileevent.DiscoveredDiscarded, fileevent.AsFileAndName(f.fsys, file), "discarding partner file")
			track.status = fileevent.DiscoveredDiscarded
			return nil
		}
		if !to.flags.KeepTrashed && md.Trashed {
			to.logMessage(ctx, fileevent.DiscoveredDiscarded, fileevent.AsFileAndName(f.fsys, file), "discarding trashed file")
			track.status = fileevent.DiscoveredDiscarded
			return nil
		}
		if !to.flags.InclusionFlags.DateRange.InRange(md.DateTaken) {
			to.logMessage(ctx, fileevent.DiscoveredDiscarded, fileevent.AsFileAndName(f.fsys, file), "discarding files out of date range")
			track.status = fileevent.DiscoveredDiscarded
			return nil
		}
	}

	title := f.base
	if md != nil {
		title = md.FileName
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
		md.FileName = title
	}
	a := adapters.LocalAssetFile{
		FileName: file,
		FileSize: int(i.Size()),
		Title:    title,
		FSys:     f.fsys,
	}

	if to.flags.ImportFromAlbum != "" {
		keep := false
		if album, ok := to.albums[dir]; ok {
			keep = keep || album.Title == to.flags.ImportFromAlbum
		}
		if !keep {
			to.logMessage(ctx, fileevent.DiscoveredDiscarded, fileevent.AsFileAndName(f.fsys, file), "discarding files not in the specified album")
			track.status = fileevent.DiscoveredDiscarded
			return nil
		}
	}

	if to.flags.CreateAlbums {
		if to.flags.ImportIntoAlbum != "" {
			// Force this album
			a.AddAlbum(adapters.LocalAlbum{Title: to.flags.ImportIntoAlbum})
		} else {
			// check if its duplicates are in some albums, and push them all at once

			track := to.fileTracker[key]
			for _, p := range track.paths {
				if album, ok := to.albums[p]; ok {
					title := album.Title
					if title == "" {
						if !to.flags.KeepUntitled {
							continue
						}
						title = filepath.Base(album.Path)
					}
					a.AddAlbum(adapters.LocalAlbum{
						Title:       title,
						Path:        p,
						Description: album.Description,
						Latitude:    album.Latitude,
						Longitude:   album.Longitude,
					})
				}
			}
		}

		// Force this album for partners photos
		if to.flags.PartnerSharedAlbum != "" && md != nil && md.FromPartner {
			a.AddAlbum(adapters.LocalAlbum{Title: to.flags.PartnerSharedAlbum})
		}

		if md != nil {
			md.Collections = md.Collections[0:0]
			for _, album := range a.Albums {
				md.Collections = append(md.Collections, album.Title)
			}
		}
	}

	if md != nil {
		a.Metadata = *md
		if a.Metadata.Latitude == 0 && a.Metadata.Longitude == 0 {
			for _, album := range a.Albums {
				if album.Latitude != 0 || album.Longitude != 0 {
					// when there isn't GPS information on the photo, but the album has a location,  use that location
					a.Metadata.Latitude = album.Latitude
					a.Metadata.Longitude = album.Longitude
					break
				}
			}
		}
	}
	return &a
}
