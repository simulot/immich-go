// Command Upload

package cmdupload

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"immich-go/fshelper"
	"immich-go/immich"
	"immich-go/immich/assets"
	"immich-go/immich/logger"
	"immich-go/immich/metadata"
	"io/fs"
	"math"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
)

type UpCmd struct {
	Immich *immich.ImmichClient // Immich client

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
	ForceSidecar           bool             // Generate a sidecar file for each file (default: TRUE)

	AssetIndex       *AssetIndex              // List of assets present on the server
	deleteServerList []*immich.Asset          // List of server assets to remove
	deleteLocalList  []*assets.LocalAssetFile // List of local assets to remove
	mediaUploaded    int                      // Count uploaded medias
	mediaCount       int                      // Count of media on the source
	updateAlbums     map[string][]string      // track immich albums changes
	logger           *logger.Logger
}

func UploadCommand(ctx context.Context, ic *immich.ImmichClient, log *logger.Logger, args []string) error {
	app, err := NewUpCmd(ctx, ic, log, args)
	if err != nil {
		return err
	}

	log.MessageContinue(logger.OK, "Get server's assets...")
	var list []*immich.Asset
	list, err = app.Immich.GetAllAssets(ctx, nil)
	if err != nil {
		return err
	}
	log.MessageTerminate(logger.OK, " %d received", len(list))

	app.AssetIndex = &AssetIndex{
		assets: list,
	}

	app.AssetIndex.ReIndex()

	// Get server's albums
	app.AssetIndex.albums, err = ic.GetAllAlbums(ctx)
	if err != nil {
		return err
	}
	for _, album := range app.AssetIndex.albums {
		info, err := ic.GetAlbumInfo(ctx, album.ID)
		if err != nil {
			return err
		}
		for _, a := range info.Assets {
			as := app.AssetIndex.byID[a.DeviceAssetID]
			if as != nil {
				as.Albums = append(as.Albums, album)
			}
		}

	}

	fsys, err := fshelper.OpenMultiFile(app.Paths...)
	if err != nil {
		return err
	}

	var browser assets.Browser

	switch {
	case app.GooglePhotos:
		log.Info("Browsing google take out archive...")
		browser, err = app.ReadGoogleTakeOut(ctx, fsys)
	default:
		log.Info("Browsing folder(s)...")
		browser, err = app.ExploreLocalFolder(ctx, fsys)
	}

	if err != nil {
		return err
	}

	if app.CreateAlbums || app.CreateAlbumAfterFolder || len(app.ImportFromAlbum) > 0 {
		log.Info("Browsing local assets for findings albums")
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
			err = app.handleAsset(ctx, a)
			if err != nil {
				log.Warning("%s: %q", err.Error(), a.FileName)
			}

		}
	}

	if app.CreateAlbums || app.CreateAlbumAfterFolder || len(app.ImportIntoAlbum) > 0 {
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
			return fmt.Errorf("Can't delete server's assets: %w", err)
		}
	}

	if len(app.deleteLocalList) > 0 {
		err = app.DeleteLocalAssets()
	}
	return err
}

func (app *UpCmd) handleAsset(ctx context.Context, a *assets.LocalAssetFile) error {
	showCount := true
	defer func() {
		a.Close()
		if showCount {
			app.logger.Progress(logger.Info, "%d media scanned...", app.mediaCount)
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
		d := a.DateTaken
		if d.IsZero() {
			app.logger.Error("Can't get capture date of the file. File %q skipped", a.FileName)
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
	case NotOnServer:
		app.logger.Info("%s: %s", a.Title, advice.Message)
		app.UploadAsset(ctx, a)
		if app.Delete {
			app.deleteLocalList = append(app.deleteLocalList, a)
		}
	case SmallerOnServer:
		app.logger.Info("%s: %s", a.Title, advice.Message)

		// add the superior asset into albums of the original asset
		for _, al := range advice.ServerAsset.Albums {
			a.AddAlbum(al.AlbumName)
		}
		app.UploadAsset(ctx, a)

		app.deleteServerList = append(app.deleteServerList, advice.ServerAsset)
		if app.Delete {
			app.deleteLocalList = append(app.deleteLocalList, a)
		}
	case SameOnServer:
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
	case BetterOnServer:
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
	cmd := flag.NewFlagSet("upload", flag.ExitOnError)

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
	cmd.BoolVar(&app.ForceSidecar, "force-sidecar", false, "Upload the photo and a sidecar file with known information like date and GPS coordinates. With GooglePhotos, information comes from the metadata files. (DEFAULT false)")
	err = cmd.Parse(args)
	if err != nil {
		return nil, err
	}

	for _, f := range cmd.Args() {
		if !hasMeta(f) {
			app.Paths = append(app.Paths, f)
		} else {
			m, err := filepath.Glob(f)
			if err != nil {
				return nil, fmt.Errorf("can't use this file argument %q: %w", f, err)
			}
			app.Paths = append(app.Paths, m...)
		}
	}

	if len(app.Paths) == 0 {
		err = errors.Join(err, errors.New("must specify at least one path for local assets"))
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

func (app *UpCmd) UploadAsset(ctx context.Context, a *assets.LocalAssetFile) {
	var resp immich.AssetResponse
	app.logger.MessageContinue(logger.OK, "Uploading %q...", a.FileName)
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

func (app *UpCmd) DeleteServerAssets(ctx context.Context, ids []string) error {
	app.logger.Warning("%d server assets to delete.", len(ids))

	if !app.DryRun {
		_, err := app.Immich.DeleteAssets(ctx, ids)
		return err
	}
	app.logger.Warning("%d server assets to delete. skipped dry-run mode", len(ids))
	return nil
}

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
						rr, err := app.Immich.AddAssetToAlbum(ctx, sal.ID, list)
						if err != nil {
							return fmt.Errorf("can't update the album list from the server: %w", err)
						}
						added := 0
						for _, r := range rr {
							if r.Success {
								added++
							}
							if !r.Success && r.Error != "duplicate" {
								app.logger.Warning("%s: %s", r.ID, r.Error)
							}
						}
						if added > 0 {
							app.logger.OK("%d asset(s) added to the album %q", added, album)
						}
					} else {
						app.logger.OK("Update album %s skipped - dry run mode", album)
					}
				}
			}
			if found {
				continue
			}
			if list != nil {
				if !app.DryRun {
					app.logger.OK("Create the album %s", album)

					_, err := app.Immich.CreateAlbum(ctx, album, list)
					if err != nil {
						return fmt.Errorf("can't create the album list from the server: %w", err)
					}
				} else {
					app.logger.OK("Create the album %s skipped - dry run mode", album)
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
	filename := la.FileName
	var err error

	if fsys, ok := la.FSys.(assets.NameResolver); ok {
		filename, err = fsys.ResolveName(la)
		if err != nil {
			return nil, err
		}
	}
	ID := fmt.Sprintf("%s-%d", path.Base(la.Title), la.Size())

	sa := ai.byID[ID]
	if sa != nil {
		// the same ID exist on the server
		return ai.adviceSameOnServer(sa), nil
	}

	var l []*immich.Asset
	var n string

	// check all files with the same name

	n = filepath.Base(filename)
	l = ai.byName[n]
	if len(l) == 0 {
		n = strings.TrimSuffix(n, filepath.Ext(n))
		l = ai.byName[n]
	}

	if len(l) > 0 {
		dateTaken := la.DateTaken
		size := int(la.Size())
		if err != nil {
			return ai.adviceIDontKnow(la), nil

		}
		for _, sa = range l {
			compareDate := dateTaken.Compare(*sa.ExifInfo.DateTimeOriginal)
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

// hasMeta reports whether path contains any of the magic characters
// recognized by Match.
// shamelessly copied from stdlib/os
func hasMeta(path string) bool {
	magicChars := `*?[`
	if runtime.GOOS != "windows" {
		magicChars = `*?[\`
	}
	return strings.ContainsAny(path, magicChars)
}
