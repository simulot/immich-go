// Command Upload

package cmdupload

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"immich-go/browser"
	"immich-go/browser/files"
	"immich-go/browser/gp"
	"immich-go/helpers/fshelper"
	"immich-go/helpers/gen"
	"immich-go/helpers/stacking"
	"immich-go/immich"
	"immich-go/immich/metadata"
	"immich-go/journal"
	"immich-go/logger"
	"io/fs"
	"math"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// iClient is an interface that implements the minimal immich client set of features for uploading
type iClient interface {
	GetAllAssetsWithFilter(context.Context, *immich.GetAssetOptions, func(*immich.Asset)) error
	AssetUpload(context.Context, *browser.LocalAssetFile) (immich.AssetResponse, error)
	DeleteAssets(context.Context, []string, bool) error

	GetAllAlbums(context.Context) ([]immich.AlbumSimplified, error)
	AddAssetToAlbum(context.Context, string, []string) ([]immich.UpdateAlbumResult, error)
	CreateAlbum(context.Context, string, []string) (immich.AlbumSimplified, error)
	UpdateAssets(ctx context.Context, IDs []string, isArchived bool, isFavorite bool, removeParent bool, stackParentId string) error
}

type UpCmd struct {
	client iClient       // Immich client
	log    logger.Logger // Application loader
	fsys   []fs.FS       // pseudo file system to browse

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

	BrowserConfig browser.Configuration

	AssetIndex       *AssetIndex               // List of assets present on the server
	deleteServerList []*immich.Asset           // List of server assets to remove
	deleteLocalList  []*browser.LocalAssetFile // List of local assets to remove
	mediaUploaded    int                       // Count uploaded medias
	mediaCount       int                       // Count of media on the source
	updateAlbums     map[string]map[string]any // track immich albums changes
	stacks           *stacking.StackBuilder
}

func NewUpCmd(ctx context.Context, ic iClient, log logger.Logger, args []string) (*UpCmd, error) {
	var err error
	cmd := flag.NewFlagSet("upload", flag.ExitOnError)

	app := UpCmd{
		updateAlbums: map[string]map[string]any{},
		log:          log,
		client:       ic,
	}
	cmd.BoolVar(&app.DryRun,
		"dry-run",
		false,
		"display actions but don't touch source or destination")
	cmd.Var(&app.DateRange,
		"date",
		"Date of capture range.")
	cmd.StringVar(&app.ImportIntoAlbum,
		"album",
		"",
		"All assets will be added to this album.")
	cmd.BoolVar(&app.ForceSidecar,
		"force-sidecar",
		false,
		"Upload the photo and a sidecar file with known information like date and GPS coordinates. With google-photos, information comes from the metadata files. (DEFAULT false)")
	cmd.BoolVar(&app.CreateAlbumAfterFolder,
		"create-album-folder",
		false,
		" folder import only: Create albums for assets based on the parent folder")

	cmd.BoolVar(&app.GooglePhotos,
		"google-photos",
		false,
		"Import GooglePhotos takeout zip files")
	cmd.BoolVar(&app.CreateAlbums,
		"create-albums",
		true,
		" google-photos only: Create albums like there were in the source")
	cmd.StringVar(&app.PartnerAlbum,
		"partner-album",
		"",
		" google-photos only: Assets from partner will be added to this album. (ImportIntoAlbum, must already exist)")
	cmd.BoolVar(&app.KeepPartner,
		"keep-partner",
		true,
		" google-photos only: Import also partner's items")
	cmd.StringVar(&app.ImportFromAlbum,
		"from-album",
		"",
		" google-photos only: Import only from this album")

	cmd.BoolVar(&app.KeepUntitled,
		"keep-untitled-albums",
		false,
		" google-photos only: Keep Untitled albums and imports their contain")

	cmd.BoolVar(&app.UseFolderAsAlbumName,
		"use-album-folder-as-name",
		false,
		" google-photos only: Use folder name and ignore albums' title")

	cmd.BoolVar(&app.CreateStacks,
		"create-stacks",
		true,
		"Stack jpg/raw or bursts  (default TRUE)")

	// cmd.BoolVar(&app.Delete, "delete", false, "Delete local assets after upload")

	cmd.Var(&app.BrowserConfig.SelectExtensions, "select-types", "list of selected extensions separated by a comma")
	cmd.Var(&app.BrowserConfig.ExcludeExtensions, "exclude-types", "list of excluded extensions separated by a comma")

	err = cmd.Parse(args)
	if err != nil {
		return nil, err
	}

	if err = app.BrowserConfig.IsValid(); err != nil {
		return nil, err
	}

	app.BrowserConfig.Journal = journal.NewJournal(app.log)

	app.fsys, err = fshelper.ParsePath(cmd.Args(), app.GooglePhotos)
	if err != nil {
		return nil, err
	}

	if app.CreateStacks {
		app.stacks = stacking.NewStackBuilder()
	}
	log.OK("Ask for server's assets...")
	var list []*immich.Asset
	err = app.client.GetAllAssetsWithFilter(ctx, nil, func(a *immich.Asset) {
		if a.IsTrashed {
			return
		}
		list = append(list, a)
	})
	if err != nil {
		return nil, err
	}
	log.OK("%d asset(s) received", len(list))

	app.AssetIndex = &AssetIndex{
		assets: list,
	}

	app.AssetIndex.ReIndex()

	return &app, err

}

func UploadCommand(ctx context.Context, ic iClient, log logger.Logger, args []string) error {
	app, err := NewUpCmd(ctx, ic, log, args)
	if err != nil {
		return err
	}

	for _, fsys := range app.fsys {
		err = errors.Join(app.Run(ctx, fsys))
	}
	return err
}

func (app *UpCmd) journalAsset(a *browser.LocalAssetFile, action journal.Action, comment string) {
	app.BrowserConfig.Journal.AddEntry(a.FileName, action, comment)
	if len(a.LivePhotoData) > 0 {
		app.BrowserConfig.Journal.AddEntry(a.LivePhotoData, action, comment)
	}
}

func (app *UpCmd) Run(ctx context.Context, fsys fs.FS) error {
	log := app.log

	var browser browser.Browser
	var err error

	switch {
	case app.GooglePhotos:
		log.Message(logger.OK, "Browsing google take out archive...")
		browser, err = app.ReadGoogleTakeOut(ctx, fsys)
	default:
		log.Message(logger.OK, "Browsing folder(s)...")
		browser, err = app.ExploreLocalFolder(ctx, fsys)
	}

	if err != nil {
		log.Message(logger.Error, err.Error())
		return err
	}
	log.Message(logger.OK, "Done.")

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
			if a.Err != nil {
				app.journalAsset(a, journal.ERROR, err.Error())
			} else {
				err = app.handleAsset(ctx, a)
				if err != nil {
					app.journalAsset(a, journal.ERROR, err.Error())
				}
			}
		}
	}

	if app.CreateStacks {
		stacks := app.stacks.Stacks()
		if len(stacks) > 0 {
			log.OK("Creating stacks")
			for _, s := range stacks {
				log.OK("  Stacking %s...", strings.Join(s.Names, ", "))
				if !app.DryRun {
					err = app.client.UpdateAssets(ctx, s.IDs, false, false, false, s.CoverID)
					if err != nil {
						log.Warning("Can't stack images: %s", err)
					}
				}
			}
		}
	}

	if app.CreateAlbums || app.CreateAlbumAfterFolder || (app.KeepPartner && len(app.PartnerAlbum) > 0) || len(app.ImportIntoAlbum) > 0 {
		log.OK("Managing albums")
		err = app.ManageAlbums(ctx)
		if err != nil {
			log.Error(err.Error())
			err = nil
		}
	}

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

	app.BrowserConfig.Journal.Report()
	counts := app.BrowserConfig.Journal.Counters()

	if c := counts[journal.UNHANDLED] + counts[journal.ERROR] + counts[journal.UNSUPPORTED]; c > 0 {
		app.log.Warning("%d files can't be handled", c)
		app.BrowserConfig.Journal.WriteJournal(journal.UNHANDLED, journal.ERROR, journal.UNSUPPORTED)
	}

	return err
}

func (app *UpCmd) handleAsset(ctx context.Context, a *browser.LocalAssetFile) error {
	defer func() {
		a.Close()
	}()
	app.mediaCount++

	ext := path.Ext(a.FileName)
	if _, err := fshelper.MimeFromExt(ext); err != nil {
		app.journalAsset(a, journal.DISCARDED, "not recognized extension")
		return nil
	}

	if !app.KeepPartner && a.FromPartner {
		app.journalAsset(a, journal.DISCARDED, "partners discarded")
		return nil
	}

	if !app.KeepTrashed && a.Trashed {
		app.journalAsset(a, journal.DISCARDED, "trashed discarded")
		return nil
	}

	if len(app.ImportFromAlbum) > 0 && !app.isInAlbum(a, app.ImportFromAlbum) {
		app.journalAsset(a, journal.DISCARDED, "not in requested album")
		return nil
	}

	if app.DateRange.IsSet() {
		d := a.DateTaken
		if d.IsZero() {
			app.journalAsset(a, journal.DISCARDED, "date range import, impossible to get the date of capture")
			return nil
		}
		if !app.DateRange.InRange(d) {
			app.journalAsset(a, journal.DISCARDED, "date of capture out of the date range")
			return nil
		}
	}

	if !app.KeepUntitled {
		a.Albums = gen.Filter(a.Albums, func(i browser.LocalAlbum) bool {
			return i.Name != ""
		})
	}

	app.log.DebugObject("handleAsset: LocalAssetFile=", a)

	advice, err := app.AssetIndex.ShouldUpload(a)
	if err != nil {
		return err
	}

	switch advice.Advice {
	case NotOnServer:
		app.UploadAsset(ctx, a)
		if app.Delete {
			app.deleteLocalList = append(app.deleteLocalList, a)
		}
	case SmallerOnServer:
		app.journalAsset(a, journal.UPGRADED, "")
		// add the superior asset into albums of the original asset
		for _, al := range advice.ServerAsset.Albums {
			a.AddAlbum(browser.LocalAlbum{Name: al.AlbumName})
		}
		app.UploadAsset(ctx, a)

		app.deleteServerList = append(app.deleteServerList, advice.ServerAsset)
		if app.Delete {
			app.deleteLocalList = append(app.deleteLocalList, a)
		}
	case SameOnServer:
		// Set add the server asset into albums determined locally
		if !advice.ServerAsset.JustUploaded {
			app.journalAsset(a, journal.SERVER_DUPLICATE, "")
		} else {
			app.journalAsset(a, journal.LOCAL_DUPLICATE, "")
		}
		if app.CreateAlbums {
			for _, al := range a.Albums {
				app.AddToAlbum(advice.ServerAsset.ID, app.albumName(al))
			}
		}
		if app.ImportIntoAlbum != "" {
			app.AddToAlbum(advice.ServerAsset.ID, app.ImportIntoAlbum)
		}
		if app.PartnerAlbum != "" && a.FromPartner {
			app.AddToAlbum(advice.ServerAsset.ID, app.PartnerAlbum)
		}
		if !advice.ServerAsset.JustUploaded {
			app.log.Info("%s: %s", a.Title, advice.Message)
			if app.Delete {
				app.deleteLocalList = append(app.deleteLocalList, a)
			}
		} else {
			return nil
		}
	case BetterOnServer:
		app.journalAsset(a, journal.SERVER_BETTER, "")
		// keep the server version but update albums
		if app.CreateAlbums {
			for _, al := range a.Albums {
				app.AddToAlbum(advice.ServerAsset.ID, app.albumName(al))
			}
		}
		if app.PartnerAlbum != "" && a.FromPartner {
			app.AddToAlbum(advice.ServerAsset.ID, app.PartnerAlbum)
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

func (a *UpCmd) ReadGoogleTakeOut(ctx context.Context, fsys fs.FS) (browser.Browser, error) {
	a.Delete = false
	return gp.NewTakeout(ctx, fsys, a.log, &a.BrowserConfig)
}

func (a *UpCmd) ExploreLocalFolder(ctx context.Context, fsys fs.FS) (browser.Browser, error) {
	return files.NewLocalFiles(ctx, fsys, a.log, &a.BrowserConfig)
}

// UploadAsset upload the asset on the server
// Add the assets into listed albums

func (app *UpCmd) UploadAsset(ctx context.Context, a *browser.LocalAssetFile) {
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

		resp, err = app.client.AssetUpload(ctx, a)
		if err != nil {
			app.journalAsset(a, journal.ERROR, err.Error())
			return
		}
	} else {
		resp.ID = uuid.NewString()
	}
	if !resp.Duplicate {
		app.journalAsset(a, journal.UPLOADED, a.Title)
		app.AssetIndex.AddLocalAsset(a, resp.ID)
		app.mediaUploaded += 1
		if app.CreateStacks {
			app.stacks.ProcessAsset(resp.ID, a.FileName, a.DateTaken)
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
					for _, al := range a.Albums {
						albums = append(albums, al)
					}
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
					app.log.DebugObject("Add asset to the album:", al)

					if app.GooglePhotos && Name == "" {
						continue
					}
					Names = append(Names, Name)
				}
				if len(Names) > 0 {
					app.journalAsset(a, journal.ALBUM, strings.Join(Names, ", "))
					for _, n := range Names {
						app.AddToAlbum(resp.ID, n)
					}
				}
			}
		}
	} else {
		app.journalAsset(a, journal.SERVER_DUPLICATE, "already on the server")
	}
}

func (app *UpCmd) albumName(al browser.LocalAlbum) string {
	app.log.DebugObject("albumName: ", al)
	Name := al.Name
	if app.GooglePhotos {
		switch {
		case app.UseFolderAsAlbumName:
			Name = al.Path
		case app.KeepUntitled && Name == "":
			Name = al.Path
		}
	}
	return Name
}

func (app *UpCmd) AddToAlbum(ID string, album string) {
	l := app.updateAlbums[album]
	if l == nil {
		l = map[string]any{}
	}
	l[ID] = nil
	app.updateAlbums[album] = l
}

func (app *UpCmd) DeleteLocalAssets() error {
	app.log.OK("%d local assets to delete.", len(app.deleteLocalList))

	for _, a := range app.deleteLocalList {
		if !app.DryRun {
			app.log.Warning("delete file %q", a.Title)
			err := a.Remove()
			if err != nil {
				return err
			}
		} else {
			app.log.Warning("file %q not deleted, dry run mode", a.Title)
		}

	}
	return nil
}

func (app *UpCmd) DeleteServerAssets(ctx context.Context, ids []string) error {
	app.log.Warning("%d server assets to delete.", len(ids))

	if !app.DryRun {
		err := app.client.DeleteAssets(ctx, ids, false)
		return err
	}
	app.log.Warning("%d server assets to delete. skipped dry-run mode", len(ids))
	return nil
}

func (app *UpCmd) ManageAlbums(ctx context.Context) error {
	if len(app.updateAlbums) > 0 {
		serverAlbums, err := app.client.GetAllAlbums(ctx)
		if err != nil {
			return fmt.Errorf("can't get the album list from the server: %w", err)
		}
		for album, list := range app.updateAlbums {

			found := false
			for _, sal := range serverAlbums {
				if sal.AlbumName == album {
					found = true
					if !app.DryRun {
						app.log.OK("Update the album %s", album)
						rr, err := app.client.AddAssetToAlbum(ctx, sal.ID, keys(list))
						if err != nil {
							return fmt.Errorf("can't update the album list from the server: %w", err)
						}
						added := 0
						for _, r := range rr {
							if r.Success {
								added++
							}
							if !r.Success && r.Error != "duplicate" {
								app.log.Warning("%s: %s", r.ID, r.Error)
							}
						}
						if added > 0 {
							app.log.OK("%d asset(s) added to the album %q", added, album)
						}
					} else {
						app.log.OK("Update album %s skipped - dry run mode", album)
					}
				}
			}
			if found {
				continue
			}
			if list != nil {
				if !app.DryRun {
					app.log.OK("Create the album %s", album)

					_, err := app.client.CreateAlbum(ctx, album, keys(list))
					if err != nil {
						return fmt.Errorf("can't create the album list from the server: %w", err)
					}
				} else {
					app.log.OK("Create the album %s skipped - dry run mode", album)
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

func (ai *AssetIndex) adviceIDontKnow(la *browser.LocalAssetFile) *Advice {
	return &Advice{
		Advice:     IDontKnow,
		Message:    fmt.Sprintf("Can't decide what to do with %q. Check this vile yourself", la.FileName),
		LocalAsset: la,
	}
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
//
//

func (ai *AssetIndex) ShouldUpload(la *browser.LocalAssetFile) (*Advice, error) {
	filename := la.Title
	if path.Ext(filename) == "" {
		filename += path.Ext(la.FileName)
	}
	var err error
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
		if err != nil {
			return ai.adviceIDontKnow(la), nil

		}
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

func keys[M ~map[K]V, K comparable, V any](m M) []K {
	r := make([]K, 0, len(m))
	for k := range m {
		r = append(r, k)
	}
	return r
}
