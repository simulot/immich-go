// Command Upload

package upload

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"math"
	"path"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	"github.com/google/uuid"
	"github.com/simulot/immich-go/browser"
	"github.com/simulot/immich-go/browser/files"
	"github.com/simulot/immich-go/browser/gp"
	"github.com/simulot/immich-go/cmd"
	"github.com/simulot/immich-go/helpers/fshelper"
	"github.com/simulot/immich-go/helpers/gen"
	"github.com/simulot/immich-go/helpers/myflag"
	"github.com/simulot/immich-go/helpers/stacking"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/immich/metadata"
	"github.com/simulot/immich-go/logger"
	"golang.org/x/sync/errgroup"
)

/*
	TODO:
		browser should't  report non fatal errors
		Add  timeouts to http clients
		deprecate ForceSidecar
		pass supported medida to googlephotobrowser
*/

type UpCmd struct {
	*cmd.SharedFlags // shared flags and immich client

	args  []string
	fsyss []fs.FS // pseudo file system to browse

	GooglePhotos           bool             // For reading Google Photos takeout files
	Delete                 bool             // Delete original file after import
	CreateAlbumAfterFolder bool             // Create albums for assets based on the parent folder or a given name
	ImportIntoAlbum        string           // All assets will be added to this album
	PartnerAlbum           string           // Partner's assets will be added to this album
	Import                 bool             // Import instead of upload
	DeviceUUID             string           // Set a device UUID
	Paths                  []string         // Path to explore
	DateRange              immich.DateRange // Set capture date range
	ImportFromAlbum        string           // Import assets from this albums
	CreateAlbums           bool             // Create albums when exists in the source
	KeepTrashed            bool             // Import trashed assets
	KeepPartner            bool             // Import partner's assets
	KeepUntitled           bool             // Keep untitled albums
	UseFolderAsAlbumName   bool             // Use folder's name instead of metadata's title as Album name
	DryRun                 bool             // Display actions but don't change anything
	ForceSidecar           bool             // Generate a sidecar file for each file (default: TRUE)
	CreateStacks           bool             // Stack jpg/raw/burst (Default: TRUE)
	StackJpgRaws           bool             // Stack jpg/raw (Default: TRUE)
	StackBurst             bool             // Stack burst (Default: TRUE)
	DiscardArchived        bool             // Don't import archived assets (Default: FALSE)
	WhenNoDate             string           // When the date can't be determined use the FILE's date or NOW (default: FILE)

	BrowserConfig Configuration

	AssetIndex       *AssetIndex               // List of assets present on the server
	deleteServerList []*immich.Asset           // List of server assets to remove
	deleteLocalList  []*browser.LocalAssetFile // List of local assets to remove
	mediaUploaded    int                       // Count uploaded medias
	mediaCount       int                       // Count of media on the source
	updateAlbums     map[string]map[string]any // track immich albums changes
	stacks           *stacking.StackBuilder
	page             *tea.Program
	counters         *logger.Counters[logger.UpLdAction]
	lc               *logger.LogAndCount[logger.UpLdAction]
	ctx              context.Context
	browser          browser.Browser
	send             logger.Sender
}

func NewUpCmd(ctx context.Context, common *cmd.SharedFlags, args []string) (*UpCmd, error) {
	var err error
	cmd := flag.NewFlagSet("upload", flag.ExitOnError)

	app := UpCmd{
		SharedFlags:  common,
		updateAlbums: map[string]map[string]any{},
		ctx:          ctx,
	}

	app.SharedFlags.SetFlags(cmd)

	cmd.BoolFunc(
		"dry-run",
		"display actions but don't touch source or destination",
		myflag.BoolFlagFn(&app.DryRun, false))
	cmd.Var(&app.DateRange,
		"date",
		"Date of capture range.")
	cmd.StringVar(&app.ImportIntoAlbum,
		"album",
		"",
		"All assets will be added to this album.")
	cmd.BoolFunc(
		"force-sidecar",
		"Upload the photo and a sidecar file with known information like date and GPS coordinates. With google-photos, information comes from the metadata files. (DEFAULT false)",
		myflag.BoolFlagFn(&app.ForceSidecar, false))
	cmd.BoolFunc(
		"create-album-folder",
		" folder import only: Create albums for assets based on the parent folder",
		myflag.BoolFlagFn(&app.CreateAlbumAfterFolder, false))
	cmd.BoolFunc(
		"google-photos",
		"Import GooglePhotos takeout zip files",
		myflag.BoolFlagFn(&app.GooglePhotos, false))
	cmd.BoolFunc(
		"create-albums",
		" google-photos only: Create albums like there were in the source (default: TRUE)",
		myflag.BoolFlagFn(&app.CreateAlbums, true))
	cmd.StringVar(&app.PartnerAlbum,
		"partner-album",
		"",
		" google-photos only: Assets from partner will be added to this album. (ImportIntoAlbum, must already exist)")
	cmd.BoolFunc(
		"keep-partner",
		" google-photos only: Import also partner's items (default: TRUE)", myflag.BoolFlagFn(&app.KeepPartner, true))
	cmd.StringVar(&app.ImportFromAlbum,
		"from-album",
		"",
		" google-photos only: Import only from this album")

	cmd.BoolFunc(
		"keep-untitled-albums",
		" google-photos only: Keep Untitled albums and imports their contain (default: FALSE)", myflag.BoolFlagFn(&app.KeepUntitled, false))

	cmd.BoolFunc(
		"use-album-folder-as-name",
		" google-photos only: Use folder name and ignore albums' title (default:FALSE)", myflag.BoolFlagFn(&app.UseFolderAsAlbumName, false))

	cmd.BoolFunc(
		"discard-archived",
		" google-photos only: Do not import archived photos (default FALSE)", myflag.BoolFlagFn(&app.DiscardArchived, false))

	cmd.BoolFunc(
		"create-stacks",
		"Stack jpg/raw or bursts  (default TRUE)", myflag.BoolFlagFn(&app.CreateStacks, true))

	cmd.BoolFunc(
		"stack-jpg-raw",
		"Control the stacking of jpg/raw photos (default TRUE)", myflag.BoolFlagFn(&app.StackJpgRaws, true))
	cmd.BoolFunc(
		"stack-burst",
		"Control the stacking bursts (default TRUE)", myflag.BoolFlagFn(&app.StackBurst, true))

	// cmd.BoolVar(&app.Delete, "delete", false, "Delete local assets after upload")

	cmd.Var(&app.BrowserConfig.SelectExtensions, "select-types", "list of selected extensions separated by a comma")
	cmd.Var(&app.BrowserConfig.ExcludeExtensions, "exclude-types", "list of excluded extensions separated by a comma")

	cmd.StringVar(&app.WhenNoDate,
		"when-no-date",
		"FILE",
		" When the date of take can't be determined, use the FILE's date or the current time NOW. (default: FILE)")
	err = cmd.Parse(args)
	if err != nil {
		return nil, err
	}

	app.WhenNoDate = strings.ToUpper(app.WhenNoDate)
	switch app.WhenNoDate {
	case "FILE", "NOW":
	default:
		return nil, fmt.Errorf("the -when-no-date accepts FILE or NOW")
	}

	app.BrowserConfig.Validate()
	app.args = cmd.Args()

	return &app, err
}

func UploadCommand(ctx context.Context, common *cmd.SharedFlags, args []string) error {
	app, err := NewUpCmd(ctx, common, args)
	if err != nil {
		return err
	}
	defer func() {
		_ = fshelper.CloseFSs(app.fsyss)
	}()

	// Get the list of files / folders to scan
	fsyss, err := fshelper.ParsePath(app.args, app.GooglePhotos)
	if err != nil {
		return err
	}
	return app.run(ctx, fsyss)
}

func (app *UpCmd) run(ctx context.Context, fsyss []fs.FS) error {
	app.fsyss = fsyss
	// Get common flags whatever their position before or after the upload command
	err := app.SharedFlags.Start(ctx)
	if err != nil {
		return err
	}

	if app.CreateStacks || app.StackBurst || app.StackJpgRaws {
		app.stacks = stacking.NewStackBuilder(app.Immich.SupportedMedia())
	}

	app.counters = logger.NewCounters[logger.UpLdAction]()

	// Initialize the TUI model
	if !app.SharedFlags.NoUI {
		app.page = tea.NewProgram(NewUploadModel(app, app.counters), tea.WithAltScreen())
		app.send = app.page.Send
	} else {
		app.send = app.sendNoUI
	}

	app.lc = logger.NewLogAndCount[logger.UpLdAction](app.Log, app.send, app.counters)

	switch {
	case app.GooglePhotos:
		app.browser, err = app.ReadGoogleTakeOut(ctx, app.fsyss)
	default:
		app.browser, err = app.ExploreLocalFolder(ctx, app.fsyss)
	}
	if err != nil {
		return err
	}
	// Sequence of actions
	fullGrp := errgroup.Group{}
	fullGrp.Go(func() error {
		initGrp := errgroup.Group{}
		initGrp.Go(app.getAssets)
		initGrp.Go(app.prepare)
		err := initGrp.Wait()
		if err != nil {
			app.page.Send(msgQuit{err})
			return err
		}
		err = app.browse()
		app.send(msgQuit{err})
		return err
	})

	if !app.SharedFlags.NoUI {
		// Run the TUI
		m, err := app.page.Run()
		if err != nil {
			return nil
		}

		err = fullGrp.Wait()
		if err != nil {
			return err
		}
		app.page.Wait()
		if m, ok := m.(UploadModel); ok {
			report := m.countersMdl.View()
			defer func() {
				app.SharedFlags.Log.Print(m.countersMdl.View())
				fmt.Println(report)
			}()
			return m.err
		}
	} else {
		return fullGrp.Wait()
	}

	return nil
}

func (app *UpCmd) sendNoUI(msg tea.Msg) {
	if !app.SharedFlags.NoUI {
		app.page.Send(msg)
		return
	}

	switch msg := msg.(type) {
	case logger.MsgLog:
		if msg.Lvl != log.InfoLevel {
			fmt.Print(msg.Lvl.String(), " ")
		}
		fmt.Println(msg.Message)
	case logger.MsgStageSpinner:
		fmt.Println(msg.Label)
	}
}

func (app *UpCmd) getAssets() error {
	app.lc.Print("Get Server Statistics")
	statistics, err := app.Immich.GetServerStatistics(app.ctx)
	if err != nil {
		return err
	}

	app.lc.Printf("Receiving %d asset(s) from the server", statistics.Photos+statistics.Videos)
	totalOnImmich := float64(statistics.Photos + statistics.Videos)
	received := 0

	var list []*immich.Asset
	err = app.Immich.GetAllAssetsWithFilter(app.ctx, func(a *immich.Asset) {
		received++
		app.counters.Add(logger.UpldReceived)
		app.send(msgReceiveAsset(float64(received) / totalOnImmich))
		if a.IsTrashed {
			return
		}
		list = append(list, a)
	})
	if err != nil {
		return err
	}

	app.send(msgReceivingAssetDone{})
	app.AssetIndex = &AssetIndex{
		assets: list,
	}
	app.AssetIndex.ReIndex()
	return err
}

func (app *UpCmd) prepare() error {
	return app.browser.Prepare(app.ctx)
}

func (app *UpCmd) browse() error {
	var err error
	assetChan := app.browser.Browse(app.ctx)
assetLoop:
	for {
		select {
		case <-app.ctx.Done():
			return app.ctx.Err()

		case a, ok := <-assetChan:
			if !ok {
				break assetLoop
			}
			if a.Err != nil {
				app.lc.AddEntry(log.ErrorLevel, logger.UpldERROR, a.FileName, "error", a.Err)
			} else {
				err = app.handleAsset(app.ctx, a)
				if err != nil {
					app.lc.AddEntry(log.ErrorLevel, logger.UpldERROR, a.FileName, "error", a.Err)
				}
			}
		}
	}

	if app.CreateStacks {
		stacks := app.stacks.Stacks()
		if len(stacks) > 0 {
			app.send(logger.MsgStageSpinner{Label: "Creating stacks"})
		nextStack:
			for _, s := range stacks {
				switch {
				case !app.StackBurst && s.StackType == stacking.StackBurst:
					continue nextStack
				case !app.StackJpgRaws && s.StackType == stacking.StackRawJpg:
					continue nextStack
				}
				app.lc.AddEntry(log.InfoLevel, logger.UpldStack, s.Names[0], "files", s.Names[1:])
				if !app.DryRun {
					err = app.Immich.StackAssets(app.ctx, s.CoverID, s.IDs)
					if err != nil {
						app.lc.Error("Can't stack images", "error", err)
					}
				}
			}
		}
	}

	if app.CreateAlbums || app.CreateAlbumAfterFolder || (app.KeepPartner && app.PartnerAlbum != "") || app.ImportIntoAlbum != "" {
		err = app.ManageAlbums(app.ctx)
		if err != nil {
			app.lc.Error("Can't manage albums", "error", err)
			err = nil
		}
	}

	if len(app.deleteServerList) > 0 {
		ids := []string{}
		for _, da := range app.deleteServerList {
			ids = append(ids, da.ID)
		}
		err = app.DeleteServerAssets(app.ctx, ids)
		if err != nil {
			app.lc.Error("Can't removing duplicates", "error", err)
			err = nil
		}
	}

	if len(app.deleteLocalList) > 0 {
		err = app.DeleteLocalAssets()
	}
	return err
}

func (app *UpCmd) handleAsset(ctx context.Context, a *browser.LocalAssetFile) error {
	const willBeAddedToAlbum = "Will be added to the album: "
	defer func() {
		a.Close()
	}()
	app.mediaCount++

	// ext := path.Ext(a.FileName)
	// if _, err := fshelper.MimeFromExt(ext); err != nil {
	// 	app.journalAsset(a, logger.NOT_SELECTED, "not recognized extension")
	// 	return nil
	// }
	ext := path.Ext(a.FileName)
	if app.BrowserConfig.ExcludeExtensions.Exclude(ext) {
		app.lc.AddEntry(log.InfoLevel, logger.UpldNotSelected, a.FileName, "reason", "extension excluded")
		return nil
	}
	if !app.BrowserConfig.SelectExtensions.Include(ext) {
		app.lc.AddEntry(log.InfoLevel, logger.UpldNotSelected, a.FileName, "reason", "extension not selected")
		return nil
	}

	if !app.KeepPartner && a.FromPartner {
		app.lc.AddEntry(log.InfoLevel, logger.UpldNotSelected, a.FileName, "reason", "partner's assets are excluded")
		return nil
	}

	if !app.KeepTrashed && a.Trashed {
		app.lc.AddEntry(log.InfoLevel, logger.UpldNotSelected, a.FileName, "reason", "trashed assets are excluded")
		return nil
	}

	if app.ImportFromAlbum != "" && !app.isInAlbum(a, app.ImportFromAlbum) {
		app.lc.AddEntry(log.InfoLevel, logger.UpldNotSelected, a.FileName, "reason", "asset not in selected album")
		return nil
	}

	if app.DiscardArchived && a.Archived {
		app.lc.AddEntry(log.InfoLevel, logger.UpldNotSelected, a.FileName, "reason", "archived assets are excluded")
		return nil
	}

	if app.DateRange.IsSet() {
		d := a.DateTaken
		if d.IsZero() {
			app.lc.AddEntry(log.InfoLevel, logger.UpldNotSelected, a.FileName, "reason", "date of capture is unknown and a date range is given")
			return nil
		}
		if !app.DateRange.InRange(d) {
			app.lc.AddEntry(log.InfoLevel, logger.UpldNotSelected, a.FileName, "reason", "date of capture is out of the date range")
			return nil
		}
	}

	if !app.KeepUntitled {
		a.Albums = gen.Filter(a.Albums, func(i browser.LocalAlbum) bool {
			return i.Name != ""
		})
	}

	advice, err := app.AssetIndex.ShouldUpload(a)
	if err != nil {
		return err
	}

	var ID string
	switch advice.Advice {
	case NotOnServer:
		ID, err = app.UploadAsset(ctx, a)
		if app.Delete && err == nil {
			app.deleteLocalList = append(app.deleteLocalList, a)
		}
	case SmallerOnServer:
		app.lc.AddEntry(log.InfoLevel, logger.UpldUpgraded, a.FileName, "reason", advice.Message)
		// add the superior asset into albums of the original asset
		for _, al := range advice.ServerAsset.Albums {
			app.lc.AddEntry(log.InfoLevel, logger.UpldINFO, a.FileName, "reason", willBeAddedToAlbum+al.AlbumName)
			a.AddAlbum(browser.LocalAlbum{Name: al.AlbumName})
		}
		ID, err = app.UploadAsset(ctx, a)
		if err != nil {
			app.deleteServerList = append(app.deleteServerList, advice.ServerAsset)
			if app.Delete {
				app.deleteLocalList = append(app.deleteLocalList, a)
			}
		}
	case SameOnServer:
		if !advice.ServerAsset.JustUploaded {
			app.lc.AddEntry(log.InfoLevel, logger.UpldServerDuplicate, a.FileName, "reason", advice.Message)
		} else {
			app.lc.AddEntry(log.InfoLevel, logger.UpldLocalDuplicate, a.FileName, "reason", "File already handled")
		}
		ID = advice.ServerAsset.ID
		if app.CreateAlbums {
			for _, al := range a.Albums {
				app.lc.AddEntry(log.InfoLevel, logger.UpldINFO, a.FileName, "reason", willBeAddedToAlbum+al.Name)
				app.AddToAlbum(advice.ServerAsset.ID, app.albumName(al))
			}
		}
		if app.ImportIntoAlbum != "" {
			app.lc.AddEntry(log.InfoLevel, logger.UpldINFO, a.FileName, "reason", willBeAddedToAlbum+app.ImportIntoAlbum)
			app.AddToAlbum(advice.ServerAsset.ID, app.ImportIntoAlbum)
		}
		if app.PartnerAlbum != "" && a.FromPartner {
			app.lc.AddEntry(log.InfoLevel, logger.UpldINFO, a.FileName, "reason", willBeAddedToAlbum+app.PartnerAlbum)
			app.AddToAlbum(advice.ServerAsset.ID, app.PartnerAlbum)
		}
		if !advice.ServerAsset.JustUploaded {
			if app.Delete {
				app.deleteLocalList = append(app.deleteLocalList, a)
			}
		} else {
			return nil
		}
	case BetterOnServer:
		app.lc.AddEntry(log.InfoLevel, logger.UpldServerBetter, a.FileName, "reason", advice.Message)
		ID = advice.ServerAsset.ID
		// keep the server version but update albums
		if app.CreateAlbums {
			for _, al := range a.Albums {
				app.lc.AddEntry(log.InfoLevel, logger.UpldINFO, a.FileName, "reason", willBeAddedToAlbum+al.Name)
				app.AddToAlbum(advice.ServerAsset.ID, app.albumName(al))
			}
		}
		if app.PartnerAlbum != "" && a.FromPartner {
			app.lc.AddEntry(log.InfoLevel, logger.UpldINFO, a.FileName, "reason", willBeAddedToAlbum+app.PartnerAlbum)
			app.AddToAlbum(advice.ServerAsset.ID, app.PartnerAlbum)
		}
	}

	if err != nil {
		return nil
	}

	if app.ImportIntoAlbum != "" ||
		(app.GooglePhotos && (app.CreateAlbums || app.PartnerAlbum != "")) ||
		(!app.GooglePhotos && app.CreateAlbumAfterFolder) {
		albums := []browser.LocalAlbum{}

		if app.ImportIntoAlbum != "" {
			albums = append(albums, browser.LocalAlbum{Path: app.ImportIntoAlbum, Name: app.ImportIntoAlbum})
		} else {
			switch {
			case app.GooglePhotos:
				albums = append(albums, a.Albums...)
				if app.PartnerAlbum != "" && a.FromPartner {
					albums = append(albums, browser.LocalAlbum{Path: app.PartnerAlbum, Name: app.PartnerAlbum})
				}
			case !app.GooglePhotos && app.CreateAlbumAfterFolder:
				album := path.Base(path.Dir(a.FileName))
				if album != "" && album != "." {
					albums = append(albums, browser.LocalAlbum{Path: album, Name: album})
				}
			}
		}

		if len(albums) > 0 {
			Names := []string{}
			for _, al := range albums {
				Name := app.albumName(al)
				if app.GooglePhotos && Name == "" {
					continue
				}
				Names = append(Names, Name)
			}
			if len(Names) > 0 {
				app.lc.AddEntry(log.InfoLevel, logger.UpldAlbum, a.FileName, "files", Names)
				for _, n := range Names {
					app.AddToAlbum(ID, n)
				}
			}
		}
	}

	shouldUpdate := a.Description != ""
	shouldUpdate = shouldUpdate || a.Favorite
	shouldUpdate = shouldUpdate || a.Longitude != 0 || a.Latitude != 0
	shouldUpdate = shouldUpdate || a.Archived

	if !app.DryRun && shouldUpdate {
		_, err := app.Immich.UpdateAsset(ctx, ID, a)
		if err != nil {
			app.lc.AddEntry(log.ErrorLevel, logger.UpldServerError, "error", fmt.Errorf("can't update the asset '%w': ", err))
		}
	}
	return nil
}

func (app *UpCmd) isInAlbum(a *browser.LocalAssetFile, album string) bool {
	for _, al := range a.Albums {
		if app.albumName(al) == album {
			return true
		}
	}
	return false
}

func (app *UpCmd) ReadGoogleTakeOut(ctx context.Context, fsyss []fs.FS) (browser.Browser, error) {
	app.Delete = false
	return gp.NewTakeout(ctx, app.lc, app.Immich.SupportedMedia(), fsyss...)
}

func (app *UpCmd) ExploreLocalFolder(ctx context.Context, fsyss []fs.FS) (browser.Browser, error) {
	b, err := files.NewLocalFiles(ctx, app.lc, fsyss...)
	if err != nil {
		return nil, err
	}
	b.SetSupportedMedia(app.Immich.SupportedMedia())
	b.SetWhenNoDate(app.WhenNoDate)
	return b, nil
}

// UploadAsset upload the asset on the server
// Add the assets into listed albums
// return ID of the asset

func (app *UpCmd) UploadAsset(ctx context.Context, a *browser.LocalAssetFile) (string, error) {
	var resp immich.AssetResponse
	var err error
	if !app.DryRun {
		if app.ForceSidecar {
			sc := metadata.SideCar{}
			sc.DateTaken = a.DateTaken
			sc.Latitude = a.Latitude
			sc.Longitude = a.Longitude
			sc.Elevation = a.Altitude
			sc.FileName = a.FileName + ".xmp"
			a.SideCar = &sc
		}

		resp, err = app.Immich.AssetUpload(ctx, a)
	} else {
		resp.ID = uuid.NewString()
	}
	if err != nil {
		app.lc.AddEntry(log.ErrorLevel, logger.UpldServerError, a.FileName, "error", err)
		return "", err
	}
	if !resp.Duplicate {
		app.lc.AddEntry(log.InfoLevel, logger.UpldUploaded, a.FileName, "name", a.Title)
		app.AssetIndex.AddLocalAsset(a, resp.ID)
		app.mediaUploaded += 1
		if app.CreateStacks {
			app.stacks.ProcessAsset(resp.ID, a.FileName, a.DateTaken)
		}
	} else {
		app.lc.AddEntry(log.InfoLevel, logger.UpldServerDuplicate, a.FileName, "reason", "already on the server")
	}

	return resp.ID, nil
}

func (app *UpCmd) albumName(al browser.LocalAlbum) string {
	Name := al.Name
	if app.GooglePhotos {
		switch {
		case app.UseFolderAsAlbumName:
			Name = path.Base(al.Path)
		case app.KeepUntitled && Name == "":
			Name = path.Base(al.Path)
		}
	}
	return Name
}

func (app *UpCmd) AddToAlbum(id string, album string) {
	l := app.updateAlbums[album]
	if l == nil {
		l = map[string]any{}
	}
	l[id] = nil
	app.updateAlbums[album] = l
}

func (app *UpCmd) DeleteLocalAssets() error {
	app.page.Printf("%d local assets to delete.", len(app.deleteLocalList))

	for _, a := range app.deleteLocalList {
		if !app.DryRun {
			app.page.Printf("delete file %q", a.Title)
			err := a.Remove()
			if err != nil {
				return err
			}
		} else {
			app.page.Printf("file %q not deleted, dry run mode", a.Title)
		}
	}
	return nil
}

func (app *UpCmd) DeleteServerAssets(ctx context.Context, ids []string) error {
	app.lc.AddEntry(log.InfoLevel, logger.UpldDeleteServerAssets, "", "ids", ids)
	if !app.DryRun {
		err := app.Immich.DeleteAssets(ctx, ids, false)
		return err
	}
	return nil
}

func (app *UpCmd) ManageAlbums(ctx context.Context) error {
	app.send(logger.MsgStageSpinner{Label: "Managing albums"})
	if len(app.updateAlbums) > 0 {
		serverAlbums, err := app.Immich.GetAllAlbums(ctx)
		if err != nil {
			return fmt.Errorf("can't get the album list from the server: %w", err)
		}
		for album, list := range app.updateAlbums {
			found := false
			for _, sal := range serverAlbums {
				if sal.AlbumName == album {
					found = true
					app.lc.AddEntry(log.InfoLevel, logger.UpldCreateAlbum, album, "ids", gen.MapKeys(list))
					if !app.DryRun {
						rr, err := app.Immich.AddAssetToAlbum(ctx, sal.ID, gen.MapKeys(list))
						if err != nil {
							app.lc.AddEntry(log.ErrorLevel, logger.UpldCreateAlbum, album, "error", err, "ids", gen.MapKeys(list))
						} else {
							added := 0
							for _, r := range rr {
								if r.Success {
									added++
								}
								if !r.Success && r.Error != "duplicate" {
									app.lc.AddEntry(log.ErrorLevel, logger.UpldCreateAlbum, album, "error", err)
								}
							}
						}
					}
				}
			}
			if found {
				continue
			}
			if list != nil {
				app.send(logger.MsgLog{Lvl: log.InfoLevel, Message: fmt.Sprintf("Create the album %s", album)})
				if !app.DryRun {
					_, err := app.Immich.CreateAlbum(ctx, album, gen.MapKeys(list))
					if err != nil {
						app.lc.AddEntry(log.ErrorLevel, logger.UpldCreateAlbum, album, "error", err)
					}
				}
			}
		}
	}
	return nil
}

// - - go:generate stringer -type=AdviceCode
type AdviceCode int

func (a AdviceCode) String() string {
	switch a {
	case IDontKnow:
		return "IDontKnow"
	// case SameNameOnServerButNotSure:
	// 	return "SameNameOnServerButNotSure"
	case SmallerOnServer:
		return "SmallerOnServer"
	case BetterOnServer:
		return "BetterOnServer"
	case SameOnServer:
		return "SameOnServer"
	case NotOnServer:
		return "NotOnServer"
	}
	return fmt.Sprintf("advice(%d)", a)
}

const (
	IDontKnow AdviceCode = iota
	SmallerOnServer
	BetterOnServer
	SameOnServer
	NotOnServer
)

type Advice struct {
	Advice      AdviceCode
	Message     string
	ServerAsset *immich.Asset
	LocalAsset  *browser.LocalAssetFile
}

func formatBytes(s int) string {
	suffixes := []string{"B", "KB", "MB", "GB"}
	bytes := float64(s)
	base := 1024.0
	if bytes < base {
		return fmt.Sprintf("%.0f %s", bytes, suffixes[0])
	}
	exp := int64(0)
	for bytes >= base && exp < int64(len(suffixes)-1) {
		bytes /= base
		exp++
	}
	roundedSize := math.Round(bytes*10) / 10
	return fmt.Sprintf("%.1f %s", roundedSize, suffixes[exp])
}

func (ai *AssetIndex) adviceSameOnServer(sa *immich.Asset) *Advice {
	return &Advice{
		Advice:      SameOnServer,
		Message:     fmt.Sprintf("An asset with the same name:%q, date:%q and size:%s exists on the server. No need to upload.", sa.OriginalFileName, sa.ExifInfo.DateTimeOriginal.Format(time.DateTime), formatBytes(sa.ExifInfo.FileSizeInByte)),
		ServerAsset: sa,
	}
}

func (ai *AssetIndex) adviceSmallerOnServer(sa *immich.Asset) *Advice {
	return &Advice{
		Advice:      SmallerOnServer,
		Message:     fmt.Sprintf("An asset with the same name:%q and date:%q but with smaller size:%s exists on the server. Replace it.", sa.OriginalFileName, sa.ExifInfo.DateTimeOriginal.Format(time.DateTime), formatBytes(sa.ExifInfo.FileSizeInByte)),
		ServerAsset: sa,
	}
}

func (ai *AssetIndex) adviceBetterOnServer(sa *immich.Asset) *Advice {
	return &Advice{
		Advice:      BetterOnServer,
		Message:     fmt.Sprintf("An asset with the same name:%q and date:%q but with bigger size:%s exists on the server. No need to upload.", sa.OriginalFileName, sa.ExifInfo.DateTimeOriginal.Format(time.DateTime), formatBytes(sa.ExifInfo.FileSizeInByte)),
		ServerAsset: sa,
	}
}

func (ai *AssetIndex) adviceNotOnServer() *Advice {
	return &Advice{
		Advice:  NotOnServer,
		Message: "This a new asset, upload it.",
	}
}

// ShouldUpload check if the server has this asset
//
// The server may have different assets with the same name. This happens with photos produced by digital cameras.
// The server may have the asset, but in lower resolution. Compare the taken date and resolution

func (ai *AssetIndex) ShouldUpload(la *browser.LocalAssetFile) (*Advice, error) {
	filename := la.Title
	if path.Ext(filename) == "" {
		filename += path.Ext(la.FileName)
	}

	ID := la.DeviceAssetID()

	sa := ai.byID[ID]
	if sa != nil {
		// the same ID exist on the server
		return ai.adviceSameOnServer(sa), nil
	}

	var l []*immich.Asset

	// check all files with the same name

	n := filepath.Base(filename)
	l = ai.byName[n]
	if len(l) == 0 {
		// n = strings.TrimSuffix(n, filepath.Ext(n))
		l = ai.byName[n]
	}

	if len(l) > 0 {
		dateTaken := la.DateTaken
		size := int(la.Size())

		for _, sa = range l {
			compareDate := compareDate(dateTaken, sa.ExifInfo.DateTimeOriginal.Time)
			compareSize := size - sa.ExifInfo.FileSizeInByte

			switch {
			case compareDate == 0 && compareSize == 0:
				return ai.adviceSameOnServer(sa), nil
			case compareDate == 0 && compareSize > 0:
				return ai.adviceSmallerOnServer(sa), nil
			case compareDate == 0 && compareSize < 0:
				return ai.adviceBetterOnServer(sa), nil
			}
		}
	}
	return ai.adviceNotOnServer(), nil
}

func compareDate(d1 time.Time, d2 time.Time) int {
	diff := d1.Sub(d2)

	switch {
	case diff < -5*time.Minute:
		return -1
	case diff >= 5*time.Minute:
		return +1
	}
	return 0
}
