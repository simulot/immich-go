package gp

import (
	"context"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/simulot/immich-go/browser"
	"github.com/simulot/immich-go/helpers/fileevent"
	"github.com/simulot/immich-go/helpers/fshelper"
	"github.com/simulot/immich-go/helpers/gen"
	"github.com/simulot/immich-go/helpers/namematcher"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/immich/metadata"
)

type Takeout struct {
	fsyss      []fs.FS
	catalogs   dirCatalog                    // file catalogs by directory in the set of the all takeout parts
	jsonByYear map[jsonKey]*GoogleMetaData   // assets by year of capture and base name
	uploaded   map[fileKey]any               // track files already uploaded
	albums     map[string]browser.LocalAlbum // tack album names by folder
	log        *fileevent.Recorder
	sm         immich.SupportedMedia
	banned     namematcher.List // Banned files
}

// dirCatalog collects all directory catalogs
type dirCatalog map[string]directoryCatalog // by directory in the walker

// directoryCatalog captures all files in a given directory
type directoryCatalog struct {
	unMatchedFiles map[string]fileInfo // map of fileInfo by base name
	matchedFiles   map[string]fileInfo // map of fileInfo by base name
}

// fileInfo keep information collected during pass one
type fileInfo struct {
	fsys   fs.FS           // Remember in which part of the archive the the file
	base   string          // Remember the original file name
	length int             // file length in bytes
	md     *GoogleMetaData // will point to the associated metadata
}

// fileKey is the key of the uploaded files map
// GP can't have duplicate JSON name in the same year.
type fileKey struct {
	base   string
	length int
	year   int
}

// jsonKey allow to map jsons by base name and year of capture
type jsonKey struct {
	name string
	year int
}

func NewTakeout(ctx context.Context, l *fileevent.Recorder, sm immich.SupportedMedia, fsyss ...fs.FS) (*Takeout, error) {
	to := Takeout{
		fsyss:      fsyss,
		jsonByYear: map[jsonKey]*GoogleMetaData{},
		albums:     map[string]browser.LocalAlbum{},
		log:        l,
		sm:         sm,
	}

	return &to, nil
}

func (to *Takeout) SetBannedFiles(banned namematcher.List) *Takeout {
	to.banned = banned
	return to
}

// Prepare scans all files in all walker to build the file catalog of the archive
// metadata files content is read and kept

func (to *Takeout) Prepare(ctx context.Context) error {
	to.catalogs = dirCatalog{}
	for _, w := range to.fsyss {
		err := to.passOneFsWalk(ctx, w)
		if err != nil {
			return err
		}
	}
	err := to.solvePuzzle(ctx)
	return err
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

			dirCatalog := to.catalogs[dir]
			if dirCatalog.unMatchedFiles == nil {
				dirCatalog.unMatchedFiles = map[string]fileInfo{}
			}
			if dirCatalog.matchedFiles == nil {
				dirCatalog.matchedFiles = map[string]fileInfo{}
			}
			finfo, err := d.Info()
			if err != nil {
				return err
			}
			switch ext {
			case ".json":
				md, err := fshelper.ReadJSON[GoogleMetaData](w, name)
				if err == nil {
					switch {
					case md.isAsset():
						to.addJSON(dir, base, md)
						to.log.Record(ctx, fileevent.DiscoveredSidecar, nil, name, "type", "asset metadata", "title", md.Title)
					case md.isAlbum():
						a := to.albums[dir]
						a.Title = md.Title
						a.Path = filepath.Base(dir)
						if e := md.Enrichments; e != nil {
							a.Description = e.Text
							a.Latitude = e.Latitude
							a.Longitude = e.Longitude
						}
						to.albums[dir] = a
						to.log.Record(ctx, fileevent.DiscoveredSidecar, nil, name, "type", "album metadata", "title", md.Title)
					default:
						to.log.Record(ctx, fileevent.DiscoveredUnsupported, nil, name, "reason", "unknown JSONfile")
						return nil
					}
				} else {
					to.log.Record(ctx, fileevent.DiscoveredUnsupported, nil, name, "reason", "unknown JSONfile")
					return nil
				}
			default:
				t := to.sm.TypeFromExt(ext)
				switch t {
				case immich.TypeUnknown:
					to.log.Record(ctx, fileevent.DiscoveredUnsupported, nil, name, "reason", "unsupported file type")
					return nil
				case immich.TypeVideo:
					to.log.Record(ctx, fileevent.DiscoveredVideo, nil, name)
					if strings.Contains(name, "Failed Videos") {
						to.log.Record(ctx, fileevent.DiscoveredDiscarded, nil, name, "reason", "can't upload failed videos")
						return nil
					}
				case immich.TypeImage:
					to.log.Record(ctx, fileevent.DiscoveredImage, nil, name)
				}

				if to.banned.Match(name) {
					to.log.Record(ctx, fileevent.DiscoveredDiscarded, nil, name, "reason", "banned file")
					return nil
				}
				dirCatalog.unMatchedFiles[base] = fileInfo{
					base:   base,
					fsys:   w,
					length: int(finfo.Size()),
				}
			}
			to.catalogs[dir] = dirCatalog
			return nil
		}
	})
	return err
}

// addJSON stores metadata and all paths where the combo base+year has been found
func (to *Takeout) addJSON(dir, base string, md *GoogleMetaData) {
	k := jsonKey{
		name: base,
		year: md.PhotoTakenTime.Time().Year(),
	}

	if mdPresent, ok := to.jsonByYear[k]; ok {
		md = mdPresent
	}
	md.foundInPaths = append(md.foundInPaths, dir)
	to.jsonByYear[k] = md
}

type matcherFn func(jsonName string, fileName string, sm immich.SupportedMedia) bool

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

// solvePuzzle prepares metadata with information collected during pass one for each accepted files
//
// JSON files give important information about the relative photos / movies:
//   - The original name (useful when it as been truncated)
//   - The date of capture (useful when the files doesn't have this date)
//   - The GPS coordinates (will be useful in a future release)
//
// Each JSON is checked. JSON is duplicated in albums folder.
// Associated files with the JSON can be found in the JSON's folder, or in the Year photos.
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
// The duplicates files (same name, same length in bytes) found in the local source are discarded before been presented to the immich server.
//

func (to *Takeout) solvePuzzle(ctx context.Context) error {
	jsonKeys := gen.MapKeys(to.jsonByYear)
	sort.Slice(jsonKeys, func(i, j int) bool {
		yd := jsonKeys[i].year - jsonKeys[j].year
		switch {
		case yd < 0:
			return true
		case yd > 0:
			return false
		}
		return jsonKeys[i].name < jsonKeys[j].name
	})

	// For the most common matcher to the least,
	for _, matcher := range matchers {
		// Check files that match each json files
		for _, k := range jsonKeys {
			md := to.jsonByYear[k]

			// list of paths where to search the assets: paths where this json has been found + year path in all of the walkers
			paths := map[string]any{}
			paths[path.Join(path.Dir(md.foundInPaths[0]), fmt.Sprintf("Photos from %d", md.PhotoTakenTime.Time().Year()))] = nil
			for _, d := range md.foundInPaths {
				paths[d] = nil
			}
			for d := range paths {
				l := to.catalogs[d]
				for f := range l.unMatchedFiles {
					select {
					case <-ctx.Done():
						return ctx.Err()
					default:
						if matcher.fn(k.name, f, to.sm) {
							i := l.unMatchedFiles[f]
							i.md = md
							l.matchedFiles[f] = i
							to.log.Record(ctx, fileevent.AnalysisAssociatedMetadata, l.unMatchedFiles[f], filepath.Join(d, f), "json", k.name, "year", k.year, "size", i.length, "matcher", matcher.name)
							delete(l.unMatchedFiles, f)
						}
					}
				}
				to.catalogs[d] = l
			}
		}
	}

	paths := gen.MapKeys(to.catalogs)
	sort.Strings(paths)
	for _, p := range paths {
		files := gen.MapKeys(to.catalogs[p].unMatchedFiles)
		sort.Strings(files)
		for _, f := range files {
			to.log.Record(ctx, fileevent.AnalysisMissingAssociatedMetadata, to.catalogs[p].unMatchedFiles[f], filepath.Join(p, f))
		}
	}
	return nil
}

// normalMatch
//
//	PXL_20230922_144936660.jpg.json
//	PXL_20230922_144936660.jpg
func normalMatch(jsonName string, fileName string, sm immich.SupportedMedia) bool {
	base := strings.TrimSuffix(jsonName, path.Ext(jsonName))
	return base == fileName
}

// livePhotoMatch
// 20231227_152817.jpg.json
// 20231227_152817.MP4
//
// PXL_20231118_035751175.MP.jpg.json
// PXL_20231118_035751175.MP.jpg
// PXL_20231118_035751175.MP
func livePhotoMatch(jsonName string, fileName string, sm immich.SupportedMedia) bool {
	fileExt := path.Ext(fileName)
	fileName = strings.TrimSuffix(fileName, fileExt)
	base := strings.TrimSuffix(jsonName, path.Ext(jsonName))
	base = strings.TrimSuffix(base, path.Ext(base))
	if base == fileName {
		return true
	}
	base = strings.TrimSuffix(base, path.Ext(base))
	return base == fileName
}

// matchWithOneCharOmitted
//
//	PXL_20230809_203449253.LONG_EXPOSURE-02.ORIGIN.json
//	PXL_20230809_203449253.LONG_EXPOSURE-02.ORIGINA.jpg
//
//	05yqt21kruxwwlhhgrwrdyb6chhwszi9bqmzu16w0 2.jp.json
//	05yqt21kruxwwlhhgrwrdyb6chhwszi9bqmzu16w0 2.jpg
//
//  😀😃😄😁😆😅😂🤣🥲☺️😊😇🙂🙃😉😌😍🥰😘😗😙😚😋.json
//  😀😃😄😁😆😅😂🤣🥲☺️😊😇🙂🙃😉😌😍🥰😘😗😙😚😋😛.jpg

func matchWithOneCharOmitted(jsonName string, fileName string, sm immich.SupportedMedia) bool {
	base := strings.TrimSuffix(jsonName, path.Ext(jsonName))
	if strings.HasPrefix(fileName, base) {
		if t := sm.TypeFromExt(path.Ext(base)); t == immich.TypeImage || t == immich.TypeVideo {
			// Trim only if the EXT is known extension, and not .COVER or .ORIGINAL
			base = strings.TrimSuffix(base, path.Ext(base))
		}
		fileName = strings.TrimSuffix(fileName, path.Ext(fileName))
		a, b := utf8.RuneCountInString(fileName), utf8.RuneCountInString(base)
		if a-b <= 1 {
			return true
		}
	}
	return false
}

// matchVeryLongNameWithNumber
//
//	Backyard_ceremony_wedding_photography_xxxxxxx_(494).json
//	Backyard_ceremony_wedding_photography_xxxxxxx_m(494).jpg
func matchVeryLongNameWithNumber(jsonName string, fileName string, sm immich.SupportedMedia) bool {
	jsonName = strings.TrimSuffix(jsonName, path.Ext(jsonName))

	p1JSON := strings.Index(jsonName, "(")
	if p1JSON < 0 {
		return false
	}
	p2JSON := strings.Index(jsonName, ")")
	if p2JSON < 0 || p2JSON != len(jsonName)-1 {
		return false
	}
	p1File := strings.Index(fileName, "(")
	if p1File < 0 || p1File != p1JSON+1 {
		return false
	}
	if jsonName[:p1JSON] != fileName[:p1JSON] {
		return false
	}
	p2File := strings.Index(fileName, ")")
	return jsonName[p1JSON+1:p2JSON] == fileName[p1File+1:p2File]
}

// matchDuplicateInYear
//
//	IMG_3479.JPG(2).json
//	IMG_3479(2).JPG
func matchDuplicateInYear(jsonName string, fileName string, sm immich.SupportedMedia) bool {
	jsonName = strings.TrimSuffix(jsonName, path.Ext(jsonName))
	p1JSON := strings.Index(jsonName, "(")
	if p1JSON < 1 {
		return false
	}
	p2JSON := strings.Index(jsonName, ")")
	if p2JSON < 0 || p2JSON != len(jsonName)-1 {
		return false
	}

	num := jsonName[p1JSON:]
	jsonName = strings.TrimSuffix(jsonName, num)
	ext := path.Ext(jsonName)
	jsonName = strings.TrimSuffix(jsonName, ext) + num + ext
	return jsonName == fileName
}

// matchEditedName
//   PXL_20220405_090123740.PORTRAIT.jpg.json
//   PXL_20220405_090123740.PORTRAIT.jpg
//   PXL_20220405_090123740.PORTRAIT-modifié.jpg

func matchEditedName(jsonName string, fileName string, sm immich.SupportedMedia) bool {
	base := strings.TrimSuffix(jsonName, path.Ext(jsonName))
	ext := path.Ext(base)
	if ext != "" {
		if sm.IsMedia(ext) {
			base := strings.TrimSuffix(base, ext)
			fname := strings.TrimSuffix(fileName, path.Ext(fileName))
			return strings.HasPrefix(fname, base)
		}
	}
	return false
}

// TODO: This one interferes with matchVeryLongNameWithNumber

// matchForgottenDuplicates
// original_1d4caa6f-16c6-4c3d-901b-9387de10e528_.json
// original_1d4caa6f-16c6-4c3d-901b-9387de10e528_P.jpg
// original_1d4caa6f-16c6-4c3d-901b-9387de10e528_P(1).jpg

func matchForgottenDuplicates(jsonName string, fileName string, sm immich.SupportedMedia) bool {
	jsonName = strings.TrimSuffix(jsonName, path.Ext(jsonName))
	fileName = strings.TrimSuffix(fileName, path.Ext(fileName))
	if strings.HasPrefix(fileName, jsonName) {
		a, b := utf8.RuneCountInString(jsonName), utf8.RuneCountInString(fileName)
		if b-a < 10 {
			return true
		}
	}
	return false
}

// Browse return a channel of assets
//
// Walkers are rewind, and scanned again
// each file net yet sent to immich is sent with associated metadata

func (to *Takeout) Browse(ctx context.Context) chan *browser.LocalAssetFile {
	to.uploaded = map[fileKey]any{}
	assetChan := make(chan *browser.LocalAssetFile)

	go func() {
		defer close(assetChan)
		for dir := range to.catalogs {
			if len(to.catalogs[dir].matchedFiles) > 0 {
				err := to.passTwo(ctx, dir, assetChan)
				if err != nil {
					assetChan <- &browser.LocalAssetFile{Err: err}
				}
			}
		}
	}()
	return assetChan
}

func (to *Takeout) passTwo(ctx context.Context, dir string, assetChan chan *browser.LocalAssetFile) error {
	catalog := to.catalogs[dir]

	linkedFiles := map[string]struct {
		video fileInfo
		image fileInfo
	}{}

	// detects couples image + video, likely been a motion picture
	for _, f := range gen.MapKeys(catalog.matchedFiles) {
		ext := path.Ext(f)
		base := strings.TrimSuffix(f, ext)
		ext2 := path.Ext(base)
		if to.sm.IsMedia(ext2) {
			base = strings.TrimSuffix(base, ext2)
		}

		linked := linkedFiles[base]
		switch to.sm.TypeFromExt(ext) {
		case immich.TypeVideo:
			linked.video = catalog.matchedFiles[f]
		case immich.TypeImage:
			linked.image = catalog.matchedFiles[f]
		}
		linkedFiles[base] = linked
	}

	for _, base := range gen.MapKeys(linkedFiles) {
		var a *browser.LocalAssetFile
		var err error

		linked := linkedFiles[base]

		if linked.image.md != nil {
			a, err = to.googleMDToAsset(linked.image.md, linked.image.fsys, path.Join(dir, linked.image.base))
			if err != nil {
				to.log.Record(ctx, fileevent.Error, nil, path.Join(dir, linked.image.base), "error", err.Error())
				continue
			}
			if linked.video.md != nil {
				i, err := to.googleMDToAsset(linked.video.md, linked.video.fsys, path.Join(dir, linked.video.base))
				if err != nil {
					to.log.Record(ctx, fileevent.Error, nil, path.Join(dir, linked.video.base), "error", err.Error())
				} else {
					a.LivePhoto = i
				}
			}
		} else {
			a, err = to.googleMDToAsset(linked.video.md, linked.video.fsys, path.Join(dir, linked.video.base))
			if err != nil {
				to.log.Record(ctx, fileevent.Error, nil, path.Join(dir, linked.video.base), "error", err.Error())
				continue
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			fk := fileKey{
				base:   filepath.Base(a.FileName),
				length: a.FileSize,
				year:   a.Metadata.DateTaken.Year(),
			}
			if _, found := to.uploaded[fk]; !found {
				assetChan <- a
				to.uploaded[fk] = nil
			} else {
				to.log.Record(ctx, fileevent.AnalysisLocalDuplicate, nil, a.FileName, "title", a.Title)
				if a.LivePhoto != nil {
					to.log.Record(ctx, fileevent.AnalysisLocalDuplicate, nil, a.LivePhoto.FileName, "title", a.LivePhoto.Title)
				}
			}
		}
	}
	return nil
}

// googleMDToAsset makes a localAssetFile based on the google metadata
func (to *Takeout) googleMDToAsset(md *GoogleMetaData, fsys fs.FS, name string) (*browser.LocalAssetFile, error) {
	// Change file's title with the asset's title and the actual file's extension
	title := md.Title
	titleExt := path.Ext(title)
	fileExt := path.Ext(name)
	if titleExt != fileExt {
		title = strings.TrimSuffix(title, titleExt)
		titleExt = path.Ext(title)
		if titleExt != fileExt {
			title = strings.TrimSuffix(title, titleExt) + fileExt
		}
	}

	i, err := fs.Stat(fsys, name)
	if err != nil {
		return nil, err
	}
	a := browser.LocalAssetFile{
		FileName:    name,
		FileSize:    int(i.Size()),
		Title:       title,
		Archived:    md.Archived,
		FromPartner: md.isPartner(),
		Trashed:     md.Trashed,
		Favorite:    md.Favorited,

		FSys: fsys,
	}

	// Prepare sidecar data to force Immich with Google metadata
	sidecar := metadata.Metadata{
		Description: md.Description,
		DateTaken:   md.PhotoTakenTime.Time(),
	}
	if md.GeoDataExif.Latitude != 0 || md.GeoDataExif.Longitude != 0 {
		sidecar.Latitude = md.GeoDataExif.Latitude
		sidecar.Longitude = md.GeoDataExif.Longitude
	}
	if md.GeoData.Latitude != 0 || md.GeoData.Longitude != 0 {
		sidecar.Latitude = md.GeoData.Latitude
		sidecar.Longitude = md.GeoData.Longitude
	}

	for _, p := range md.foundInPaths {
		if album, exists := to.albums[p]; exists {
			if (album.Latitude != 0 || album.Longitude != 0) && (sidecar.Latitude == 0 && sidecar.Longitude == 0) {
				sidecar.Latitude = album.Latitude
				sidecar.Longitude = album.Longitude
			}
			a.Albums = append(a.Albums, album)
		}
	}

	a.Metadata = sidecar
	return &a, nil
}
