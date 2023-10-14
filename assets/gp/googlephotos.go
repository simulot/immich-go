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

func NewTakeout(ctx context.Context, fsys fs.FS) (*Takeout, error) {
	to := Takeout{
		fsys:            fsys,
		filesByDir:      map[string][]string{},
		jsonByDirByBase: map[string]map[string]*googleMetaData{},
		filesByKey:      map[key][]string{},
		albumsByDir:     map[string]string{},
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
		if dir == "" {
			dir = "."
		}
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

// checkJSONs search among JSON files to establish
// - Date of taken based on the JSON content
// - album
// - partner
// - archive

func (to *Takeout) checkJSONs(base string, paths []string) *assets.LocalAssetFile {
	a := assets.LocalAssetFile{
		// FileName: paths[0],
		FSys:  to.fsys,
		Title: base,
	}

	// Search for a suitable JSON
	for _, dup := range paths {
		d := path.Dir(dup)

		// Check if we have a the base.json file somewhere
		if md := to.jsonMatchInDir(d, base); md != nil {
			to.copyGoogleMDToAsset(&a, md)
		}

		if al, ok := to.albumsByDir[d]; ok {
			a.AddAlbum(al)
			if a.FileName == "" {
				a.FileName = path.Join(d, base)
			}
		} else {
			// whenever possible, peek the main dir for the image
			a.FileName = path.Join(d, base)
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
	if md, ok := list[jsonBase]; ok {
		return md
	}

	// may be the file name has been shortened by 1 char
	// json named like  verylong.jp.json
	jsonBase = base
	_, size := utf8.DecodeLastRuneInString(jsonBase)
	jsonBase = jsonBase[:len(jsonBase)-size] + ".json"
	if md, ok := list[jsonBase]; ok {
		return md
	}

	// may the base name without the extension is shorten by 1 char
	ext := path.Ext(base)
	jsonBase = strings.TrimSuffix(base, ext)
	_, size = utf8.DecodeLastRuneInString(jsonBase)
	jsonBase = jsonBase[:len(jsonBase)-size] + ".json"
	if md, ok := list[jsonBase]; ok {
		return md
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
