package main

import (
	"archive/zip"
	"errors"
	"fmt"
	"immich-go/immich"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ttacon/chalk"
)

var stripSpaces = regexp.MustCompile(`\s+`)

func main() {
	app, err := Start()
	if err != nil {
		os.Exit(1)
	}

	err = app.Run()
	if err != nil {
		app.Logger.Print(chalk.Red, err.Error(), chalk.ResetColor)
		os.Exit(1)
	}
}

func (app *Application) Run() error {
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
	app.Logger.Println(chalk.Green, app.AssetIndex.Len(), "assets on the server", chalk.ResetColor)

	var LocalAssetIndex *immich.LocalAssetIndex
	switch {
	case app.GooglePhotos:
		app.Logger.Println(chalk.Green, "Scanning google take out archive...", chalk.ResetColor)
		LocalAssetIndex, err = app.ReadGoogleTakeOut()
	default:
		LocalAssetIndex, err = app.ExploreLocalFolder()
	}
	if err != nil {
		return err
	}

	defer LocalAssetIndex.Close()
	if LocalAssetIndex.Len() == 0 {
		app.Logger.Println(chalk.Yellow, "No local assets found, exiting.", chalk.ResetColor)
		return nil
	}

	app.Logger.Println(chalk.Green, "Local scan completed, found", LocalAssetIndex.Len(), "assets.", chalk.ResetColor)

	if !app.Yes {
		var s string
		fmt.Print("Do you want to start upload now? (y/n) ")
		fmt.Fscanf(os.Stdin, "%s", &s)
		if strings.ToUpper(s) != "Y" {
			return errors.New("Abort Upload Process")
		}
	}

	for _, a := range LocalAssetIndex.List() {
		advice, _ := app.AssetIndex.ShouldUpload(a)
		app.Logger.Println(a.Name, advice.Message)
		switch advice.Advice {
		case immich.NotOnServer:
			app.UploadAsset(a)
		case immich.SmallerOnServer:
			app.UploadAsset(a)
			app.DeleteList = append(app.DeleteList, advice.ServerAsset)
		}

		// if app.OnLineAssets.Includes(a.ID) {
		// 	app.Logger.Println(chalk.Yellow, filepath.Base(a.Name), "is already uploaded", chalk.ResetColor)
		// 	continue
		// }
	}

	if len(app.DeleteList) > 0 {
		ids := []string{}
		for _, da := range app.DeleteList {
			ids = append(ids, da.ID)
		}
		app.DeleteAssets(ids)
	}
	return err
}

func (app *Application) UploadAsset(a *immich.LocalAsset) {
	resp, err := app.Immich.AssetUpload(a.Fsys, a.Name)

	if err != nil {
		app.Logger.Println(chalk.Yellow, "Can't upload file:", a.Name, err, chalk.ResetColor)
		// if errors.Is(err, immich.LocalFileError(nil)) || errors.Is(err, &immich.UnsupportedMedia{}) {
		// } else if errors.Is(err, &immich.TooManyInternalError{}) {
		// 	close(app.tooManyServerErrors)
		// } else {
		// 	app.Logger.Println(chalk.Red, "Can't upload file:", a.Name)
		// 	app.Logger.Println(chalk.Red, err, chalk.ResetColor)
		// }
		return
	}

	app.mediaCount.Add(1)
	app.Logger.Println(chalk.Green, filepath.Base(a.Name), "uploaded.", app.mediaCount.Load(), chalk.ResetColor)
	_ = resp
	if app.Delete {
		// TODO
	}
}

func (app *Application) DeleteAssets(ids []string) {
	app.Logger.Println(chalk.Yellow, len(ids), "asset to delete.", chalk.ResetColor)

	_, err := app.Immich.DeleteAsset(ids)

	if err != nil {
		app.Logger.Println(chalk.Yellow, "Can't delete assets", err, chalk.ResetColor)
		// if errors.Is(err, immich.LocalFileError(nil)) || errors.Is(err, &immich.UnsupportedMedia{}) {
		// } else if errors.Is(err, &immich.TooManyInternalError{}) {
		// 	close(app.tooManyServerErrors)
		// } else {
		// 	app.Logger.Println(chalk.Red, "Can't upload file:", a.Name)
		// 	app.Logger.Println(chalk.Red, err, chalk.ResetColor)
		// }
		return
	}

}

func (a *Application) ReadGoogleTakeOut() (*immich.LocalAssetIndex, error) {
	fss, err := a.listFS()
	if err != nil {
		return nil, err
	}
	return immich.LoadGooglePhotosAssets(fss, immich.OptionRange(a.DateRange), immich.OptionCreateAlbum(a.Album))
}

func (a *Application) ExploreLocalFolder() (*immich.LocalAssetIndex, error) {
	fss, err := a.listFS()
	if err != nil {
		return nil, err
	}
	return immich.LoadLocalAssets(fss, immich.OptionRange(a.DateRange), immich.OptionCreateAlbum(a.Album))
}

func (a *Application) listFS() ([]fs.FS, error) {
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
			fsys := os.DirFS(p)
			fss = append(fss, fsys)
		}
	}
	return fss, nil
}
