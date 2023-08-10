package upcmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"immich-go/fshelper"
	"immich-go/immich"
	"immich-go/immich/assets"
	"immich-go/immich/logger"
	"io/fs"

	"github.com/google/uuid"
)

/*
Upload command handling
*/

type UpCmd struct {
	Immich *immich.ImmichClient // Immich client

	EndPoint               string           // Immich server address (http://<your-ip>:2283/api or https://<your-domain>/api)
	Key                    string           // API Key
	Recursive              bool             // Explore sub folders
	GooglePhotos           bool             // For reading Google Photos takeout files
	Delete                 bool             // Delete original file after import
	CreateAlbumAfterFolder bool             // Create albums for assets based on the parent folder or a given name
	ImportIntoAlbum        string           // All assets will be added to this album
	Import                 bool             // Import instead of upload
	DeviceUUID             string           // Set a device UUID
	Paths                  []string         // Path to explore
	DateRange              immich.DateRange // Set capture date range
	ImportFromAlbum        string           // Import assets from this albums
	CreateAlbums           bool             // Create albums when exists in the source
	KeepTrashed            bool             // Import trashed assets
	KeepPartner            bool             // Import partner's assets
	DryRun                 bool             // Display actions but don't change anything
	// OnLineAssets           *immich.StringList       // Keep track on published assets
	AssetIndex       *immich.AssetIndex       // List of assets present on the server
	deleteServerList []*immich.Asset          // List of server assets to remove
	deleteLocalList  []*assets.LocalAssetFile // List of local assets to remove
	mediaUploaded    int                      // Count uploaded medias
	mediaCount       int                      // Count of media on the source
	updateAlbums     map[string][]string      // track immich albums changes
	logger           *logger.Logger
}

func UploadCommand(ctx context.Context, ic *immich.ImmichClient, logger *logger.Logger, args []string) error {
	app, err := NewUpCmd(ctx, ic, logger, args)

	logger.Info("Get server's assets...")
	app.AssetIndex, err = app.Immich.GetAllAssets(nil)
	if err != nil {
		return err
	}
	logger.OK("%d assets on the server.", app.AssetIndex.Len())

	fsys, err := fshelper.OpenMultiFile(app.Paths...)
	if err != nil {
		return err
	}

	var browser assets.Browser

	switch {
	case app.GooglePhotos:
		logger.Info("Browsing google take out archive...")
		browser, err = app.ReadGoogleTakeOut(ctx, fsys)
	default:
		logger.Info("Browsing folder(s)...")
		browser, err = app.ExploreLocalFolder(ctx, fsys)
	}

	if err != nil {
		return err
	}

	if app.CreateAlbums || app.CreateAlbumAfterFolder || len(app.ImportFromAlbum) > 0 {
		logger.Info("Browsing local assets for findings albums")
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
				logger.Warning("%s: %q", err.Error(), a.FileName)
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

func (app *UpCmd) handleAsset(a *assets.LocalAssetFile) error {
	showCount := true
	defer func() {
		a.Close()
		if showCount {
			app.logger.Progress("%d media scanned", app.mediaCount)
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
			app.logger.Error("Can't get capture date of the file. File %q skiped", a.FileName)
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
		app.logger.Info("%s: %s", a.Title, advice.Message)
		app.UploadAsset(a)
		if app.Delete {
			app.deleteLocalList = append(app.deleteLocalList, a)
		}
	case immich.SmallerOnServer:
		app.logger.Info("%s: %s", a.Title, advice.Message)

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
			app.logger.Info("%s: %s", a.Title, advice.Message)
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

}

func NewUpCmd(ctx context.Context, ic *immich.ImmichClient, logger *logger.Logger, args []string) (*UpCmd, error) {
	var err error
	cmd := flag.NewFlagSet("generate", flag.ExitOnError)

	app := UpCmd{
		updateAlbums: map[string][]string{},
		logger:       logger,
		Immich:       ic,
	}
	cmd.BoolVar(&app.DryRun, "dry-run", false, "display actions but don't touch source or destination")
	cmd.BoolVar(&app.GooglePhotos, "google-photos", false, "Import GooglePhotos takeout zip files")
	cmd.BoolVar(&app.Delete, "delete", false, "Delete local assets after upload")
	cmd.BoolVar(&app.KeepTrashed, "keep-trashed", false, "Import also trashed items")
	cmd.BoolVar(&app.KeepPartner, "keep-partner", true, "Import also partner's items")
	cmd.BoolVar(&app.CreateAlbumAfterFolder, "create-album-folder", false, "Create albums for assets based on the parent folder or a given name")
	cmd.StringVar(&app.ImportIntoAlbum, "album", "", "All assets will be added to this album.")
	cmd.Var(&app.DateRange, "date", "Date of capture range.")
	cmd.StringVar(&app.ImportFromAlbum, "from-album", "", "Import only from this album")
	cmd.BoolVar(&app.CreateAlbums, "create-albums", true, "Create albums like there were in the source")
	err = cmd.Parse(args)
	if err != nil {
		return nil, err
	}
	app.Paths = cmd.Args()

	if len(app.Paths) == 0 {
		err = errors.Join(err, errors.New("Must specify at least one path for local assets"))
	}
	return &app, err

}

func (a *UpCmd) ReadGoogleTakeOut(ctx context.Context, fsys fs.FS) (assets.Browser, error) {
	a.Delete = false
	return assets.BrowseGooglePhotosAssets(fsys), nil
}

func (a *UpCmd) ExploreLocalFolder(ctx context.Context, fsys fs.FS) (assets.Browser, error) {
	return assets.BrowseLocalAssets(fsys), nil
}

func (app *UpCmd) UploadAsset(a *assets.LocalAssetFile) {
	var resp immich.AssetResponse
	app.logger.MessageContinue(logger.OK, "Uploading %q...", a.FileName)
	var err error
	if !app.DryRun {
		resp, err = app.Immich.AssetUpload(a)
		if err != nil {
			app.logger.MessageTerminate(logger.Error, "Error: %s", err)
			return
		}
	} else {
		resp.ID = uuid.NewString()
	}
	app.AssetIndex.AddLocalAsset(a)
	if !resp.Duplicate {
		app.mediaUploaded += 1
		if !app.DryRun {
			app.logger.OK("Done, total %d uploaded", app.mediaUploaded)
		} else {
			app.logger.OK("Skipped - dry run mode, total %d uploaded", app.mediaUploaded)

		}
	} else {
		app.logger.MessageTerminate(logger.Warning, "already exists on the server")
	}
	for _, al := range a.Album {
		app.AddToAlbum(resp.ID, al)
	}
}

func (app *UpCmd) AddToAlbum(ID string, album string) {
	if !app.CreateAlbums && !app.CreateAlbumAfterFolder && len(app.ImportIntoAlbum) == 0 {
		return
	}
	switch {
	case len(app.ImportIntoAlbum) > 0:
		l := app.updateAlbums[app.ImportIntoAlbum]
		l = append(l, ID)
		app.updateAlbums[app.ImportIntoAlbum] = l
	case len(app.ImportFromAlbum) > 0:
		l := app.updateAlbums[album]
		l = append(l, ID)
		app.updateAlbums[album] = l
	case app.CreateAlbums && len(album) > 0:
		l := app.updateAlbums[album]
		l = append(l, ID)
		app.updateAlbums[album] = l
	}
}

func (app *UpCmd) DeleteLocalAssets() error {
	app.logger.OK("%d local assets to delete.", len(app.deleteLocalList))

	for _, a := range app.deleteLocalList {
		if !app.DryRun {
			app.logger.Warning("delete file %q", a.Title)
			err := a.Remove()
			if err != nil {
				return err
			}
		} else {
			app.logger.Warning("file %q not delested, dry run mode", a.Title)
		}

	}
	return nil
}

func (app *UpCmd) DeleteServerAssets(ids []string) error {
	app.logger.Warning("%d server assets to delete.", len(ids))

	if !app.DryRun {
		_, err := app.Immich.DeleteAsset(ids)
		return err
	}
	app.logger.Warning("%d server assets to delete. skipped dry-run mode", len(ids))
	return nil
}

func (app *UpCmd) ManageAlbums() error {
	if len(app.updateAlbums) > 0 {
		serverAlbums, err := app.Immich.GetAllAlbums()
		if err != nil {
			return fmt.Errorf("can't get the album list from the server: %w", err)
		}
		for album, list := range app.updateAlbums {
			found := false
			for _, sal := range serverAlbums {
				if sal.AlbumName == album {
					found = true
					if !app.DryRun {
						app.logger.OK("Update the album %s", album)
						_, err := app.Immich.UpdateAlbum(sal.ID, list)
						if err != nil {
							return fmt.Errorf("can't update the album list from the server: %w", err)
						}
					} else {
						app.logger.OK("Update album %s skipped - dry run mode", album)
					}
				}
			}
			if found {
				continue
			}
			if !app.DryRun {
				app.logger.OK("Create the album %s", album)

				_, err := app.Immich.CreateAlbum(album, list)
				if err != nil {
					return fmt.Errorf("can't create the album list from the server: %w", err)
				}
			} else {
				app.logger.OK("Create the album %s skipped - dry run mode", album)
			}
		}
	}
	return nil
}
