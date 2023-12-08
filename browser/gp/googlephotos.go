package gp

import (
	"context"
	"fmt"
	"immich-go/browser"
	"immich-go/helpers/fshelper"
	"immich-go/helpers/gen"
	"immich-go/journal"
	"immich-go/logger"
	"io/fs"
	"path"
	"sort"
	"strings"
	"unicode/utf8"
)

type Takeout struct {
	fsys        fs.FS
	filesByDir  map[string][]fileReference    // files name mapped by dir
	jsonByYear  map[jsonKey]*GoogleMetaData   // assets by year of capture and full path
	albumsByDir map[string]browser.LocalAlbum // album title mapped by dir
	log         logger.Logger
	conf        *browser.Configuration
}

type fileReference struct {
	fileKey
	taken bool // True, when the file as been associated to a json and sent to the uploader
}
type fileKey struct {
	name string
	size int64
}

type jsonKey struct {
	year int
	name string
}
type Album struct {
	Title string
}

func NewTakeout(ctx context.Context, fsys fs.FS, log logger.Logger, conf *browser.Configuration) (*Takeout, error) {
	to := Takeout{
		fsys:        fsys,
		filesByDir:  map[string][]fileReference{},
		jsonByYear:  map[jsonKey]*GoogleMetaData{},
		albumsByDir: map[string]browser.LocalAlbum{},
		log:         log,
		conf:        conf,
	}
	err := to.walk(ctx, fsys)

	return &to, err
}

// walk the given FS to collect images file names and metadata files
func (to *Takeout) walk(ctx context.Context, fsys fs.FS) error {
	to.log.OK("Scanning the Google Photos takeout")
	err := fs.WalkDir(fsys, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			// Check if the context has been cancelled
			return ctx.Err()
		default:
		}
		dir, base := path.Split(name)
		dir = strings.TrimSuffix(dir, "/")
		if dir == "" {
			dir = "."
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(path.Ext(name))
		switch ext {
		case ".json":
			md, err := fshelper.ReadJSON[GoogleMetaData](fsys, name)
			if err == nil {
				switch {
				case md.isAlbum():
					to.albumsByDir[dir] = browser.LocalAlbum{
						Path: path.Base(dir),
						Name: md.Title,
					}
					to.conf.Journal.AddEntry(name, journal.METADATA, "Album title: "+md.Title)
				case md.Category != "":
					to.conf.Journal.AddEntry(name, journal.METADATA, "unknown json file")
					return nil
				default:
					key := jsonKey{
						year: md.PhotoTakenTime.Time().Year(),
						name: base,
					}
					if prevMD, exists := to.jsonByYear[key]; exists {
						prevMD.foundInPaths = append(prevMD.foundInPaths, dir)
						to.jsonByYear[key] = prevMD
					} else {
						md.foundInPaths = append(md.foundInPaths, dir)
						to.jsonByYear[key] = md
					}
					to.conf.Journal.AddEntry(name, journal.METADATA, "Title: "+md.Title)
				}
			} else {
				to.conf.Journal.AddEntry(name, journal.ERROR, err.Error())
			}
		default:
			to.conf.Journal.AddEntry(name, journal.SCANNED, "")
			if _, err := fshelper.MimeFromExt(ext); err != nil {
				to.conf.Journal.AddEntry(name, journal.UNSUPPORTED, "")
				return nil
			}

			if strings.Contains(name, "Failed Videos") {
				to.conf.Journal.AddEntry(name, journal.FAILED_VIDEO, "")
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return err
			}
			key := fileReference{fileKey: fileKey{name: base, size: info.Size()}}
			l := to.filesByDir[dir]
			l = append(l, key)
			to.filesByDir[dir] = l
		}
		return nil
	})
	to.log.OK("Scanning the Google Photos takeout completed.")
	return err
}

type matcherFn func(jsonName string, fileName string) bool

// matchers is a list of matcherFn from the most likely to be used to the least one
var matchers = []matcherFn{
	normalMatch,
	matchWithOneCharOmitted,
	matchVeryLongNameWithNumber,
	matchDuplicateInYear,
	matchEditedName,
	matchForgottenDuplicates,
}

// Browse gives back to the main program the list of assets with resolution of file name, album, dates...
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
// -   if there are several files with same original name, the first instance kept as it is, the next have a a sequence number.
//       File is renamed as IMG_1234(1).JPG and the JSON is renamed as IMG_1234.JPG(1).JSON
// -   of course those rules are likely to collide. They have to be applied from the most common to the least one.
// -   sometimes the file isn't in the same folder than the json... It can be found in Year's photos folder
//
// The duplicates files (same name, same length in bytes) found in the local source are discarded before been presented to the immich server.
//

func (to *Takeout) Browse(ctx context.Context) chan *browser.LocalAssetFile {
	c := make(chan *browser.LocalAssetFile)
	passed := map[fileKey]any{}
	go func() {
		defer close(c)

		jsonFile := gen.MapKeys(to.jsonByYear)
		sort.Slice(jsonFile, func(i, j int) bool {
			return to.jsonByYear[jsonFile[i]].foundInPaths[0] < to.jsonByYear[jsonFile[j]].foundInPaths[0]
		})

		// For the most common matcher to the least,
		// Check files that match each json files
		for _, matcher := range matchers {
			for _, k := range jsonFile {
				md := to.jsonByYear[k]
				assets := to.jsonAssets(k, md, matcher)

				for _, a := range assets {
					to.conf.Journal.AddEntry(a.FileName, journal.JSON, k.name)
					ext := path.Ext(a.FileName)
					if !to.conf.SelectExtensions.Include(ext) {
						to.conf.Journal.AddEntry(a.FileName, journal.DISCARDED, "because of select-type option")
						continue
					}
					if to.conf.ExcludeExtensions.Exclude(ext) {
						to.conf.Journal.AddEntry(a.FileName, journal.DISCARDED, "because of exclude-type option")
						continue
					}
					fk := fileKey{name: path.Base(a.FileName), size: int64(a.FileSize)}
					if _, exist := passed[fk]; !exist {
						passed[fk] = nil
						select {
						case <-ctx.Done():
							return
						default:
							c <- a
						}
					} else {
						to.conf.Journal.AddEntry(a.FileName, journal.LOCAL_DUPLICATE, fk.name)
					}
				}
			}
		}

		leftOver := 0
		for _, l := range to.filesByDir {
			for _, f := range l {
				if !f.taken {
					leftOver++
				}
			}
		}
		to.log.Error("%d files left over", leftOver)

	}()

	return c

}

// jsonAssets search assets that are linked to this JSON using the given matcher

func (to *Takeout) jsonAssets(key jsonKey, md *GoogleMetaData, matcher matcherFn) []*browser.LocalAssetFile {

	var list []*browser.LocalAssetFile

	yearDir := path.Join(path.Dir(md.foundInPaths[0]), fmt.Sprintf("Photos from %d", md.PhotoTakenTime.Time().Year()))

	jsonInYear := false
	paths := md.foundInPaths
	for _, d := range md.foundInPaths {
		if d == yearDir {
			jsonInYear = true
			break
		}
	}
	if !jsonInYear {
		paths = append(paths, yearDir)
	}

	// Search for the assets in folders where the JSON has been found
	for _, d := range paths {
		l := to.filesByDir[d]

		for i, f := range l {
			if !f.taken && matcher(key.name, f.name) {
				list = append(list, to.copyGoogleMDToAsset(md, path.Join(d, f.name), int(f.size)))
				l[i].taken = true
			}
		}
	}
	return list
}

// normalMatch
//
//	PXL_20230922_144936660.jpg.json
//	PXL_20230922_144936660.jpg
func normalMatch(jsonName string, fileName string) bool {
	base := strings.TrimSuffix(jsonName, path.Ext(jsonName))
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

func matchWithOneCharOmitted(jsonName string, fileName string) bool {
	base := strings.TrimSuffix(jsonName, path.Ext(jsonName))
	if strings.HasPrefix(fileName, base) {
		if fshelper.IsExtensionPrefix(path.Ext(base)) {
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
func matchVeryLongNameWithNumber(jsonName string, fileName string) bool {
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
func matchDuplicateInYear(jsonName string, fileName string) bool {
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

func matchEditedName(jsonName string, fileName string) bool {
	base := strings.TrimSuffix(jsonName, path.Ext(jsonName))
	ext := path.Ext(base)
	if ext != "" {
		if _, err := fshelper.MimeFromExt(ext); err == nil {
			base := strings.TrimSuffix(base, ext)
			fname := strings.TrimSuffix(fileName, path.Ext(fileName))
			return strings.HasPrefix(fname, base)
		}
	}
	return false
}

//TODO: This one interferes with matchVeryLongNameWithNumber

// matchForgottenDuplicates
// original_1d4caa6f-16c6-4c3d-901b-9387de10e528_.json
// original_1d4caa6f-16c6-4c3d-901b-9387de10e528_P.jpg
// original_1d4caa6f-16c6-4c3d-901b-9387de10e528_P(1).jpg

func matchForgottenDuplicates(jsonName string, fileName string) bool {
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

func (to *Takeout) copyGoogleMDToAsset(md *GoogleMetaData, filename string, length int) *browser.LocalAssetFile {
	// Change file's title with the asset's title and the actual file's extension
	title := md.Title
	titleExt := path.Ext(title)
	fileExt := path.Ext(filename)
	if titleExt != fileExt {
		title = strings.TrimSuffix(title, titleExt)
		titleExt = path.Ext(title)
		if titleExt != fileExt {
			title = strings.TrimSuffix(title, titleExt) + fileExt
		}
	}

	a := browser.LocalAssetFile{
		FileName:    filename,
		FileSize:    length,
		Title:       title,
		Description: md.Description,
		Altitude:    md.GeoDataExif.Altitude,
		Latitude:    md.GeoDataExif.Latitude,
		Longitude:   md.GeoDataExif.Longitude,
		Archived:    md.Archived,
		FromPartner: md.isPartner(),
		Trashed:     md.Trashed,
		DateTaken:   md.PhotoTakenTime.Time(),
		Favorite:    md.Favorited,
		FSys:        to.fsys,
	}
	for _, p := range md.foundInPaths {
		if album, exists := to.albumsByDir[p]; exists {
			a.Albums = append(a.Albums, album)
		}

	}
	return &a
}
