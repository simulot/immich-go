package immich

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"io"
	"io/fs"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type LocalFileError error
type Closer interface {
	Close() error
}

// LocalAsset represent an asset
type LocalAsset struct {
	Fsys      fs.FS
	ID        string
	Name      string
	FileSize  int
	ModTime   time.Time
	DateTaken time.Time
	Mime      string
	Ext       string
	Albums    []string
	Archived  bool
	// XMP      string
	Hash string
}

/*
	func (la *LocalAsset) OpenFile() (fs.File, error) {
		f, err := la.Fsys.Open(la.Name)
		if err != nil {
			return nil, err
		}
		return &localAssetFile{
			f: f,
		}, nil
	}

	type localAssetFile struct {
		f         fs.File
		preloaded []byte
		pos       int
	}

	func (f *localAssetFile) PreLoad(length int) ([]byte, error) {
		if len(f.preloaded) <= length {
			b := make([]byte, 0, length-len(f.preloaded))
			_, err := f.f.Read(b)
			if err != nil {
				return nil, err
			}
			f.preloaded = append(f.preloaded, b...)
		}
		return f.preloaded[:length], nil
	}

	func (f *localAssetFile) Close() error {
		return f.f.Close()
	}

	func (f *localAssetFile) Stat() (fs.FileInfo, error) {
		return f.f.Stat()
	}

	func (f *localAssetFile) Read(b []byte) (int, error) {
		if f.pos < len(f.preloaded) {
			n := copy(b, f.preloaded[f.pos:])
			f.pos += n
			return n, nil
		}
		r, err := f.f.Read(b)
		f.pos += r
		return r, err
	}
*/

type LocalAssetIndex struct {
	totalAssets int
	fss         []fs.FS
	assets      []*LocalAsset
	bAssetID    map[string]*LocalAsset
	byAlbums    map[string][]*LocalAsset
	extensions  map[string]int
}

func (lai LocalAssetIndex) Len() int {
	return len(lai.assets)
}

func (lai *LocalAssetIndex) Close() error {
	var err error
	for _, fsys := range lai.fss {
		if fsys, ok := fsys.(Closer); ok {
			err = errors.Join(err, fsys.Close())
		}
	}
	return err
}

type IndexerOptionsFn func(*indexerOptions)
type indexerOptions struct {
	dateRange    DateRange
	createAlbums bool
	albums       map[string]any
}

func OptionRange(r DateRange) func(o *indexerOptions) {
	return func(o *indexerOptions) {
		o.dateRange = r
	}
}
func OptionCreateAlbum(album string) func(o *indexerOptions) {
	return func(o *indexerOptions) {
		if o.albums == nil {
			o.albums = map[string]any{}
		}
		o.albums[album] = nil
		o.createAlbums = true
	}
}

func (lai *LocalAssetIndex) List() []*LocalAsset { return lai.assets }

func NewLocalAsset(fsys fs.FS, name string) (*LocalAsset, error) {
	f, err := fsys.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}

	asset := LocalAsset{
		Fsys:     fsys,
		Name:     name,
		FileSize: int(info.Size()),
		ModTime:  info.ModTime(),
	}
	b4k := &bytes.Buffer{}
	_, err = io.CopyN(b4k, f, 4*1024)
	if err != nil {
		return nil, err
	}

	asset.Mime, err = GetMimeType(b4k.Bytes())
	if err != nil {
		return nil, err
	}

	mr := io.MultiReader(b4k, f)

	hash := sha1.New()
	_, err = io.Copy(hash, mr)
	if err != nil {
		return nil, err
	}

	asset.Hash = base64.StdEncoding.EncodeToString(hash.Sum(nil))
	asset.ID = filepath.Base(name) + "-" + asset.Hash
	return &asset, nil
}

func LoadLocalAssets(fss []fs.FS, opts ...IndexerOptionsFn) (*LocalAssetIndex, error) {
	var options = indexerOptions{}
	options.dateRange.Set("")

	for _, f := range opts {
		f(&options)
	}

	localAssets := &LocalAssetIndex{
		fss:        fss,
		assets:     []*LocalAsset{},
		bAssetID:   map[string]*LocalAsset{},
		byAlbums:   map[string][]*LocalAsset{},
		extensions: map[string]int{},
	}

	// Browse all given FS to collect the list of files
	for _, fsys := range fss {
		err := fs.WalkDir(fsys, ".",
			func(name string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					return nil
				}
				ext := strings.ToLower(path.Ext(name))
				switch ext {
				case ".jpg", "jpeg", ".png", ".mp4", ".heic", ".mov", ".m4v", ".gif":

					s, err := d.Info()
					if err != nil {
						return err
					}
					size := s.Size()
					ID := filepath.Base(name) + "-" + strconv.Itoa(int(size))
					album := path.Base(path.Dir(name))

					var a *LocalAsset
					var ok bool

					if a, ok = localAssets.bAssetID[ID]; !ok {
						// a new one
						dateTaken, err := ExtractDateTaken(fsys, name)
						if err != nil {
							return err
						}

						if options.dateRange.InRange(dateTaken) {
							a = &LocalAsset{
								Fsys:      fsys,
								ID:        ID,
								Name:      name,
								FileSize:  int(size),
								ModTime:   s.ModTime(),
								DateTaken: dateTaken,
								Ext:       ext,
							}
							localAssets.assets = append(localAssets.assets, a)
							localAssets.bAssetID[ID] = a
						}
					}

					if a != nil && len(album) > 0 {
						a.Albums = append(a.Albums, album)
						l := localAssets.byAlbums[album]
						l = append(l, a)
						localAssets.byAlbums[album] = l
					}
				default:
					localAssets.extensions[ext] = localAssets.extensions[ext] + 1
				}
				return nil
			})
		if err != nil {
			return nil, err
		}
	}
	return localAssets, nil
}
