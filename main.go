package main

import (
	"archive/zip"
	"context"
	"fmt"
	"immich-go/immich"
	"immich-go/immich/assets"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ttacon/chalk"
)

var stripSpaces = regexp.MustCompile(`\s+`)

func main() {
	app, err := Initialize()
	if err != nil {
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
		app.Logger.Print(chalk.Red, err.Error(), chalk.ResetColor)
		os.Exit(1)
	}
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
	app.Logger.Println(chalk.Green, "Server status: OK", chalk.ResetColor)

	user, err := app.Immich.ValidateConnection()
	if err != nil {
		return err
	}
	app.Logger.Println(chalk.Green, "Connected, user:", user.Email, chalk.ResetColor)
	app.Logger.Println(chalk.Green, "Get server's assets...", chalk.ResetColor)

	app.AssetIndex, err = app.Immich.GetAllAssets(nil)
	if err != nil {
		return err
	}
	app.Logger.Println(chalk.Green, app.AssetIndex.Len(), "assets on the server.", chalk.ResetColor)

	fsys, err := app.OpenFSs()
	if err != nil {
		return err
	}

	var assetChan chan *assets.LocalAssetFile

	switch {
	case app.GooglePhotos:
		app.Logger.Println(chalk.Green, "Browswing google take out archive...", chalk.ResetColor)
		assetChan, err = app.ReadGoogleTakeOut(ctx, fsys)
	default:
		app.Logger.Println(chalk.Green, "Browswing folder(s)...", chalk.ResetColor)
		assetChan, err = app.ExploreLocalFolder(ctx, fsys)
	}

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
				return err
			}

		}
	}

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
					_, err := app.Immich.UpdateAlbum(sal.ID, list)
					if err != nil {
						return fmt.Errorf("can't update the album list from the server: %w", err)
					}
				}
			}
			if found {
				continue
			}
			_, err := app.Immich.CreateAlbum(album, list)
			if err != nil {
				return fmt.Errorf("can't create the album list from the server: %w", err)
			}
		}
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
	defer a.Close()

	if !app.KeepPartner && a.FromPartner {
		return nil
	}

	if !app.KeepTrashed && a.Trashed {
		return nil
	}

	if len(app.ImportFromAlbum) > 0 && a.Album != app.ImportFromAlbum {
		return nil
	}

	if app.DateRange.IsSet() {
		d, err := a.DateTaken()
		if err != nil {
			app.Logger.Println(chalk.Yellow, "Can't get capture date of the file. File skiped", a.FileName, err, chalk.ResetColor)
			return nil
		}
		if !app.DateRange.InRange(d) {
			return nil
		}
	}

	advice, _ := app.AssetIndex.ShouldUpload(a)
	switch advice.Advice {
	case immich.NotOnServer:
		app.Logger.Println(chalk.Green, a.FileName, advice.Message, chalk.ResetColor)
		app.UploadAsset(a)
		if app.Delete {
			app.deleteLocalList = append(app.deleteLocalList, a)
		}
	case immich.SmallerOnServer:
		app.Logger.Println(chalk.Green, a.FileName, advice.Message, chalk.ResetColor)
		app.UploadAsset(a)
		app.deleteServerList = append(app.deleteServerList, advice.ServerAsset)
		if app.Delete {
			app.deleteLocalList = append(app.deleteLocalList, a)
		}
	case immich.SameOnServer:
		app.Logger.Println(chalk.Blue, a.FileName, advice.Message, chalk.ResetColor)
		if app.Delete {
			app.deleteLocalList = append(app.deleteLocalList, a)
		}
	default:
		app.Logger.Println(chalk.Blue, a.FileName, advice.Message, chalk.ResetColor)

	}
	return nil
}

func (app *Application) UploadAsset(a *assets.LocalAssetFile) {
	resp, err := app.Immich.AssetUpload(a)

	if err != nil {
		app.Logger.Println(chalk.Yellow, "Can't upload file:", a.FileName, err, chalk.ResetColor)
		return
	}
	app.AssetIndex.AddLocalAsset(a)
	app.mediaCount.Add(1)
	app.Logger.Println(chalk.Green, filepath.Base(a.FileName), "uploaded.", app.mediaCount.Load(), chalk.ResetColor)

	switch {
	case len(app.ImportIntoAlbum) > 0:
		l := app.updateAlbums[app.ImportIntoAlbum]
		l = append(l, resp.ID)
		app.updateAlbums[app.ImportIntoAlbum] = l
	}
}

func (a *Application) ReadGoogleTakeOut(ctx context.Context, fsys fs.FS) (chan *assets.LocalAssetFile, error) {
	a.Delete = false
	return assets.BrowseGooglePhotos(ctx, fsys), nil
}

func (a *Application) ExploreLocalFolder(ctx context.Context, fsys fs.FS) (chan *assets.LocalAssetFile, error) {
	return assets.BrowseLocalAssets(ctx, fsys), nil
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
	return assets.NewMergedFS(fss), nil
}

func (app *Application) DeleteLocalAssets() error {
	app.Logger.Println(chalk.Yellow, len(app.deleteLocalList), "local assets to delete.", chalk.ResetColor)

	for _, a := range app.deleteLocalList {
		app.Logger.Println(chalk.Yellow, "delete", a.FileName, chalk.ResetColor)
		err := a.Remove()
		if err != nil {
			return err
		}
	}
	return nil
}

func (app *Application) DeleteServerAssets(ids []string) error {
	app.Logger.Println(chalk.Yellow, len(ids), "server assets to delete.", chalk.ResetColor)

	// _, err := app.Immich.DeleteAsset(ids)
	return nil

}
