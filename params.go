package main

import (
	"errors"
	"flag"
	"immich-go/immich"
	"log"
	"os"
	"sync/atomic"
)

type Application struct {
	EndPoint             string           // Immich server address (http://<your-ip>:2283/api or https://<your-domain>/api)
	Key                  string           // API Key
	Recursive            bool             // Explore sub folders
	GooglePhotos         bool             // For reading Google Photos takeout files
	Yes                  bool             // Assume Yes to all questions
	Delete               bool             // Delete original file after import
	Album                string           // Create albums for assets based on the parent folder or a given name
	Import               bool             // Import instead of upload
	DeviceUUID           string           // Set a device UUID
	Paths                []string         // Path to explore
	DateRange            immich.DateRange // Set capture date range
	ReplaceInferiorAsset bool             // When uploading replace server's inferior assets with the uploaded one

	OnLineAssets        *immich.StringList   // Keep track on published assets
	Logger              *log.Logger          // Program's logger
	Immich              *immich.ImmichClient // Immich client
	AssetIndex          *immich.AssetIndex   // List of assets present on the server
	DeleteList          []*immich.Asset      // List of assets to be removed
	mediaCount          atomic.Int64         // Count uploaded medias
	tooManyServerErrors chan any             // Signal of permanent server error condition

}

func Start() (*Application, error) {
	deviceID, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	app := Application{
		Logger: log.New(os.Stdout, "", log.LstdFlags),
	}
	app.DateRange.Set("")

	flag.StringVar(&app.EndPoint, "server", "", "Immich server address (http://<your-ip>:2283/api or https://<your-domain>/api)")
	flag.StringVar(&app.Key, "key", "", "API Key")

	flag.BoolVar(&app.GooglePhotos, "GooglePhotos", false, "Import GooglePhotos takeout zip files")
	flag.BoolVar(&app.Recursive, "recursive", false, "Recursive")
	flag.BoolVar(&app.Yes, "yes", true, "Assume yes on all interactive prompts")
	flag.BoolVar(&app.Delete, "delete", false, "Delete local assets after upload")
	flag.StringVar(&app.Album, "album", "", "Create albums for assets based on the parent folder or a given name")
	flag.Var(&app.DateRange, "date", "Date of capture range.")
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
