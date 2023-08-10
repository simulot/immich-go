package upcmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"immich-go/immich"
	"immich-go/immich/assets"
	"immich-go/immich/logger"
)

/*
Upload command handling
*/

type UpCmd struct {
	Immich *immich.ImmichClient // Immich client

	EndPoint               string                   // Immich server address (http://<your-ip>:2283/api or https://<your-domain>/api)
	Key                    string                   // API Key
	Recursive              bool                     // Explore sub folders
	GooglePhotos           bool                     // For reading Google Photos takeout files
	Delete                 bool                     // Delete original file after import
	CreateAlbumAfterFolder bool                     // Create albums for assets based on the parent folder or a given name
	ImportIntoAlbum        string                   // All assets will be added to this album
	Import                 bool                     // Import instead of upload
	DeviceUUID             string                   // Set a device UUID
	Paths                  []string                 // Path to explore
	DateRange              immich.DateRange         // Set capture date range
	ImportFromAlbum        string                   // Import assets from this albums
	CreateAlbums           bool                     // Create albums when exists in the source
	KeepTrashed            bool                     // Import trashed assets
	KeepPartner            bool                     // Import partner's assets
	DryRun                 bool                     // Display actions but don't change anything
	OnLineAssets           *immich.StringList       // Keep track on published assets
	Logger                 logger.Logger            // Program's logger
	AssetIndex             *immich.AssetIndex       // List of assets present on the server
	deleteServerList       []*immich.Asset          // List of server assets to remove
	deleteLocalList        []*assets.LocalAssetFile // List of local assets to remove
	mediaUploaded          int                      // Count uploaded medias
	mediaCount             int                      // Count of media on the source
	updateAlbums           map[string][]string      // track immich albums changes
}

func UploadCommand(ctx context.Context, ic *immich.ImmichClient) error {
	app, err := NewUpCmd(ctx, ic)

	app.Logger.Info("Get server's assets...")
	app.AssetIndex, err = app.Immich.GetAllAssets(nil)
	if err != nil {
		return err
	}
	app.Logger.OK("%d assets on the server.", app.AssetIndex.Len())

	fsys, err := app.OpenFSs()
	if err != nil {
		return err
	}

	var browser assets.Browser

	switch {
	case app.GooglePhotos:
		app.Logger.Info("Browswing google take out archive...")
		browser, err = app.ReadGoogleTakeOut(ctx, fsys)
	default:
		app.Logger.Info("Browswing folder(s)...")
		browser, err = app.ExploreLocalFolder(ctx, fsys)
	}

	if err != nil {
		return err
	}

	if app.CreateAlbums || app.CreateAlbumAfterFolder || len(app.ImportFromAlbum) > 0 {
		app.Logger.Info("Browsing local assets for findings albums")
		err = browser.BrowseAlbums(ctx)
		if err != nil {
			return err
		}
	}

	assetChan := browser.Browse(ctx)
assetLoop:
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case a, ok := <-assetChan:
			if !ok {
				break assetLoop
			}
			err = app.handleAsset(a)
			if err != nil {
				app.Logger.Warning(a.FileName, err.Error())
			}

		}
	}

	if app.CreateAlbums || app.CreateAlbumAfterFolder || len(app.ImportIntoAlbum) > 0 {
		app.ManageAlbums()
	}

	if len(app.deleteServerList) > 0 {
		ids := []string{}
		for _, da := range app.deleteServerList {
			ids = append(ids, da.ID)
		}
		err := app.DeleteServerAssets(ids)
		if err != nil {
			return fmt.Errorf("Can't delete server's assets: %w", err)
		}
	}

	if len(app.deleteLocalList) > 0 {
		err = app.DeleteLocalAssets()
	}
	return err
}

func (app *Application) handleAsset(a *assets.LocalAssetFile) error {
	showCount := true
	defer func() {
		a.Close()
		if showCount {
			app.Logger.Progress("%d media scanned", app.mediaCount)
		}
	}()
	app.mediaCount++

	if !app.KeepPartner && a.FromPartner {
		return nil
	}

	if !app.KeepTrashed && a.Trashed {
		return nil
	}

	if len(app.ImportFromAlbum) > 0 && !a.IsInAlbum(app.ImportFromAlbum) {
		return nil
	}

	if app.DateRange.IsSet() {
		d, err := a.DateTaken()
		if err != nil {
			app.Logger.Error("Can't get capture date of the file. File %q skiped", a.FileName)
			return nil
		}
		if !app.DateRange.InRange(d) {
			return nil
		}
	}

	advice, err := app.AssetIndex.ShouldUpload(a)
	if err != nil {
		return err
	}

	switch advice.Advice {
	case immich.NotOnServer:
		app.Logger.Info("%s: %s", a.Title, advice.Message)
		app.UploadAsset(a)
		if app.Delete {
			app.deleteLocalList = append(app.deleteLocalList, a)
		}
	case immich.SmallerOnServer:
		app.Logger.Info("%s: %s", a.Title, advice.Message)

		// add the superior asset into albums of the orginal asset
		for _, al := range advice.ServerAsset.Albums {
			a.AddAlbum(al.AlbumName)
		}
		app.UploadAsset(a)

		app.deleteServerList = append(app.deleteServerList, advice.ServerAsset)
		if app.Delete {
			app.deleteLocalList = append(app.deleteLocalList, a)
		}
	case immich.SameOnServer:
		// Set add the server asset into albums determined locally
		for _, al := range a.Album {
			app.AddToAlbum(advice.ServerAsset.ID, al)
		}
		if !advice.ServerAsset.JustUploaded {
			app.Logger.Info("%s: %s", a.Title, advice.Message)
			if app.Delete {
				app.deleteLocalList = append(app.deleteLocalList, a)
			}
		} else {
			return nil
		}
	case immich.BetterOnServer:
		// keep the server version but update albums
		for _, al := range a.Album {
			app.AddToAlbum(advice.ServerAsset.ID, al)
		}
	}
	showCount = false
	return nil

	return err

}

func NewUpCmd(ctx context.Context, ic *immich.ImmichClient) (*UpCmd, error) {
	var err error
	app := UpCmd{
		updateAlbums: map[string][]string{},
	}
	flag.BoolVar(&app.DryRun, "dry-run", false, "display actions but don't touch source or destination")
	flag.BoolVar(&app.GooglePhotos, "google-photos", false, "Import GooglePhotos takeout zip files")
	flag.BoolVar(&app.Delete, "delete", false, "Delete local assets after upload")
	flag.BoolVar(&app.KeepTrashed, "keep-trashed", false, "Import also trashed items")
	flag.BoolVar(&app.KeepPartner, "keep-partner", true, "Import also partner's items")
	flag.BoolVar(&app.CreateAlbumAfterFolder, "create-album-folder", false, "Create albums for assets based on the parent folder or a given name")
	flag.StringVar(&app.ImportIntoAlbum, "album", "", "All assets will be added to this album.")
	flag.Var(&app.DateRange, "date", "Date of capture range.")
	flag.StringVar(&app.ImportFromAlbum, "from-album", "", "Import only from this album")
	flag.BoolVar(&app.CreateAlbums, "create-albums", true, "Create albums like there were in the source")
	flag.Parse()
	app.Paths = flag.Args()

	if len(app.Paths) == 0 {
		err = errors.Join(err, errors.New("Must specify at least one path for local assets"))
	}
	return &app, err

}
