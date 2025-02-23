package upload

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/gdamore/tcell/v2"
	"github.com/simulot/immich-go/adapters"
	gp "github.com/simulot/immich-go/adapters/googlePhotos"
	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/bulktags"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/filters"
	"github.com/simulot/immich-go/internal/fshelper"
)

type UpCmd struct {
	Mode UpLoadMode
	*UploadOptions
	app *app.Application

	AssetIndex       *AssetIndex     // List of assets present on the server
	deleteServerList []*immich.Asset // List of server assets to remove

	adapter       adapters.Reader
	DebugCounters bool // Enable CSV action counters per file

	Paths  []string                // Path to explore
	albums map[string]assets.Album // Albums by title

	tm *bulktags.BulkTagManager // Bulk tag manager

	takeoutOptions *gp.ImportFlags
}

func newUpload(mode UpLoadMode, app *app.Application, options *UploadOptions) *UpCmd {
	upCmd := &UpCmd{
		UploadOptions: options,
		app:           app,
		Mode:          mode,
	}
	return upCmd
}

func (upCmd *UpCmd) setTakeoutOptions(options *gp.ImportFlags) *UpCmd {
	upCmd.takeoutOptions = options
	return upCmd
}

func (upCmd *UpCmd) run(ctx context.Context, adapter adapters.Reader, app *app.Application, fsys []fs.FS) error {
	upCmd.adapter = adapter
	upCmd.tm = bulktags.NewBulkTagManager(ctx, app.Client().Immich, app.Log().Logger)
	defer upCmd.tm.Close()

	runner := upCmd.runUI

	if upCmd.NoUI {
		runner = upCmd.runNoUI
	}
	_, err := tcell.NewScreen()
	if err != nil {
		upCmd.app.Log().Error("can't initialize the screen for the UI mode. Falling back to no-gui mode")
		fmt.Println("can't initialize the screen for the UI mode. Falling back to no-gui mode")
		runner = upCmd.runNoUI
	}
	err = runner(ctx, app)

	err = errors.Join(err, fshelper.CloseFSs(fsys))
	app.Jnl().Report()

	return err
}

func (upCmd *UpCmd) getImmichAlbums(ctx context.Context) error {
	serverAlbums, err := upCmd.app.Client().Immich.GetAllAlbums(ctx)
	upCmd.albums = map[string]assets.Album{}
	if err != nil {
		return fmt.Errorf("can't get the album list from the server: %w", err)
	}
	for _, a := range serverAlbums {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			upCmd.albums[a.Title] = a
		}
	}
	return nil
}

func (upCmd *UpCmd) getImmichAssets(ctx context.Context, updateFn progressUpdate) error {
	statistics, err := upCmd.app.Client().Immich.GetAssetStatistics(ctx)
	if err != nil {
		return err
	}
	totalOnImmich := statistics.Total
	received := 0

	var list []*immich.Asset

	err = upCmd.app.Client().Immich.GetAllAssetsWithFilter(ctx, nil, func(a *immich.Asset) error {
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
	upCmd.AssetIndex = &AssetIndex{
		assets: list,
	}
	upCmd.app.Log().Info(fmt.Sprintf("Assets on the server: %d", len(list)))
	upCmd.AssetIndex.ReIndex()
	return nil
}

func (upCmd *UpCmd) uploadLoop(ctx context.Context, groupChan chan *assets.Group) error {
	var err error
assetLoop:
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case g, ok := <-groupChan:
			if !ok {
				break assetLoop
			}
			err = upCmd.handleGroup(ctx, g)
			if err != nil {
				return err
			}
		}
	}

	if len(upCmd.deleteServerList) > 0 {
		ids := []string{}
		for _, da := range upCmd.deleteServerList {
			ids = append(ids, da.ID)
		}
		err := upCmd.DeleteServerAssets(ctx, ids)
		if err != nil {
			return fmt.Errorf("can't delete server's assets: %w", err)
		}
	}

	return err
}

func (upCmd *UpCmd) handleGroup(ctx context.Context, g *assets.Group) error {
	var errGroup error

	g = filters.ApplyFilters(g, upCmd.UploadOptions.Filters...)

	// discard rejected assets
	for _, a := range g.Removed {
		a.Asset.Close()
		upCmd.app.Jnl().Record(ctx, fileevent.DiscoveredDiscarded, a.Asset.File, "reason", a.Reason)
	}

	// Upload assets from the group
	for _, a := range g.Assets {
		err := upCmd.handleAsset(ctx, a)
		errGroup = errors.Join(err)
	}

	// Manage groups
	// after the filtering and the upload, we can stack the assets

	if len(g.Assets) > 1 && g.Grouping != assets.GroupByNone {
		client := upCmd.app.Client().Immich.(immich.ImmichStackInterface)
		ids := []string{g.Assets[g.CoverIndex].ID}
		for i, a := range g.Assets {
			upCmd.app.Jnl().Record(ctx, fileevent.Stacked, g.Assets[i].File)
			if i != g.CoverIndex && a.ID != "" {
				ids = append(ids, a.ID)
			}
		}
		if len(ids) > 1 {
			_, err := client.CreateStack(ctx, ids)
			if err != nil {
				upCmd.app.Jnl().Log().Error("Can't create stack", "error", err)
			}
		}
	}

	if errGroup != nil {
		return errGroup
	}

	switch g.Grouping {
	case assets.GroupByNone:
	}

	return nil
}

func (upCmd *UpCmd) handleAsset(ctx context.Context, a *assets.Asset) error {
	defer func() {
		a.Close() // Close and clean resources linked to the local asset
	}()

	advice, err := upCmd.AssetIndex.ShouldUpload(a)
	if err != nil {
		return err
	}

	switch advice.Advice {
	case NotOnServer: // Upload and manage albums
		err = upCmd.uploadAsset(ctx, a)
		if err != nil {
			return err
		}

		// Manage albums
		if len(a.Albums) > 0 {
			upCmd.manageAssetAlbums(ctx, a.File, a.ID, a.Albums)
		}
		upCmd.manageAssetTags(ctx, a)
		return nil
	case SmallerOnServer: // Upload, manage albums and delete the server's asset

		// Remember existing asset's albums, if any
		for _, al := range advice.ServerAsset.Albums {
			a.Albums = append(a.Albums, assets.Album{
				Title:       al.AlbumName,
				Description: al.Description,
			})
		}

		// Upload the superior asset
		err = upCmd.replaceAsset(ctx, advice.ServerAsset.ID, a)
		if err != nil {
			return err
		}
		upCmd.app.Jnl().Record(ctx, fileevent.UploadUpgraded, a, "reason", advice.Message)

		// Manage albums
		if len(a.Albums) > 0 {
			upCmd.manageAssetAlbums(ctx, a.File, advice.ServerAsset.ID, a.Albums)
		}

		upCmd.manageAssetTags(ctx, a)
		return err

	case SameOnServer:
		a.ID = advice.ServerAsset.ID
		for _, al := range advice.ServerAsset.Albums {
			a.Albums = append(a.Albums, assets.Album{
				Title:       al.AlbumName,
				Description: al.Description,
			})
		}
		upCmd.app.Jnl().Record(ctx, fileevent.UploadServerDuplicate, a.File, "reason", advice.Message)
		upCmd.manageAssetAlbums(ctx, a.File, a.ID, a.Albums)

	case BetterOnServer: // and manage albums
		a.ID = advice.ServerAsset.ID
		upCmd.app.Jnl().Record(ctx, fileevent.UploadServerBetter, a.File, "reason", advice.Message)
		upCmd.manageAssetAlbums(ctx, a.File, a.ID, a.Albums)
	}
	return nil
}

// uploadAsset uploads the asset to the server.
// set the server's asset ID to the asset.
func (upCmd *UpCmd) uploadAsset(ctx context.Context, a *assets.Asset) error {
	defer upCmd.app.Log().Debug("", "file", a)
	ar, err := upCmd.app.Client().Immich.AssetUpload(ctx, a)
	if err != nil {
		upCmd.app.Jnl().Record(ctx, fileevent.UploadServerError, a.File, "error", err.Error())
		return err // Must signal the error to the caller
	}
	if ar.Status == immich.UploadDuplicate {
		upCmd.app.Jnl().Record(ctx, fileevent.UploadServerDuplicate, a.File, "reason", "the server has this file")
	} else {
		upCmd.app.Jnl().Record(ctx, fileevent.Uploaded, a.File)
	}
	a.ID = ar.ID

	if a.FromApplication != nil {
		// metadata from application (immich or google photos) are forced.
		// if a.Description != "" || (a.Latitude != 0 && a.Longitude != 0) || a.Rating != 0 || !a.CaptureDate.IsZero() {
		a.UseMetadata(a.FromApplication)
		_, err := upCmd.app.Client().Immich.UpdateAsset(ctx, a.ID, immich.UpdAssetField{
			Description:      a.Description,
			Latitude:         a.Latitude,
			Longitude:        a.Longitude,
			Rating:           a.Rating,
			DateTimeOriginal: a.CaptureDate,
		})
		if err != nil {
			upCmd.app.Jnl().Record(ctx, fileevent.UploadServerError, a.File, "error", err.Error())
			return err
		}
	}
	return nil
}

func (upCmd *UpCmd) replaceAsset(ctx context.Context, ID string, a *assets.Asset) error {
	defer upCmd.app.Log().Debug("replaced by", "file", a)
	ar, err := upCmd.app.Client().Immich.ReplaceAsset(ctx, ID, a)
	if err != nil {
		upCmd.app.Jnl().Record(ctx, fileevent.UploadServerError, a.File, "error", err.Error())
		return err // Must signal the error to the caller
	}
	if ar.Status == immich.UploadDuplicate {
		upCmd.app.Jnl().Record(ctx, fileevent.UploadServerDuplicate, a.File, "reason", "the server has this file")
	} else {
		a.ID = ID
		upCmd.app.Jnl().Record(ctx, fileevent.UploadUpgraded, a.File)
	}
	return nil
}

// manageAssetAlbums add the assets to the albums listed.
// If an album does not exist, it is created.
// Errors are logged.
func (upCmd *UpCmd) manageAssetAlbums(ctx context.Context, f fshelper.FSAndName, ID string, albums []assets.Album) {
	for _, album := range albums {
		title := album.Title
		l, exist := upCmd.albums[title]
		if !exist {
			newAl, err := upCmd.app.Client().Immich.CreateAlbum(ctx, title, album.Description, []string{ID})
			if err != nil {
				upCmd.app.Jnl().Record(ctx, fileevent.Error, nil, "error", err)
			}
			upCmd.albums[title] = newAl
			l = newAl
		} else {
			_, err := upCmd.app.Client().Immich.AddAssetToAlbum(ctx, l.ID, []string{ID})
			if err != nil {
				upCmd.app.Jnl().Record(ctx, fileevent.Error, nil, "error", err)
				return
			}
		}

		// Log the action
		upCmd.app.Jnl().Record(ctx, fileevent.UploadAddToAlbum, f, "Album", title)
	}
}

func (upCmd *UpCmd) manageAssetTags(ctx context.Context, a *assets.Asset) {
	if len(a.Tags) > 0 {
		// Get asset's tags
		for _, t := range a.Tags {
			upCmd.tm.AddTag(t.Value, a.ID)
			upCmd.app.Jnl().Record(ctx, fileevent.Tagged, a.File, "tags", t.Name)
		}
	}
}

func (upCmd *UpCmd) DeleteServerAssets(ctx context.Context, ids []string) error {
	upCmd.app.Log().Message("%d server assets to delete.", len(ids))
	return upCmd.app.Client().Immich.DeleteAssets(ctx, ids, false)
}

/*
func (app *UpCmd) DeleteLocalAssets() error {
	app.RootImmichFlags.Message(fmt.Sprintf("%d local assets to delete.", len(app.deleteLocalList)))

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
*/
