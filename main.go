package main

import (
	"errors"
	"flag"
	"fmt"
	"immich-go/immich"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/ttacon/chalk"
)

var stripSpaces = regexp.MustCompile(`\s+`)

func main() {
	app := Application{
		Logger: log.New(os.Stdout, "", log.LstdFlags),
	}

	deviceID, err := os.Hostname()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	flag.StringVar(&app.EndPoint, "server", "", "Immich server address (http://<your-ip>:2283/api or https://<your-domain>/api)")
	flag.StringVar(&app.Key, "key", "", "API Key")

	flag.BoolVar(&app.Recursive, "recursive", false, "Recursive")
	flag.BoolVar(&app.Yes, "yes", true, "Assume yes on all interactive prompts")
	flag.BoolVar(&app.Delete, "delete", false, "Delete local assets after upload")
	flag.UintVar(&app.Threads, "threads", uint(runtime.NumCPU()), fmt.Sprintf("Amount of concurrent upload threads (default=%d)", runtime.NumCPU()))
	flag.StringVar(&app.Album, "album", "", "Create albums for assets based on the parent folder or a given name")
	// flag.BoolVar(&app.Import, "import", false, "Import instead of upload")
	flag.StringVar(&app.DeviceUUID, "device-uuid", deviceID, "Set a device UUID")
	flag.Parse()
	app.Paths = flag.Args()

	err = app.Run()
	if err != nil {
		app.Logger.Print(chalk.Red, err.Error(), chalk.ResetColor)
		os.Exit(1)
	}
}

type Application struct {
	EndPoint     string               // Immich server address (http://<your-ip>:2283/api or https://<your-domain>/api)
	Key          string               // API Key
	Recursive    bool                 // Explore sub folders
	Yes          bool                 // Assume Yes to all questions
	Delete       bool                 // Delete original file after import
	Threads      uint                 // Number of trheads
	Album        string               // Create albums for assets based on the parent folder or a given name
	Import       bool                 // Import instead of upload
	DeviceUUID   string               // Set a device UUID
	Paths        []string             // Path to explore
	OnLineAssets *immich.StringList   // Keep track on published assets
	Logger       *log.Logger          // Progam's logger
	Immich       *immich.ImmichClient // Immich client
	Worker       *Worker              // Worker to manage multitthead
}

func (app *Application) CheckParameters() error {
	var err error

	if len(app.EndPoint) == 0 {
		err = errors.Join(err, errors.New("Must specify a serveur address"))
	}

	if len(app.Key) == 0 {
		err = errors.Join(err, errors.New("Must specify an API key"))
	}
	if len(app.Paths) == 0 {
		err = errors.Join(err, errors.New("Must specify at least one path"))
	}

	return err
}

type localAsset struct {
	ID   string
	Path string
	Mime string
}

func (app *Application) Run() error {

	err := app.CheckParameters()
	if err != nil {
		return err
	}

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

	app.Logger.Println(chalk.Green, "Indexing assets...", chalk.ResetColor)
	app.OnLineAssets, err = app.Immich.GetUserAssetsByDeviceId(app.DeviceUUID)
	if err != nil {
		return err
	}
	localAssets := []localAsset{}
	depth := 0

	for _, p := range app.Paths {
		err = filepath.WalkDir(p, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if depth > 0 && !app.Recursive {
					return filepath.SkipDir
				}
				depth++
				return nil
			}
			info, err := d.Info()
			if err != nil {
				app.Logger.Println(chalk.Red, "can't stat %s: %s", path, err)
				return nil
			}
			id := stripSpaces.ReplaceAllString(filepath.Base(d.Name()+"-"+strconv.FormatInt(info.Size(), 10)), "")
			if app.OnLineAssets.Includes(id) {
				return nil
			}
			localAssets = append(localAssets, localAsset{
				Path: path,
				ID:   id,
			})
			return nil
		})
		if err != nil {
			return fmt.Errorf("can't parse %s: %w", p, err)
		}
	}

	if len(localAssets) == 0 {
		app.Logger.Println(chalk.Yellow, "No local assets found, exiting", chalk.ResetColor)
		return nil
	}

	app.Logger.Println(chalk.Green, "Indexing complete, found", len(localAssets), "local assets to upload", chalk.ResetColor)

	if !app.Yes {
		var s string
		fmt.Print("Do you want to start upload now? (y/n) ")
		fmt.Fscanf(os.Stdin, "%s", &s)
		if strings.ToUpper(s) != "Y" {
			return errors.New("Abort Upload Process")
		}
	}

	app.Worker = NewWorker(int(app.Threads))
	stop := app.Worker.Run()

	for _, a := range localAssets {
		if app.OnLineAssets.Includes(a.ID) {
			app.Logger.Println(chalk.Yellow, filepath.Base(a.Path), "is already uploaded", chalk.ResetColor)
			continue
		}
		app.Upload(a)
	}

	stop()
	return err
}

func (app *Application) Upload(a localAsset) {
	app.Worker.Push(func() {
		if app.OnLineAssets.Includes(a.ID) {
			app.Logger.Println(chalk.Yellow, filepath.Base(a.Path), "have been already uploaded", chalk.ResetColor)
			return
		}
		app.OnLineAssets.Push(a.ID)
		resp, err := app.Immich.AssetUpload(a.Path)

		if err != nil {
			if errors.Is(err, immich.LocalFileError(nil)) || errors.Is(err, immich.UnsupportedMedia(nil)) {
				app.Logger.Println(chalk.Yellow, "Can't upload file: %s", a.Path, err, chalk.ResetColor)
			} else {
				app.Logger.Println(chalk.Red, "Can't upload file:", a.Path, "\n", err, chalk.ResetColor)

			}

			return

		}
		app.Logger.Println(chalk.Green, filepath.Base(a.Path), "uploaded.", chalk.ResetColor)
		_ = resp
		if app.Delete {
			// TODO
		}
	})
}
