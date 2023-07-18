package immich

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// LoadGooglePhotosAssets
func LoadGooglePhotosAssets(fss []fs.FS, opts ...IndexerOptionsFn) (*LocalAssetIndex, error) {
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

	localAssets := &LocalAssetIndex{
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
					Fsys:      fsys,
					ID:        ID,
					Name:      name,
					FileSize:  int(size),
					ModTime:   s.ModTime(),
					DateTaken: md.PhotoTakenTime.Time(),
					Ext:       ext,
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

type gpTimeStamp uint64

func (t gpTimeStamp) Time() time.Time {
	return time.Unix(int64(t), 0)
}

func (t *gpTimeStamp) UnmarshalJSON(b []byte) error {
	var i int64
	if len(b) < 2 {
		return errors.New("invalid timestamp")
	}
	b = b[1 : len(b)-1]
	err := json.Unmarshal(b, &i)
	if err != nil {
		return fmt.Errorf("can't marshal timestamp: %w", err)
	}
	(*t) = gpTimeStamp(i)
	return nil
}

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

func (gt googTimeObject) Time() time.Time {
	t := time.Unix(gt.Timestamp, 0)
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC)
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

// type googleMetaData struct {
// 	Title        string `json:"title"`
// 	CreationTime struct {
// 		Timestamp gpTimeStamp `json:"timestamp"`
// 	} `json:"creationTime"`
// 	PhotoTakenTime struct {
// 		Timestamp gpTimeStamp `json:"timestamp"`
// 	} `json:"photoTakenTime"`
// 	GooglePhotosOrigin struct {
// 		MobileUpload *struct {
// 			DeviceType string `json:"deviceType"`
// 		} `json:"mobileUpload"`
// 		FromPartnerSharing *struct {
// 		} `json:"fromPartnerSharing"`
// 	} `json:"googlePhotosOrigin,omitempty"`
// 	Trashed  bool   `json:"trashed,omitempty"`
// 	Archived bool   `json:"archived,omitempty"`
// 	URL      string `json:"url,omitempty"`
// 	name     string
// }
