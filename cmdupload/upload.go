// Command Upload

package cmdupload

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"immich-go/assets"
	"immich-go/assets/files"
	"immich-go/assets/gp"
	"immich-go/helpers/fshelper"
	"immich-go/immich"
	"immich-go/immich/logger"
	"immich-go/immich/metadata"
	"io/fs"
	"math"
	"path"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// iClient is an interface that implements the minimal immich client set of features for uploading
type iClient interface {
	GetAllAssetsWithFilter(context.Context, *immich.GetAssetOptions, func(*immich.Asset)) error
	AssetUpload(context.Context, *assets.LocalAssetFile) (immich.AssetResponse, error)
	DeleteAssets(context.Context, []string) error

	GetAllAlbums(context.Context) ([]immich.AlbumSimplified, error)
	AddAssetToAlbum(context.Context, string, []string) ([]immich.UpdateAlbumResult, error)
	CreateAlbum(context.Context, string, []string) (immich.AlbumSimplified, error)
}

type UpCmd struct {
	client iClient        // Immich client
	log    *logger.Logger // Application loader
	fsys   fs.FS          // pseudo file system to browse

	Recursive              bool             // Explore sub folders
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

	AssetIndex       *AssetIndex               // List of assets present on the server
	deleteServerList []*immich.Asset           // List of server assets to remove
	deleteLocalList  []*assets.LocalAssetFile  // List of local assets to remove
	mediaUploaded    int                       // Count uploaded medias
	mediaCount       int                       // Count of media on the source
	updateAlbums     map[string]map[string]any // track immich albums changes
}

func NewUpCmd(ctx context.Context, ic iClient, log *logger.Logger, args []string) (*UpCmd, error) {
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

	// cmd.BoolVar(&app.Delete, "delete", false, "Delete local assets after upload")
	err = cmd.Parse(args)
	if err != nil {
		return nil, err
	}

	for _, f := range cmd.Args() {
		if !fshelper.HasMagic(f) {
			app.Paths = append(app.Paths, f)
		} else {
			m, err := filepath.Glob(f)
			if err != nil {
				return nil, fmt.Errorf("can't use this file argument %q: %w", f, err)
			}
			if len(m) == 0 {
				return nil, fmt.Errorf("no file matches %q", f)
			}
			app.Paths = append(app.Paths, m...)
		}
	}

	if len(app.Paths) == 0 {
		return nil, errors.Join(err, errors.New("must specify at least one path for local assets"))
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

	app.fsys, err = fshelper.OpenMultiFile(app.Paths...)
	if err != nil {
		return nil, err
	}

	return &app, err

}

func UploadCommand(ctx context.Context, ic iClient, log *logger.Logger, args []string) error {
	app, err := NewUpCmd(ctx, ic, log, args)
	if err != nil {
		return err
	}

	return app.Run(ctx)
}

func (app *UpCmd) Run(ctx context.Context) error {
	fsys := app.fsys
	log := app.log

	var browser assets.Browser
	var err error

	switch {
	case app.GooglePhotos:
		log.MessageContinue(logger.OK, "Browsing google take out archive...")
		browser, err = app.ReadGoogleTakeOut(ctx, fsys)
	default:
		log.MessageContinue(logger.OK, "Browsing folder(s)...")
		browser, err = app.ExploreLocalFolder(ctx, fsys)
	}

	if err != nil {
		log.MessageTerminate(logger.Error, err.Error())
		return err
	}
	log.MessageTerminate(logger.OK, "Done.")

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
				log.Warning("%s: %q", err.Error(), a.FileName)
			} else {
				err = app.handleAsset(ctx, a)
				if err != nil {
					log.Warning("%s: %q", err.Error(), a.FileName)
				}
			}
		}
	}

	if app.CreateAlbums || app.CreateAlbumAfterFolder || (app.KeepPartner && len(app.PartnerAlbum) > 0) || len(app.ImportIntoAlbum) > 0 {
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
			return fmt.Errorf("an't delete server's assets: %w", err)
		}
	}

	if len(app.deleteLocalList) > 0 {
		err = app.DeleteLocalAssets()
	}
	app.log.OK("%d media scanned, %d uploaded.", app.mediaCount, app.mediaUploaded)
	return err
}

func (app *UpCmd) handleAsset(ctx context.Context, a *assets.LocalAssetFile) error {
	showCount := true
	defer func() {
		a.Close()
		if showCount {
			app.log.Progress(logger.Info, "%d media scanned...", app.mediaCount)
		}
	}()
	app.mediaCount++

	ext := path.Ext(a.FileName)
	if _, err := fshelper.MimeFromExt(ext); err != nil {
		return nil
	}

	if !app.KeepPartner && a.FromPartner {
		return nil
	}

	if !app.KeepTrashed && a.Trashed {
		return nil
	}

	if len(app.ImportFromAlbum) > 0 && !app.isInAlbum(a, app.ImportFromAlbum) {
		return nil
	}

	if app.DateRange.IsSet() {
		d := a.DateTaken
		if d.IsZero() {
			app.log.Error("Can't get capture date of the file. File %q skipped", a.FileName)
			return nil
		}
		if !app.DateRange.InRange(d) {
			return nil
		}
	}
	app.log.DebugObject("handleAsset: LocalAssetFile=", a)

	advice, err := app.AssetIndex.ShouldUpload(a)
	if err != nil {
		return err
	}

	switch advice.Advice {
	case NotOnServer:
		app.log.Info("%s: %s", a.Title, advice.Message)
		app.UploadAsset(ctx, a)
		if app.Delete {
			app.deleteLocalList = append(app.deleteLocalList, a)
		}
	case SmallerOnServer:
		app.log.Info("%s: %s", a.Title, advice.Message)

		// add the superior asset into albums of the original asset
		for _, al := range advice.ServerAsset.Albums {
			a.AddAlbum(assets.LocalAlbum{Name: al.AlbumName})
		}
		app.UploadAsset(ctx, a)

		app.deleteServerList = append(app.deleteServerList, advice.ServerAsset)
		if app.Delete {
			app.deleteLocalList = append(app.deleteLocalList, a)
		}
	case SameOnServer:
		// Set add the server asset into albums determined locally
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
	showCount = false
	return nil

}

func (app *UpCmd) isInAlbum(a *assets.LocalAssetFile, album string) bool {
	for _, al := range a.Albums {
		if app.albumName(al) == album {
			return true
		}
	}
	return false
}

func (a *UpCmd) ReadGoogleTakeOut(ctx context.Context, fsys fs.FS) (assets.Browser, error) {
	a.Delete = false
	return gp.NewTakeout(ctx, fsys)
}

func (a *UpCmd) ExploreLocalFolder(ctx context.Context, fsys fs.FS) (assets.Browser, error) {
	return files.NewLocalFiles(ctx, fsys)
}

// UploadAsset upload the asset on the server
// Add the assets into listed albums

func (app *UpCmd) UploadAsset(ctx context.Context, a *assets.LocalAssetFile) {
	var resp immich.AssetResponse
	app.log.MessageContinue(logger.OK, "Uploading %q...", a.FileName)
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
			app.log.MessageTerminate(logger.Error, "Error: %s", err)
			return
		}
	} else {
		resp.ID = uuid.NewString()
	}
	if !resp.Duplicate {
		app.AssetIndex.AddLocalAsset(a, resp.ID)
		app.mediaUploaded += 1
		if !app.DryRun {
			app.log.Progress(logger.OK, "Done, total %d uploaded", app.mediaUploaded)
		} else {
			app.log.Progress(logger.OK, "Skipped - dry run mode, total %d uploaded", app.mediaUploaded)

		}

		if app.ImportIntoAlbum != "" ||
			(app.GooglePhotos && (app.CreateAlbums || app.PartnerAlbum != "")) ||
			(!app.GooglePhotos && app.CreateAlbumAfterFolder) {
			albums := []assets.LocalAlbum{}

			if app.ImportIntoAlbum != "" {
				albums = append(albums, assets.LocalAlbum{Path: app.ImportIntoAlbum, Name: app.ImportIntoAlbum})
			} else {
				switch {
				case app.GooglePhotos:
					for _, al := range a.Albums {
						albums = append(albums, al)
					}
					if app.PartnerAlbum != "" && a.FromPartner {
						albums = append(albums, assets.LocalAlbum{Path: app.PartnerAlbum, Name: app.PartnerAlbum})
					}
				case !app.GooglePhotos && app.CreateAlbumAfterFolder:
					album := path.Base(path.Dir(a.FileName))
					if album != "" && album != "." {
						albums = append(albums, assets.LocalAlbum{Path: album, Name: album})
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
					app.log.Progress(logger.OK, " added in albums(s) ")
					for _, n := range Names {
						app.log.Progress(logger.OK, " %s", n)
						app.AddToAlbum(resp.ID, n)
					}
				}
			}
		}
		app.log.MessageTerminate(logger.OK, "")
	} else {
		app.log.MessageTerminate(logger.Warning, "already exists on the server")
	}
}

func (app *UpCmd) albumName(al assets.LocalAlbum) string {
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
		err := app.client.DeleteAssets(ctx, ids)
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
	LocalAsset  *assets.LocalAssetFile
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

func (ai *AssetIndex) adviceIDontKnow(la *assets.LocalAssetFile) *Advice {
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

func (ai *AssetIndex) ShouldUpload(la *assets.LocalAssetFile) (*Advice, error) {
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
