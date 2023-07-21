package immich

import (
	"errors"
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

// LocalAsset represent an asset to upload.
// It could represent a file on the local file system, or another source like a Google takeout archive

type LocalAsset struct {
	Fsys      fs.FS      // The system when the file reside
	ID        string     // The ID of the asset, used for duplicate detection
	Name      string     // File base name
	FileSize  int        // size in bytes, part of the ID
	DateTaken *time.Time // date of capture, used for filtering
	Mime      string     // mime type of the file
	Ext       string     // file name's extension
	Albums    []string   // List of albums
	Archived  bool       // Archived flag coming from Google takeout
	// XMP      string
	// Hash string
}

// LocalAssetIndex is the collection of local assets
// It allows efficient search by ID, Album

type LocalAssetCollection struct {
	fss        []fs.FS                  // list of FS to browse (coming from  command line options)
	assets     []*LocalAsset            // All assets
	bAssetID   map[string]*LocalAsset   // ID, should be unique for the session
	byAlbums   map[string][]*LocalAsset // collect assets that belong to each albums
	extensions map[string]int
}

// Number of indexed assets
func (lai LocalAssetCollection) Len() int {
	return len(lai.assets)
}

// Close opened file systems, like zip archives
func (lai *LocalAssetCollection) Close() error {
	var err error
	for _, fsys := range lai.fss {
		if fsys, ok := fsys.(Closer); ok {
			err = errors.Join(err, fsys.Close())
		}
	}
	return err
}

// IndexerOptionsFn is a function to change Indexer parameters
type IndexerOptionsFn func(*indexerOptions)

// Indexer options
type indexerOptions struct {
	dateRange    DateRange // Accepted range of capture date
	createAlbums bool      // CLI flag fore album creation
}

// Set the OptionRange
func OptionRange(r DateRange) func(o *indexerOptions) {
	return func(o *indexerOptions) {
		o.dateRange = r
	}
}

func (lai *LocalAssetCollection) List() []*LocalAsset { return lai.assets }

/*
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
*/

func LoadLocalAssets(fss []fs.FS, opts ...IndexerOptionsFn) (*LocalAssetCollection, error) {
	var options = indexerOptions{}
	options.dateRange.Set("")

	for _, f := range opts {
		f(&options)
	}

	localAssets := &LocalAssetCollection{
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
