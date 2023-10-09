package gp

import (
	"context"
	"immich-go/assets"
	"immich-go/helpers/fshelper"
	"io/fs"
	"path"
	"regexp"
	"strings"
	"unicode/utf8"
)

type Takeout struct {
	fsys            fs.FS
	filesByDir      map[string][]string                   // files name mapped by dir
	filesByKey      map[key][]string                      // files name mapped by key (base name + length)
	jsonByDirByBase map[string]map[string]*googleMetaData // json files per dir and title
	// assetsByKey     map[string]*GPAsset                   // assets mapped by key (name+date)
	albumsByDir map[string]string // album title mapped by dir
}

type key struct {
	name string
	size int64
}

type Album struct {
	Title string
}

// type GPAsset struct {
// 	*assets.LocalAssetFile
// 	dirs []string // Keep track of all dirs when have seen this asset
// }

func NewTakeout(ctx context.Context, fsys fs.FS) (*Takeout, error) {
	to := Takeout{
		fsys:            fsys,
		filesByDir:      map[string][]string{},
		jsonByDirByBase: map[string]map[string]*googleMetaData{},
		filesByKey:      map[key][]string{},
		albumsByDir:     map[string]string{},
		// assetsByKey:     map[string]*GPAsset{},
	}
	err := to.walk(ctx, fsys)

	return &to, err
}

// walk the given FS to collect images file names and metadata files
func (to *Takeout) walk(ctx context.Context, fsys fs.FS) error {
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
		if d.IsDir() {
			if base == "Failed Videos" {
				return fs.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(path.Ext(name))
		switch ext {
		case ".json":
			md, err := fshelper.ReadJSON[googleMetaData](fsys, name)
			if err == nil {
				if md.isAlbum() {
					to.albumsByDir[dir] = path.Base(dir)
				} else {
					l := to.jsonByDirByBase[dir]
					if l == nil {
						l = map[string]*googleMetaData{}
					}
					l[base] = md
					to.jsonByDirByBase[dir] = l
				}
			}
		default:
			// build the list of files by directory
			l := to.filesByDir[dir]
			to.filesByDir[dir] = append(l, base)

			// keep the list of directory where the asset is found
			info, err := d.Info()
			if err != nil {
				return err
			}
			key := key{name: strings.ToUpper(base), size: info.Size()}
			l = to.filesByKey[key]
			to.filesByKey[key] = append(l, name)
		}
		return nil
	})

	return err
}

func (to *Takeout) Browse(ctx context.Context) chan *assets.LocalAssetFile {
	c := make(chan *assets.LocalAssetFile)
	go func() {
		defer close(c)
	next:
		for k, f := range to.filesByKey {
			select {
			case <-ctx.Done():
				return
			default:
				name := f[0]
				base := path.Base(name)
				ext := path.Ext(name)
				if _, err := fshelper.MimeFromExt(ext); err != nil {
					continue next
				}
				a := to.checkJSONs(base, f)
				a.FileSize = int(k.size)
				c <- a
			}
		}
	}()

	return c

}

// checkJSONs search among JSON files, all related files to establish
// - album
// - partner
// - archive
//
// Json presence induces the related image belongs to given album / year
// But the related image can be absent from the album folder
func (to *Takeout) checkJSONs(base string, paths []string) *assets.LocalAssetFile {
	a := assets.LocalAssetFile{
		FileName: paths[0],
		FSys:     to.fsys,
	}
	a.Title = base

	// m := radical.FindAllStringSubmatch(base, 1)

	for _, dup := range paths {
		d := path.Dir(dup)

		// Check if we have a the base.json file somewhere
		if md := to.jsonMatchInDir(d, base); md != nil {
			to.copyGoogleMDToAsset(&a, md)
			continue
		}
		// }
	}

	// Check jsons from albums, image can be referenced but not present

	for d, al := range to.albumsByDir {
		if md := to.jsonMatchInDir(d, base); md != nil {
			a.AddAlbum(al)
		}
	}

	return &a
}

func (to *Takeout) jsonMatchInDir(dir, base string) *googleMetaData {
	jsonBase := base
	if numberedName.MatchString(jsonBase) {
		jsonBase = numberedName.ReplaceAllString(jsonBase, "$1$3$2")
	}
	jsonBase += ".json"

	list := to.jsonByDirByBase[dir]
	for k, v := range list {
		if k == jsonBase {
			return v
		}
	}

	// may be the file name has been shortened by 1 char
	jsonBase = strings.TrimSuffix(base, path.Ext(base))
	_, size := utf8.DecodeLastRuneInString(jsonBase)
	jsonBase = jsonBase[:len(jsonBase)-size] + ".json"

	for k, v := range list {
		if k == jsonBase {
			return v
		}
	}
	return nil
}

func (to *Takeout) copyGoogleMDToAsset(a *assets.LocalAssetFile, md *googleMetaData) {
	a.Title = md.Title
	a.Altitude = md.GeoDataExif.Altitude
	a.Latitude = md.GeoDataExif.Latitude
	a.Longitude = md.GeoDataExif.Longitude
	a.Archived = md.Archived
	a.FromPartner = md.isPartner()
	a.Trashed = md.Trashed
	a.DateTaken = md.PhotoTakenTime.Time()
}

var (
	numberedName = regexp.MustCompile(`(?m)(.*)(\(\d+\))(\.\w+)$`)
)
