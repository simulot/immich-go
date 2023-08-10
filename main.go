package main

import (
	"archive/zip"
	"context"
	"fmt"
	"immich-go/immich"
	"immich-go/immich/assets"
	"immich-go/immich/logger"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/yalue/merged_fs"
)

func main() {
	app, err := Initialize()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Create a context with cancel function to gracefully handle Ctrl+C events
	ctx, cancel := context.WithCancel(context.Background())

	// Handle Ctrl+C signal (SIGINT)
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt)

	go func() {
		<-signalChannel
		fmt.Println("\nCtrl+C received. Gracefully shutting down...")
		cancel() // Cancel the context when Ctrl+C is received
	}()

	select {
	case <-ctx.Done():
		err = ctx.Err()
	default:
		err = app.Run(ctx)
	}
	if err != nil {
		app.Logger.Error(err.Error())
		os.Exit(1)
	}
	app.Logger.OK("Done.")
}

func (app *Application) Run(ctx context.Context) error {
	var err error
	app.Immich, err = immich.NewImmichClient(app.EndPoint, app.Key, app.DeviceUUID)
	if err != nil {
		return err
	}

	err = app.Immich.PingServer()
	if err != nil {
		return err
	}
	app.Logger.OK("Server status: OK")

	user, err := app.Immich.ValidateConnection()
	if err != nil {
		return err
	}
	app.Logger.Info("Connected, user: %s", user.Email)
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
}

func (app *Application) UploadAsset(a *assets.LocalAssetFile) {
	var resp immich.AssetResponse
	app.Logger.MessageContinue(logger.OK, "Uploading %q...", a.FileName)
	var err error
	if !app.DryRun {
		resp, err = app.Immich.AssetUpload(a)
		if err != nil {
			app.Logger.MessageTerminate(logger.Error, "Error: %s", err)
			return
		}
	} else {
		resp.ID = uuid.NewString()
	}
	app.AssetIndex.AddLocalAsset(a)
	if !resp.Duplicate {
		app.mediaUploaded += 1
		if !app.DryRun {
			app.Logger.OK("Done, total %d uploaded", app.mediaUploaded)
		} else {
			app.Logger.OK("Skipped - dry run mode, total %d uploaded", app.mediaUploaded)

		}
	} else {
		app.Logger.MessageTerminate(logger.Warning, "already exists on the server")
	}
	for _, al := range a.Album {
		app.AddToAlbum(resp.ID, al)
	}
}

func (app *Application) AddToAlbum(ID string, album string) {
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

func (a *Application) ReadGoogleTakeOut(ctx context.Context, fsys fs.FS) (assets.Browser, error) {
	a.Delete = false
	return assets.BrowseGooglePhotosAssets(fsys), nil
}

func (a *Application) ExploreLocalFolder(ctx context.Context, fsys fs.FS) (assets.Browser, error) {
	return assets.BrowseLocalAssets(fsys), nil
}

func (a *Application) OpenFSs() (fs.FS, error) {
	fss := []fs.FS{}

	for _, p := range a.Paths {
		s, err := os.Stat(p)
		if err != nil {
			return nil, err
		}

		switch {
		case !s.IsDir() && strings.ToLower(filepath.Ext(s.Name())) == ".zip":
			fsys, err := zip.OpenReader(p)
			if err != nil {
				return nil, err
			}
			fss = append(fss, fsys)
		default:
			fsys := DirRemoveFS(p)
			fss = append(fss, fsys)
		}
	}
	return merged_fs.MergeMultiple(fss...), nil

	// assets.NewMergedFS(fss), nil
}

func (app *Application) DeleteLocalAssets() error {
	app.Logger.OK("%d local assets to delete.", len(app.deleteLocalList))

	for _, a := range app.deleteLocalList {
		if !app.DryRun {
			app.Logger.Warning("delete file %q", a.Title)
			err := a.Remove()
			if err != nil {
				return err
			}
		} else {
			app.Logger.Warning("file %q not delested, dry run mode", a.Title)
		}

	}
	return nil
}

func (app *Application) DeleteServerAssets(ids []string) error {
	app.Logger.Warning("%d server assets to delete.", len(ids))

	if !app.DryRun {
		_, err := app.Immich.DeleteAsset(ids)
		return err
	}
	app.Logger.Warning("%d server assets to delete. skipped dry-run mode", len(ids))
	return nil
}

func (app *Application) ManageAlbums() error {
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
						app.Logger.OK("Update the album %s", album)
						_, err := app.Immich.UpdateAlbum(sal.ID, list)
						if err != nil {
							return fmt.Errorf("can't update the album list from the server: %w", err)
						}
					} else {
						app.Logger.OK("Update album %s skipped - dry run mode", album)
					}
				}
			}
			if found {
				continue
			}
			if !app.DryRun {
				app.Logger.OK("Create the album %s", album)

				_, err := app.Immich.CreateAlbum(album, list)
				if err != nil {
					return fmt.Errorf("can't create the album list from the server: %w", err)
				}
			} else {
				app.Logger.OK("Create the album %s skipped - dry run mode", album)
			}
		}
	}
	return nil
}
