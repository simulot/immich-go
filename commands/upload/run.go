package upload

import (
	"context"
	"errors"
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/simulot/immich-go/adapters"
	gp "github.com/simulot/immich-go/adapters/googlePhotos"
	"github.com/simulot/immich-go/commands/application"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/filters"
)

type UpCmd struct {
	Mode UpLoadMode
	*UploadOptions
	app *application.Application

	AssetIndex       *AssetIndex     // List of assets present on the server
	deleteServerList []*immich.Asset // List of server assets to remove

	adapter       adapters.Reader
	DebugCounters bool // Enable CSV action counters per file

	Paths  []string                          // Path to explore
	albums map[string]immich.AlbumSimplified // Albums by title

	takeoutOptions *gp.ImportFlags
}

func newUpload(mode UpLoadMode, app *application.Application, options *UploadOptions) *UpCmd {
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

func (upCmd *UpCmd) run(ctx context.Context, adapter adapters.Reader, app *application.Application) error {
	upCmd.adapter = adapter

	if upCmd.NoUI {
		return upCmd.runNoUI(ctx, app)
	}
	_, err := tcell.NewScreen()
	if err != nil {
		upCmd.app.Log().Error("can't initialize the screen for the UI mode. Falling back to no-gui mode")
		fmt.Println("can't initialize the screen for the UI mode. Falling back to no-gui mode")
		return upCmd.runNoUI(ctx, app)
	}
	return upCmd.runUI(ctx, app)
}

func (upCmd *UpCmd) getImmichAlbums(ctx context.Context) error {
	serverAlbums, err := upCmd.app.Client().Immich.GetAllAlbums(ctx)
	upCmd.albums = map[string]immich.AlbumSimplified{}
	if err != nil {
		return fmt.Errorf("can't get the album list from the server: %w", err)
	}
	for _, a := range serverAlbums {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			upCmd.albums[a.AlbumName] = a
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
		upCmd.app.Jnl().Record(ctx, fileevent.DiscoveredDiscarded, fileevent.AsFileAndName(a.Asset.FSys, a.Asset.FileName), "reason", a.Reason)
	}

	// Upload assets from the group
	for _, a := range g.Assets {
		err := upCmd.handleAsset(ctx, g, a)
		errGroup = errors.Join(err)
	}

	// Manage albums
	if len(g.Albums) > 0 {
		upCmd.manageGroupAlbums(ctx, g)
	}

	// Manage groups
	// after the filtering and the upload, we can stack the assets

	if len(g.Assets) > 1 && g.Grouping != assets.GroupByNone {
		client := upCmd.app.Client().Immich.(immich.ImmichStackInterface)
		ids := []string{g.Assets[g.CoverIndex].ID}
		for i, a := range g.Assets {
			upCmd.app.Jnl().Record(ctx, fileevent.Stacked, fileevent.AsFileAndName(g.Assets[i].FSys, g.Assets[i].FileName))
			if i != g.CoverIndex {
				ids = append(ids, a.ID)
			}
		}
		_, err := client.CreateStack(ctx, ids)
		if err != nil {
			upCmd.app.Jnl().Log().Error("Can't create stack", "error", err)
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

func (upCmd *UpCmd) handleAsset(ctx context.Context, g *assets.Group, a *assets.Asset) error {
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
		return upCmd.manageAssetTags(ctx, a)
	case SmallerOnServer: // Upload, manage albums and delete the server's asset
		upCmd.app.Jnl().Record(ctx, fileevent.UploadUpgraded, a, "reason", advice.Message)

		// Remember existing asset's albums, if any
		for _, al := range advice.ServerAsset.Albums {
			g.AddAlbum(assets.Album{
				Title:       al.AlbumName,
				Description: al.Description,
			})
		}

		// Upload the superior asset
		err = upCmd.uploadAsset(ctx, a)
		if err != nil {
			return err
		}
		err = upCmd.manageAssetTags(ctx, a)
		if err != nil {
			return err
		}

		// delete the existing lower quality asset
		err = upCmd.app.Client().Immich.DeleteAssets(ctx, []string{advice.ServerAsset.ID}, true)
		if err != nil {
			upCmd.app.Jnl().Record(ctx, fileevent.Error, fileevent.FileAndName{}, "error", err.Error())
		}
		return err

	case SameOnServer:
		a.ID = advice.ServerAsset.ID
		for _, al := range advice.ServerAsset.Albums {
			g.AddAlbum(assets.Album{
				Title:       al.AlbumName,
				Description: al.Description,
			})
		}
		upCmd.app.Jnl().Record(ctx, fileevent.UploadServerDuplicate, fileevent.AsFileAndName(a.FSys, a.FileName), "reason", advice.Message)
		err = upCmd.manageAssetTags(ctx, a)
		if err != nil {
			return err
		}

	case BetterOnServer: // and manage albums
		a.ID = advice.ServerAsset.ID
		upCmd.app.Jnl().Record(ctx, fileevent.UploadServerBetter, fileevent.AsFileAndName(a.FSys, a.FileName), "reason", advice.Message)
		err = upCmd.manageAssetTags(ctx, a)
		if err != nil {
			return err
		}

	}

	return nil
}

// uploadAsset uploads the asset to the server.
// set the server's asset ID to the asset.
func (upCmd *UpCmd) uploadAsset(ctx context.Context, a *assets.Asset) error {
	defer upCmd.app.Log().Debug("", "file", a)
	ar, err := upCmd.app.Client().Immich.AssetUpload(ctx, a)
	if err != nil {
		upCmd.app.Jnl().Record(ctx, fileevent.UploadServerError, fileevent.AsFileAndName(a.FSys, a.FileName), "error", err.Error())
		return err // Must signal the error to the caller
	}
	if ar.Status == immich.UploadDuplicate {
		upCmd.app.Jnl().Record(ctx, fileevent.UploadServerDuplicate, fileevent.AsFileAndName(a.FSys, a.FileName), "reason", "the server has this file")
	} else {
		upCmd.app.Jnl().Record(ctx, fileevent.Uploaded, fileevent.AsFileAndName(a.FSys, a.FileName))
	}
	a.ID = ar.ID
	return nil
}

// manageGroupAlbums add the assets to the albums listed in the group.
// If an album does not exist, it is created.
// Errors are logged.
func (upCmd *UpCmd) manageGroupAlbums(ctx context.Context, g *assets.Group) {
	assetIDs := []string{}
	for _, a := range g.Assets {
		assetIDs = append(assetIDs, a.ID)
	}

	for _, album := range g.Albums {
		title := album.Title
		l, exist := upCmd.albums[title]
		if !exist {
			newAl, err := upCmd.app.Client().Immich.CreateAlbum(ctx, title, album.Description, assetIDs)
			if err != nil {
				upCmd.app.Jnl().Record(ctx, fileevent.Error, fileevent.FileAndName{}, err)
			}
			upCmd.albums[title] = newAl
			l = newAl
		} else {
			_, err := upCmd.app.Client().Immich.AddAssetToAlbum(ctx, l.ID, assetIDs)
			if err != nil {
				upCmd.app.Jnl().Record(ctx, fileevent.Error, fileevent.FileAndName{}, err)
				return
			}
		}

		// Log the action
		for _, a := range g.Assets {
			upCmd.app.Jnl().Record(ctx, fileevent.UploadAddToAlbum, fileevent.AsFileAndName(a.FSys, a.FileName), "Album", title)
		}
	}
}

func (upCmd *UpCmd) manageAssetTags(ctx context.Context, a *assets.Asset) error { // nolint
	if len(a.Tags) > 0 {
		ss := []string{}
		for _, t := range a.Tags {
			tags, err := upCmd.app.Client().Immich.UpsertTags(ctx, []string{t.Value})
			if err != nil {
				upCmd.app.Jnl().Record(ctx, fileevent.Error, fileevent.AsFileAndName(a.FSys, a.FileName), "error", err.Error())
				continue
			}
			for _, t := range tags {
				_, err = upCmd.app.Client().Immich.TagAssets(ctx, t.ID, []string{a.ID})
				if err != nil {
					upCmd.app.Jnl().Record(ctx, fileevent.Error, fileevent.AsFileAndName(a.FSys, a.FileName), "error", err.Error())
				}
				ss = append(ss, t.Value)
			}
		}
		upCmd.app.Jnl().Record(ctx, fileevent.Tagged, fileevent.AsFileAndName(a.FSys, a.FileName), "tags", ss)
	}
	return nil
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
