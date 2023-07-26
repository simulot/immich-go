package main

import (
	"errors"
	"flag"
	"fmt"
	"immich-go/immich"
	"immich-go/immich/assets"
	"immich-go/immich/logger"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Application struct {
	EndPoint               string    // Immich server address (http://<your-ip>:2283/api or https://<your-domain>/api)
	Key                    string    // API Key
	Recursive              bool      // Explore sub folders
	GooglePhotos           bool      // For reading Google Photos takeout files
	Yes                    bool      // Assume Yes to all questions
	Delete                 bool      // Delete original file after import
	CreateAlbumAfterFolder bool      // Create albums for assets based on the parent folder or a given name
	ImportIntoAlbum        string    // All assets will be added to this album
	Import                 bool      // Import instead of upload
	DeviceUUID             string    // Set a device UUID
	Paths                  []string  // Path to explore
	DateRange              DateRange // Set capture date range
	ImportFromAlbum        string    // Import assets from this albums
	CreateAlbums           bool      // Create albums when exists in the source
	ReplaceInferiorAsset   bool      // When uploading replace server's inferior assets with the uploaded one
	KeepTrashed            bool      // Import trashed assets
	KeepPartner            bool      // Import partner's assets

	OnLineAssets     *immich.StringList       // Keep track on published assets
	Logger           logger.Logger            // Program's logger
	Immich           *immich.ImmichClient     // Immich client
	AssetIndex       *immich.AssetIndex       // List of assets present on the server
	deleteServerList []*immich.Asset          // List of server assets to remove
	deleteLocalList  []*assets.LocalAssetFile // List of local assets to remove
	mediaUploaded    int                      // Count uploaded medias
	mediaCount       int
	// serverAlbums     []immich.AlbumSimplified // Server's Albums
	updateAlbums map[string][]string // Local assets albums
}

func Initialize() (*Application, error) {
	deviceID, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	app := Application{
		Logger:       logger.Logger{},
		updateAlbums: map[string][]string{},
	}
	flag.StringVar(&app.EndPoint, "server", "", "Immich server address (http://<your-ip>:2283/api or https://<your-domain>/api)")
	flag.StringVar(&app.Key, "key", "", "API Key")

	flag.BoolVar(&app.GooglePhotos, "google-photos", false, "Import GooglePhotos takeout zip files")
	flag.BoolVar(&app.Delete, "delete", false, "Delete local assets after upload")
	flag.BoolVar(&app.KeepTrashed, "keep-trashed", false, "Import also trashed items")
	flag.BoolVar(&app.KeepPartner, "keep-partner", true, "Import also partner's items")

	// TODO flag.BoolVar(&app.Recursive, "recursive", false, "Recursive")
	// TODO KeepArchived
	flag.BoolVar(&app.Yes, "yes", true, "Assume yes on all interactive prompts")
	flag.BoolVar(&app.CreateAlbumAfterFolder, "create-album-folder", false, "Create albums for assets based on the parent folder or a given name")
	flag.StringVar(&app.ImportIntoAlbum, "album", "", "All assets will be added to this album.")
	flag.Var(&app.DateRange, "date", "Date of capture range.")
	flag.StringVar(&app.ImportFromAlbum, "from-album", "", "Import only from this album")
	flag.BoolVar(&app.CreateAlbums, "create-albums", true, "Create albums like there were in the source")

	// flag.BoolVar(&app.Import, "import", false, "Import instead of upload")
	flag.StringVar(&app.DeviceUUID, "device-uuid", deviceID, "Set a device UUID")
	flag.BoolVar(&app.ReplaceInferiorAsset, "replace-inferior", false, "When uploading replace server's inferior assets with the uploaded one")
	flag.Parse()
	app.Paths = flag.Args()

	if len(app.EndPoint) == 0 {
		err = errors.Join(err, errors.New("Must specify a server address"))
	}

	if len(app.Key) == 0 {
		err = errors.Join(err, errors.New("Must specify an API key"))
	}
	if len(app.Paths) == 0 {
		err = errors.Join(err, errors.New("Must specify at least one path"))
	}

	return &app, err

}

// DateRange represent the date range for capture date
type DateRange struct {
	After, Before         time.Time
	day, month, year, set bool
}

func (dr DateRange) String() string {
	if dr.day {
		return dr.After.Format("2006-01-02")
	} else if dr.month {
		return dr.After.Format("2006-01")
	} else if dr.year {
		return dr.After.Format("2006")
	}
	return dr.After.Format("2006-01-02") + "," + dr.Before.AddDate(0, 0, -1).Format("2006-01-02")
}

func (dr *DateRange) Set(s string) (err error) {
	dr.set = true
	switch len(s) {
	case 0:
		dr.Before = time.Date(999, 12, 31, 0, 0, 0, 0, time.UTC)
	case 4:
		dr.year = true
		dr.After, err = time.ParseInLocation("2006", s, time.UTC)
		if err == nil {
			dr.Before = dr.After.AddDate(1, 0, 0)
			return
		}
	case 7:
		dr.month = true
		dr.After, err = time.ParseInLocation("2006-01", s, time.UTC)
		if err == nil {
			dr.Before = dr.After.AddDate(0, 1, 0)
			return
		}
	case 10:
		dr.day = true
		dr.After, err = time.ParseInLocation("2006-01-02", s, time.UTC)
		if err == nil {
			dr.Before = dr.After.AddDate(0, 0, 1)
			return
		}
	case 21:
		dr.After, err = time.ParseInLocation("2006-01-02", s[:10], time.UTC)
		if err == nil {
			dr.Before, err = time.ParseInLocation("2006-01-02", s[11:], time.UTC)
			if err == nil {
				dr.Before = dr.Before.AddDate(0, 0, 1)
				return
			}
		}
	}
	dr.set = false
	return fmt.Errorf("invalid date range:%w", err)
}

func (dr DateRange) IsSet() bool { return dr.set }

func (dr DateRange) InRange(d time.Time) bool {
	if !dr.set || d.IsZero() {
		return true
	}
	//	--------------After----------d------------Before
	return (d.Compare(dr.After) >= 0 && dr.Before.Compare(d) > 0)
}

type StringSliceFlag []string

func (ss *StringSliceFlag) String() string {
	if ss == nil {
		return ""
	}
	return strings.Join(*ss, ",")
}

func (ss *StringSliceFlag) Set(value string) error {
	values := strings.Split(value, ",")
	for _, v := range values {
		*ss = append(*ss, v)
	}
	return nil
}

func (ss *StringSliceFlag) Get() interface{} {
	return []string(*ss)
}

func (ss StringSliceFlag) IsIn(v string) bool {
	for i := range ss {
		if ss[i] == v {
			return true
		}
	}
	return false
}

type dirRemoveFS struct {
	dir string
	fs.FS
}

func DirRemoveFS(name string) fs.FS {

	fsys := &dirRemoveFS{
		FS:  os.DirFS(name),
		dir: name,
	}

	return fsys
}
func (fsys dirRemoveFS) Remove(name string) error {
	return os.Remove(filepath.Join(fsys.dir, name))
}
