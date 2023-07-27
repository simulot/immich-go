package assets

import (
	"context"
	"encoding/json"
	"io/fs"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type GooglePhotosAssetBrowser struct {
	fs.FS
}

func BrowseGooglePhotosAssets(fsys fs.FS) *GooglePhotosAssetBrowser {
	return &GooglePhotosAssetBrowser{
		FS: fsys,
	}
}

// browseGooglePhotos collects and filters assets from a file systems (fs.FS) to create a channel of localFile.
// The function scans all given file systems and processes JSON metadata files to extract relevant assets.

func (fsys *GooglePhotosAssetBrowser) Browse(ctx context.Context) chan *LocalAssetFile {
	fileChan := make(chan *LocalAssetFile)

	// Start a goroutine to browse the FS and collect the list of files
	go func(ctx context.Context) {
		defer close(fileChan) // Close the channel when the goroutine finishes

		err := fs.WalkDir(fsys, ".",
			func(name string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}

				// Check if the context has been cancelled
				select {
				case <-ctx.Done():
					// If the context has been cancelled, return immediately
					return ctx.Err()
				default:
				}

				if d.IsDir() {
					return nil
				}

				if strings.ToLower(path.Ext(name)) != ".json" {
					return nil
				}

				md, err := readJSON[googleMetaData](fsys, name)
				if err == nil && md != nil && len(md.URL) > 0 {
					dir := path.Dir(name)
					ext := path.Ext(md.Title)
					base := strings.TrimSuffix(md.Title, ext)
					if len(base) > 47 {
						base = base[:47]
					}

					f := LocalAssetFile{
						srcFS:       fsys,
						FileName:    path.Join(dir, base+ext),
						Title:       md.Title,
						Trashed:     md.Trashed,
						FromPartner: md.GooglePhotosOrigin.FromPartnerSharing != nil,
						dateTaken:   md.PhotoTakenTime.Time(),
					}
					if !gp.MatchString(path.Dir(name)) {
						f.Album = commaAlbum.ReplaceAllString(path.Base(path.Dir(name)), "")
					}

					// Check if the context has been cancelled before sending the file
					select {
					case <-ctx.Done():
						// If the context has been cancelled, return immediately
						return ctx.Err()
					case fileChan <- &f:
					}
				}
				return nil // ignore json errors...
			})

		if err != nil {
			// Check if the context has been cancelled before sending the error
			select {
			case <-ctx.Done():
				// If the context has been cancelled, return immediately
				return
			case fileChan <- &LocalAssetFile{
				Err: err,
			}:
			}
		}
	}(ctx)

	return fileChan
}

var gp = regexp.MustCompile(`Photos from \d{4}`)
var commaAlbum = regexp.MustCompile(`^,\s+`)

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
	t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC)
	return t
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

// readJSON reads a JSON file from the provided file system (fs.FS)
// with the given name and unmarshals it into the provided type T.

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
