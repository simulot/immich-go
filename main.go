package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"immich-go/immich"
	"immich-go/immich/assets"
	"immich-go/immich/logger"
	"io/fs"
	"os"
	"os/signal"

	"github.com/google/uuid"
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
		err = Run(ctx)
	}
	if err != nil {
		app.Logger.Error(err.Error())
		os.Exit(1)
	}
	app.Logger.OK("Done.")
}

type Application struct {
	EndPoint   string               // Immich server address (http://<your-ip>:2283/api or https://<your-domain>/api)
	Key        string               // API Key
	DeviceUUID string               // Set a device UUID
	Immich     *immich.ImmichClient // Immich client
	Logger     logger.Logger        // Program's logger

}

func Run(ctx context.Context) error {
	var err error
	deviceID, err := os.Hostname()
	if err != nil {
		return err
	}

	app := Application{
		Logger: *logger.NewLogger(logger.OK),
	}
	flag.StringVar(&app.EndPoint, "server", "", "Immich server address (http://<your-ip>:2283 or https://<your-domain>)")
	flag.StringVar(&app.Key, "key", "", "API Key")
	flag.StringVar(&app.DeviceUUID, "device-uuid", deviceID, "Set a device UUID")

	flag.Parse()
	if len(app.EndPoint) == 0 {
		err = errors.Join(err, errors.New("Must specify a server address"))
	}

	if len(app.Key) == 0 {
		err = errors.Join(err, errors.New("Must specify an API key"))
	}

	if len(flag.Args()) == 0 {
		err = errors.Join(err, errors.New("Missing command"))
	}

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

	cmd := flag.Args()[0]
	switch cmd {
	case "upload":
		err = UploadCommand(ctx, app.Immich)
	default:
		err = fmt.Errorf("unknwon command: %q", cmd)
	}

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
