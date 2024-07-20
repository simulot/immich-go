// Command Upload

package upload

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/google/uuid"
	"github.com/simulot/immich-go/browser"
	"github.com/simulot/immich-go/browser/files"
	"github.com/simulot/immich-go/browser/gp"
	"github.com/simulot/immich-go/cmd"
	"github.com/simulot/immich-go/helpers/fileevent"
	"github.com/simulot/immich-go/helpers/fshelper"
	"github.com/simulot/immich-go/helpers/gen"
	"github.com/simulot/immich-go/helpers/myflag"
	"github.com/simulot/immich-go/helpers/namematcher"
	"github.com/simulot/immich-go/helpers/stacking"
	"github.com/simulot/immich-go/immich"
)

type UpCmd struct {
	*cmd.SharedFlags // shared flags and immich client

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
	CreateStacks           bool             // Stack jpg/raw/burst (Default: TRUE)
	StackJpgRaws           bool             // Stack jpg/raw (Default: TRUE)
	StackBurst             bool             // Stack burst (Default: TRUE)
	DiscardArchived        bool             // Don't import archived assets (Default: FALSE)
	WhenNoDate             string           // When the date can't be determined use the FILE's date or NOW (default: FILE)
	BannedFiles            namematcher.List // List of banned file name patterns

	BrowserConfig Configuration

	albums map[string]immich.AlbumSimplified // Albums by title

	AssetIndex       *AssetIndex               // List of assets present on the server
	deleteServerList []*immich.Asset           // List of server assets to remove
	deleteLocalList  []*browser.LocalAssetFile // List of local assets to remove
	// updateAlbums     map[string]map[string]any // track immich albums changes
	stacks  *stacking.StackBuilder
	browser browser.Browser
}

func UploadCommand(ctx context.Context, common *cmd.SharedFlags, args []string) error {
	app, err := newCommand(ctx, common, args, func() ([]fs.FS, error) {
		return fshelper.ParsePath(args)
	})
	if err != nil {
		return err
	}
	if len(app.fsyss) == 0 {
		return nil
	}
	return app.run(ctx)
}

type fsOpener func() ([]fs.FS, error)

func newCommand(ctx context.Context, common *cmd.SharedFlags, args []string, fsOpener fsOpener) (*UpCmd, error) {
	var err error
	cmd := flag.NewFlagSet("upload", flag.ExitOnError)

	app := UpCmd{
		SharedFlags: common,
	}
	app.BannedFiles, err = namematcher.New(
		`@eaDir/`,
		`@__thumb/`,          // QNAP
		`SYNOFILE_THUMB_*.*`, // SYNOLOGY
		`Lightroom Catalog/`, // LR
		`thumbnails/`,        // Android photo
		`.DS_Store/`,         // Mac OS custom attributes
	)
	if err != nil {
		return nil, err
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

	cmd.Var(&app.BannedFiles, "exclude-files", "Ignore files based on a pattern. Case insensitive. Add one option for each pattern do you need.")

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
	err = app.SharedFlags.Start(ctx)
	if err != nil {
		return nil, err
	}

	app.fsyss, err = fsOpener()
	if err != nil {
		return nil, err
	}
	if len(app.fsyss) == 0 {
		fmt.Println("No file found matching the pattern: ", strings.Join(cmd.Args(), ","))
		app.Log.Info("No file found matching the pattern: " + strings.Join(cmd.Args(), ","))
	}
	return &app, nil
}

func (app *UpCmd) run(ctx context.Context) error {
	defer func() {
		_ = fshelper.CloseFSs(app.fsyss)
	}()

	if app.CreateStacks || app.StackBurst || app.StackJpgRaws {
		app.stacks = stacking.NewStackBuilder(app.Immich.SupportedMedia())
	}

	var err error
	switch {
	case app.GooglePhotos:
		app.Log.Info("Browsing google take out archive...")
		app.browser, err = app.ReadGoogleTakeOut(ctx, app.fsyss)
	default:
		app.Log.Info("Browsing folder(s)...")
		app.browser, err = app.ExploreLocalFolder(ctx, app.fsyss)
	}

	if err != nil {
		return err
	}

	defer func() {
		if app.DebugCounters {
			fn := strings.TrimSuffix(app.LogFile, filepath.Ext(app.LogFile)) + ".csv"
			f, err := os.Create(fn)
			if err == nil {
				_ = app.Jnl.WriteFileCounts(f)
				fmt.Println("\nCheck the counters file: ", f.Name())
				f.Close()
			}
		}
	}()

	if app.NoUI {
		return app.runNoUI(ctx)
	}

	_, err = tcell.NewScreen()
	if err != nil {
		app.Log.Error("can't initialize the screen for the UI mode. Falling back to no-gui mode")
		fmt.Println("can't initialize the screen for the UI mode. Falling back to no-gui mode")
		return app.runNoUI(ctx)
	}
	return app.runUI(ctx)
}

func (app *UpCmd) getImmichAlbums(ctx context.Context) error {
	serverAlbums, err := app.Immich.GetAllAlbums(ctx)
	app.albums = map[string]immich.AlbumSimplified{}
	if err != nil {
		return fmt.Errorf("can't get the album list from the server: %w", err)
	}
	for _, a := range serverAlbums {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			app.albums[a.AlbumName] = a
		}
	}
	return nil
}

func (app *UpCmd) getImmichAssets(ctx context.Context, updateFn progressUpdate) error {
	statistics, err := app.Immich.GetAssetStatistics(ctx)
	if err != nil {
		return err
	}
	totalOnImmich := statistics.Total
	received := 0

	var list []*immich.Asset

	err = app.Immich.GetAllAssetsWithFilter(ctx, func(a *immich.Asset) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			received++
			list = append(list, a)
			if updateFn != nil {
				updateFn(received, totalOnImmich)
			}
			return nil
		}
	})
	if err != nil {
		return err
	}
	if updateFn != nil {
		updateFn(totalOnImmich, totalOnImmich)
	}
	app.AssetIndex = &AssetIndex{
		assets: list,
	}
	app.AssetIndex.ReIndex()
	return nil
}

func (app *UpCmd) uploadLoop(ctx context.Context) error {
	var err error
	assetChan := app.browser.Browse(ctx)
assetLoop:
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case a, ok := <-assetChan:
			if !ok {
				break assetLoop
			}
			if a.Err != nil {
				app.Jnl.Record(ctx, fileevent.Error, a, a.FileName, a.Err.Error())
			} else {
				err = app.handleAsset(ctx, a)
				if err != nil {
					app.Jnl.Record(ctx, fileevent.Error, a, a.FileName, a.Err.Error())
				}
			}
		}
	}

	if app.CreateStacks {
		stacks := app.stacks.Stacks()
		if len(stacks) > 0 {
			app.Log.Info("Creating stacks")
		nextStack:
			for _, s := range stacks {
				switch {
				case !app.StackBurst && s.StackType == stacking.StackBurst:
					continue nextStack
				case !app.StackJpgRaws && s.StackType == stacking.StackRawJpg:
					continue nextStack
				}
				app.Log.Info(fmt.Sprintf("Stacking %s...", strings.Join(s.Names, ", ")))
				if !app.DryRun {
					err = app.Immich.StackAssets(ctx, s.CoverID, s.IDs)
					if err != nil {
						app.Log.Error(fmt.Sprintf("Can't stack images: %s", err))
					}
				}
			}
		}
	}

	// if app.CreateAlbums || app.CreateAlbumAfterFolder || (app.KeepPartner && app.PartnerAlbum != "") || app.ImportIntoAlbum != "" {
	// 	app.Log.Info("Managing albums")
	// 	err = app.ManageAlbums(ctx)
	// 	if err != nil {
	// 		app.Log.Error(err.Error())
	// 		err = nil
	// 	}
	// }

	if len(app.deleteServerList) > 0 {
		ids := []string{}
		for _, da := range app.deleteServerList {
			ids = append(ids, da.ID)
		}
		err := app.DeleteServerAssets(ctx, ids)
		if err != nil {
			return fmt.Errorf("can't delete server's assets: %w", err)
		}
	}

	if len(app.deleteLocalList) > 0 {
		err = app.DeleteLocalAssets()
	}

	return err
}

func (app *UpCmd) handleAsset(ctx context.Context, a *browser.LocalAssetFile) error {
	defer func() {
		a.Close()
	}()
	ext := path.Ext(a.FileName)
	if app.BrowserConfig.ExcludeExtensions.Exclude(ext) {
		app.Jnl.Record(ctx, fileevent.UploadNotSelected, a, a.FileName, "reason", "extension in rejection list")
		return nil
	}
	if !app.BrowserConfig.SelectExtensions.Include(ext) {
		app.Jnl.Record(ctx, fileevent.UploadNotSelected, a.FileName, "reason", "extension not in selection list")
		return nil
	}

	if !app.KeepPartner && a.FromPartner {
		app.Jnl.Record(ctx, fileevent.UploadNotSelected, a, a.FileName, "reason", "partners asset excluded")
		return nil
	}

	if !app.KeepTrashed && a.Trashed {
		app.Jnl.Record(ctx, fileevent.UploadNotSelected, a, a.FileName, "reason", "trashed asset excluded")
		return nil
	}

	if app.ImportFromAlbum != "" && !app.isInAlbum(a, app.ImportFromAlbum) {
		app.Jnl.Record(ctx, fileevent.UploadNotSelected, a.FileName, "reason", "doesn't belong to required album")
		return nil
	}

	if app.DiscardArchived && a.Archived {
		app.Jnl.Record(ctx, fileevent.UploadNotSelected, a, a.FileName, "reason", "archived asset are discarded")
		return nil
	}

	if app.DateRange.IsSet() {
		d := a.Metadata.DateTaken
		if d.IsZero() {
			app.Jnl.Record(ctx, fileevent.UploadNotSelected, a, a.FileName, "reason", "date of capture is unknown")
			return nil
		}
		if !app.DateRange.InRange(d) {
			app.Jnl.Record(ctx, fileevent.UploadNotSelected, a, a.FileName, "reason", "date of capture is out of the given range")
			return nil
		}
	}

	if !app.KeepUntitled {
		a.Albums = gen.Filter(a.Albums, func(i browser.LocalAlbum) bool {
			return i.Title != ""
		})
	}

	advice, err := app.AssetIndex.ShouldUpload(a)
	if err != nil {
		return err
	}

	switch advice.Advice {
	case NotOnServer: // Upload and manage albums
		ID, err := app.UploadAsset(ctx, a)
		if err != nil {
			return nil
		}
		app.manageAssetAlbum(ctx, ID, a, advice)

	case SmallerOnServer: // Upload, manage albums and delete the server's asset
		app.Jnl.Record(ctx, fileevent.UploadUpgraded, a, a.FileName)
		// add the superior asset into albums of the original asset
		ID, err := app.UploadAsset(ctx, a)
		if err != nil {
			return nil
		}
		app.manageAssetAlbum(ctx, ID, a, advice)
		// delete the existing lower quality asset
		err = app.deleteAsset(ctx, advice.ServerAsset.ID)
		if err != nil {
			app.Jnl.Record(ctx, fileevent.Error, a, a.FileName, "error", err.Error())
		}

	case SameOnServer: // manage albums
		// Set add the server asset into albums determined locally
		if !advice.ServerAsset.JustUploaded {
			app.Jnl.Record(ctx, fileevent.UploadServerDuplicate, a, a.FileName)
		} else {
			app.Jnl.Record(ctx, fileevent.AnalysisLocalDuplicate, a, a.FileName)
		}
		app.manageAssetAlbum(ctx, advice.ServerAsset.ID, a, advice)

	case BetterOnServer: // and manage albums
		app.Jnl.Record(ctx, fileevent.UploadServerBetter, a, a.FileName)
		app.manageAssetAlbum(ctx, advice.ServerAsset.ID, a, advice)
	}

	return nil
}

func (app *UpCmd) deleteAsset(ctx context.Context, id string) error {
	return app.Immich.DeleteAssets(ctx, []string{id}, true)
}

// manageAssetAlbum keep the albums updated
// errors are logged, but not returned
func (app *UpCmd) manageAssetAlbum(ctx context.Context, assetID string, a *browser.LocalAssetFile, advice *Advice) {
	addedTo := map[string]any{}
	if advice.ServerAsset != nil {
		for _, al := range advice.ServerAsset.Albums {
			app.Jnl.Record(ctx, fileevent.UploadAddToAlbum, a, a.FileName, "album", al.AlbumName, "reason", "lower quality asset's album")
			if !app.DryRun {
				err := app.AddToAlbum(ctx, assetID, browser.LocalAlbum{Title: al.AlbumName, Description: al.Description})
				if err != nil {
					app.Jnl.Record(ctx, fileevent.Error, a, a.FileName, "error", err.Error())
				}
			}
			addedTo[al.AlbumName] = nil
		}
	}

	if app.CreateAlbums {
		for _, al := range a.Albums {
			album := al.Title
			if app.GooglePhotos && (app.UseFolderAsAlbumName || album == "") {
				album = filepath.Base(al.Path)
			}
			if _, exist := addedTo[album]; !exist {
				app.Jnl.Record(ctx, fileevent.UploadAddToAlbum, a, a.FileName, "album", album)
				if !app.DryRun {
					err := app.AddToAlbum(ctx, assetID, al)
					if err != nil {
						app.Jnl.Record(ctx, fileevent.Error, a, a.FileName, "error", err.Error())
					}
				}
			}
		}
	}
	if app.ImportIntoAlbum != "" {
		app.Jnl.Record(ctx, fileevent.UploadAddToAlbum, a, a.FileName, "album", app.ImportIntoAlbum, "reason", "option -album")
		if !app.DryRun {
			err := app.AddToAlbum(ctx, assetID, browser.LocalAlbum{Title: app.ImportIntoAlbum})
			if err != nil {
				app.Jnl.Record(ctx, fileevent.Error, a, a.FileName, "error", err.Error())
			}
		}
	}

	if app.GooglePhotos {
		if app.PartnerAlbum != "" && a.FromPartner {
			app.Jnl.Record(ctx, fileevent.UploadAddToAlbum, a, a.FileName, "album", app.PartnerAlbum, "reason", "option -partner-album")
			if !app.DryRun {
				err := app.AddToAlbum(ctx, assetID, browser.LocalAlbum{Title: app.PartnerAlbum})
				if err != nil {
					app.Jnl.Record(ctx, fileevent.Error, a, a.FileName, "error", err.Error())
				}
			}
		}
	} else {
		if app.CreateAlbumAfterFolder {
			album := path.Base(path.Dir(a.FileName))
			if album == "" || album == "." {
				if fsys, ok := a.FSys.(fshelper.NameFS); ok {
					album = fsys.Name()
				} else {
					album = "no-folder-name"
				}
			}
			app.Jnl.Record(ctx, fileevent.UploadAddToAlbum, a, a.FileName, "album", album, "reason", "option -create-album-folder")
			if !app.DryRun {
				err := app.AddToAlbum(ctx, assetID, browser.LocalAlbum{Title: album})
				if err != nil {
					app.Jnl.Record(ctx, fileevent.Error, a, a.FileName, "error", err.Error())
				}
			}
		}
	}
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
	b, err := gp.NewTakeout(ctx, app.Jnl, app.Immich.SupportedMedia(), fsyss...)
	if err != nil {
		return nil, err
	}
	b.SetBannedFiles(app.BannedFiles)
	return b, err
}

func (app *UpCmd) ExploreLocalFolder(ctx context.Context, fsyss []fs.FS) (browser.Browser, error) {
	b, err := files.NewLocalFiles(ctx, app.Jnl, fsyss...)
	if err != nil {
		return nil, err
	}
	b.SetSupportedMedia(app.Immich.SupportedMedia())
	b.SetWhenNoDate(app.WhenNoDate)
	b.SetBannedFiles(app.BannedFiles)
	return b, nil
}

// UploadAsset upload the asset on the server
// Add the assets into listed albums
// return ID of the asset

func (app *UpCmd) UploadAsset(ctx context.Context, a *browser.LocalAssetFile) (string, error) {
	var resp, liveResp immich.AssetResponse
	var err error
	if !app.DryRun {
		if a.LivePhoto != nil {
			liveResp, err = app.Immich.AssetUpload(ctx, a.LivePhoto)
			if err == nil {
				if liveResp.Status == immich.UploadDuplicate {
					app.Jnl.Record(ctx, fileevent.UploadServerDuplicate, a.LivePhoto, a.LivePhoto.FileName, "info", "the server has this file")
				} else {
					a.LivePhotoID = liveResp.ID
					app.Jnl.Record(ctx, fileevent.Uploaded, a.LivePhoto, a.LivePhoto.FileName)
				}
			} else {
				app.Jnl.Record(ctx, fileevent.UploadServerError, a.LivePhoto, a.LivePhoto.FileName, "error", err.Error())
			}
		}
		resp, err = app.Immich.AssetUpload(ctx, a)
		if err == nil {
			if resp.Status == immich.UploadDuplicate {
				app.Jnl.Record(ctx, fileevent.UploadServerDuplicate, a, a.FileName, "info", "the server has this file")
			} else {
				app.Jnl.Record(ctx, fileevent.Uploaded, a, a.FileName, "capture date", a.Metadata.DateTaken.String())
			}
		} else {
			app.Jnl.Record(ctx, fileevent.UploadServerError, a, a.FileName, "error", err.Error())
			return "", err
		}
	} else {
		// dry-run mode
		if a.LivePhoto != nil {
			liveResp.ID = uuid.NewString()
			app.Jnl.Record(ctx, fileevent.Uploaded, a.LivePhoto, a.LivePhoto.FileName)
		}
		resp.ID = uuid.NewString()
		app.Jnl.Record(ctx, fileevent.Uploaded, a, a.FileName, "capture date", a.Metadata.DateTaken.String())
	}
	if resp.Status != immich.UploadDuplicate {
		if a.LivePhoto != nil {
			app.AssetIndex.AddLocalAsset(a, liveResp.ID)
		}
		app.AssetIndex.AddLocalAsset(a, resp.ID)
		if app.CreateStacks {
			app.stacks.ProcessAsset(resp.ID, a.FileName, a.Metadata.DateTaken)
		}
	}

	return resp.ID, nil
}

func (app *UpCmd) albumName(al browser.LocalAlbum) string {
	Name := al.Title
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

// AddToAlbum add the ID to the immich album having the same name as the local album
func (app *UpCmd) AddToAlbum(ctx context.Context, id string, album browser.LocalAlbum) error {
	title := album.Title
	if (app.GooglePhotos && (title == "" || app.CreateAlbumAfterFolder)) || app.UseFolderAsAlbumName {
		title = filepath.Base(album.Path)
	}

	l, exist := app.albums[title]
	if !exist {
		a, err := app.Immich.CreateAlbum(ctx, title, album.Description, []string{id})
		if err != nil {
			return err
		}
		app.albums[title] = immich.AlbumSimplified{ID: a.ID, AlbumName: a.AlbumName, Description: a.Description}
	} else {
		_, err := app.Immich.AddAssetToAlbum(ctx, l.ID, []string{id})
		if err != nil {
			return err
		}
	}
	return nil
}

func (app *UpCmd) DeleteLocalAssets() error {
	app.Log.Info(fmt.Sprintf("%d local assets to delete.", len(app.deleteLocalList)))

	for _, a := range app.deleteLocalList {
		if !app.DryRun {
			app.Log.Info(fmt.Sprintf("delete file %q", a.Title))
			err := a.Remove()
			if err != nil {
				return err
			}
		} else {
			app.Log.Info(fmt.Sprintf("file %q not deleted, dry run mode.", a.Title))
		}
	}
	return nil
}

func (app *UpCmd) DeleteServerAssets(ctx context.Context, ids []string) error {
	app.Log.Info(fmt.Sprintf("%d server assets to delete.", len(ids)))

	if !app.DryRun {
		err := app.Immich.DeleteAssets(ctx, ids, false)
		return err
	}
	app.Log.Info(fmt.Sprintf("%d server assets to delete. skipped dry-run mode", len(ids)))
	return nil
}

/*
func (app *UpCmd) ManageAlbums(ctx context.Context) error {
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
					if !app.DryRun {
						app.Log.Info(fmt.Sprintf("Update the album %s", album))
						rr, err := app.Immich.AddAssetToAlbum(ctx, sal.ID, gen.MapKeys(list))
						if err != nil {
							return fmt.Errorf("can't update the album list from the server: %w", err)
						}
						added := 0
						for _, r := range rr {
							if r.Success {
								added++
							}
							if !r.Success && r.Error != "duplicate" {
								app.Log.Info(fmt.Sprintf("%s: %s", r.ID, r.Error))
							}
						}
						if added > 0 {
							app.Log.Info(fmt.Sprintf("%d asset(s) added to the album %q", added, album))
						}
					} else {
						app.Log.Info(fmt.Sprintf("Update album %s skipped - dry run mode", album))
					}
				}
			}
			if found {
				continue
			}
			if list != nil {
				if !app.DryRun {
					app.Log.Info(fmt.Sprintf("Create the album %s", album))

					_, err := app.Immich.CreateAlbum(ctx, album, gen.MapKeys(list))
					if err != nil {
						return fmt.Errorf("can't create the album list from the server: %w", err)
					}
				} else {
					app.Log.Info(fmt.Sprintf("Create the album %s skipped - dry run mode", album))
				}
			}
		}
	}
	return nil
}
*/
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
		dateTaken := la.Metadata.DateTaken
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
