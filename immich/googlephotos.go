package immich

import (
	"context"
	"encoding/json"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// LoadGooglePhotosAssets collects and filters assets from multiple file systems (fs.FS) to create a LocalAssetCollection.
// The function scans all given file systems and processes JSON metadata files to extract relevant assets.
// Assets are filtered based on optional IndexerOptionsFn provided as variadic arguments.
//
// Parameters:
//   fss ([]fs.FS): A slice of file systems containing Google Photos takeout archives.
//   opts (...IndexerOptionsFn): Optional IndexerOptionsFn functions to apply specific filtering criteria.
//
// Returns:
//   *LocalAssetCollection: A pointer to the LocalAssetCollection containing the selected assets.
//   error: An error, if encountered during the scanning, reading, or processing of the archives.
//
// Note:
//   - The provided file systems (fss) should contain Google Photos takeout archives with metadata JSON files.

tupe


func LoadGooglePhotosAssets(ctx context.Context, []fs.FS, opts ...IndexerOptionsFn) (*LocalAssetCollection, error) {
	var options = indexerOptions{}
	options.dateRange.Set("")

	for _, f := range opts {
		f(&options)
	}

	jsons := []*googleMetaData{}
	extensions := map[string]int{}

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
				if ext != ".json" {
					extensions[ext] = extensions[ext] + 1
					return nil
				}

				md, err := readJSON[googleMetaData](fsys, name)
				if err == nil && md != nil && len(md.URL) > 0 {
					if md.GooglePhotosOrigin.FromPartnerSharing != nil || md.Trashed {
						return nil
					}
					if options.dateRange.InRange(md.PhotoTakenTime.Time()) {
						md.name = name
						jsons = append(jsons, md)
					}
				}
				return err
			})
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(jsons, func(i, j int) bool {
		return jsons[i].name < jsons[j].name
	})

	localAssets := &LocalAssetCollection{
		fss:        fss,
		assets:     []*LocalAsset{},
		bAssetID:   map[string]*LocalAsset{},
		byAlbums:   map[string][]*LocalAsset{},
		extensions: extensions,
	}

	for _, md := range jsons {
		ext := path.Ext(md.Title)
		switch ext {
		case ".jpg", "jpeg", ".png", ".mp4", ".heic", ".mov", ".m4v", ".gif":
			name := path.Join(path.Dir(md.name), md.Title)
			var s fs.FileInfo
			var fsys fs.FS
			var err error

			for _, fsys = range fss {
				s, err = fs.Stat(fsys, name)
				if err == nil {
					break
				}
			}
			if err != nil {
				continue
			}

			size := s.Size()
			album := ""
			ID := md.Title + "-" + strconv.Itoa(int(size))
			folder := path.Base(path.Dir(name))
			if !gp.MatchString(folder) {
				album = folder
			}

			var a *LocalAsset
			var ok bool

			if a, ok = localAssets.bAssetID[ID]; !ok {
				// a new one
				a = &LocalAsset{
					Fsys:     fsys,
					ID:       ID,
					Name:     name,
					FileSize: int(size),
					Ext:      ext,
				}
				a.DateTaken = md.PhotoTakenTime.Time()
				localAssets.assets = append(localAssets.assets, a)
				localAssets.bAssetID[ID] = a
			}

			if len(album) > 0 {
				a.Albums = append(a.Albums, album)
				l := localAssets.byAlbums[album]
				l = append(l, a)
				localAssets.byAlbums[album] = l
			}
			a.Archived = a.Archived || md.Archived
		}
	}

	return localAssets, nil
}

var gp = regexp.MustCompile(`Photos from \d{4}`)

// readJSON reads a JSON file from the provided file system (fs.FS) with the given name and unmarshals it into the provided type T.
//
// Parameters:
//   FSys (fs.FS): The file system to read the JSON file from.
//   name (string): The name or path of the JSON file to be read.
//
// Type Parameters:
//   T: The type into which the JSON data will be unmarshaled. It should be a pointer to the target object.
//
// Returns:
//   *T: A pointer to the object of type T with the unmarshaled JSON data.
//   error: An error, if any, encountered during the read or unmarshal process.
//
// Note:
//   - Ensure that the file system (FSys) contains the JSON file at the specified name or path.
//   - The provided type T should be a pointer to the object that matches the structure of the JSON data.
//   - Any existing data in the object will be overwritten during unmarshaling.

func readJSON[T any](FSys fs.FS, name string) (*T, error) {
	var object T
	b, err := fs.ReadFile(FSys, name)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(b, &object)
	if err != nil {
		return nil, err
	}

	return &object, nil
}

type googleMetaData struct {
	Title string `json:"title"`
	// Description        string         `json:"description"`
	// ImageViews         string         `json:"imageViews"`
	// CreationTime       googTimeObject `json:"creationTime"`
	PhotoTakenTime googTimeObject `json:"photoTakenTime"`
	// GeoData            googGeoData    `json:"geoData"`
	// GeoDataExif        googGeoData    `json:"geoDataExif"`
	Trashed            bool   `json:"trashed,omitempty"`
	Archived           bool   `json:"archived,omitempty"`
	URL                string `json:"url"`
	GooglePhotosOrigin struct {
		MobileUpload struct {
			DeviceFolder struct {
				LocalFolderName string `json:"localFolderName"`
			} `json:"deviceFolder"`
			DeviceType string `json:"deviceType"`
		} `json:"mobileUpload"`
		FromPartnerSharing *struct {
		} `json:"fromPartnerSharing"`
	} `json:"googlePhotosOrigin"`

	// Additional fields
	name string
}

// type googGeoData struct {
// 	Latitude      float64 `json:"latitude"`
// 	Longitude     float64 `json:"longitude"`
// 	Altitude      float64 `json:"altitude"`
// 	LatitudeSpan  float64 `json:"latitudeSpan"`
// 	LongitudeSpan float64 `json:"longitudeSpan"`
// }

type googTimeObject struct {
	Timestamp int64 `json:"timestamp"`
	// Formatted string    `json:"formatted"`
}

func (gt googTimeObject) Time() *time.Time {
	t := time.Unix(gt.Timestamp, 0)
	t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC)
	return &t
}

func (t *googTimeObject) UnmarshalJSON(data []byte) error {
	type Alias googTimeObject
	aux := &struct {
		Timestamp string `json:"timestamp"`
		*Alias
	}{
		Alias: (*Alias)(t),
	}

	err := json.Unmarshal(data, &aux)
	if err != nil {
		return err
	}

	t.Timestamp, err = strconv.ParseInt(aux.Timestamp, 10, 64)

	return err
}
