package upload

import (
	"context"
	"errors"
	"fmt"
	"path"

	"github.com/gdamore/tcell/v2"
	"github.com/simulot/immich-go/adapters"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/metadata"
)

func (app *UpCmd) run(ctx context.Context, adapter adapters.Adapter) error {
	// if app.CommonFlags.StackBurstPhotos || app.CommonFlags.StackJpgWithRaw {
	// 	app.stacks = stacking.NewStackBuilder(app.ImmichServerFlags.Immich.SupportedMedia())
	// }

	// todo counters
	// defer func() {
	// 	if app.DebugCounters {
	// 		fn := strings.TrimSuffix(app.LogFile, filepath.Ext(app.LogFile)) + ".csv"
	// 		f, err := os.Create(fn)
	// 		if err == nil {
	// 			_ = app.Jnl.WriteFileCounts(f)
	// 			fmt.Println("\nCheck the counters file: ", f.Name())
	// 			f.Close()
	// 		}
	// 	}
	// }()

	app.browser = adapter

	if app.NoUI {
		return app.runNoUI(ctx)
	}
	_, err := tcell.NewScreen()
	if err != nil {
		app.Root.Log.Error("can't initialize the screen for the UI mode. Falling back to no-gui mode")
		fmt.Println("can't initialize the screen for the UI mode. Falling back to no-gui mode")
		return app.runNoUI(ctx)
	}
	return app.runUI(ctx)
}

func (app *UpCmd) getImmichAlbums(ctx context.Context) error {
	serverAlbums, err := app.Server.Immich.GetAllAlbums(ctx)
	app.albums = map[string]*immich.AlbumSimplified{}
	if err != nil {
		return fmt.Errorf("can't get the album list from the server: %w", err)
	}
	for _, a := range serverAlbums {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			app.albums[a.AlbumName] = &a
		}
	}
	return nil
}

func (app *UpCmd) getImmichAssets(ctx context.Context, updateFn progressUpdate) error {
	statistics, err := app.Server.Immich.GetAssetStatistics(ctx)
	if err != nil {
		return err
	}
	totalOnImmich := statistics.Total
	received := 0

	var list []*immich.Asset

	err = app.Server.Immich.GetAllAssetsWithFilter(ctx, func(a *immich.Asset) error {
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
	groupChan, err := app.browser.Browse(ctx)
	if err != nil {
		return err
	}
assetLoop:
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case g, ok := <-groupChan:
			if !ok {
				break assetLoop
			}
			err = app.handleGroup(ctx, g)
			if err != nil {
				return err
			}
		}
	}

	// if app.StackBurstPhotos || app.StackJpgWithRaw {
	// 	stacks := app.stacks.Stacks()
	// 	if len(stacks) > 0 {
	// 		app.Root.Log.Info("Creating stacks")
	// 	nextStack:
	// 		for _, s := range stacks {
	// 			switch {
	// 			case !app.StackBurstPhotos && s.StackType == stacking.StackBurst:
	// 				continue nextStack
	// 			case !app.StackJpgWithRaw && s.StackType == stacking.StackRawJpg:
	// 				continue nextStack
	// 			}
	// 			app.Root.Message(fmt.Sprintf("Stacking %s...", strings.Join(s.Names, ", ")))
	// 			err = app.Server.Immich.StackAssets(ctx, s.CoverID, s.IDs)
	// 			if err != nil {
	// 				app.Root.Log.Error(fmt.Sprintf("Can't stack images: %s", err))
	// 			}
	// 		}
	// 	}
	// }

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

	// if len(app.deleteLocalList) > 0 {
	// 	err = app.DeleteLocalAssets()
	// }

	return err
}

func (app *UpCmd) handleGroup(ctx context.Context, g *adapters.AssetGroup) error {
	var errGroup error

	// Upload assets from the group
	for _, a := range g.Assets {
		err := app.handleAsset(ctx, g, a)
		errGroup = errors.Join(err)
	}
	if errGroup != nil {
		return errGroup
	}

	switch g.Kind {
	case adapters.GroupKindMotionPhoto:
		var imageAsset *adapters.LocalAssetFile
		var videoAsset *adapters.LocalAssetFile
		for _, a := range g.Assets {
			switch app.Server.Immich.SupportedMedia().TypeFromExt(path.Ext(a.FileName)) {
			case metadata.TypeImage:
				imageAsset = a
			case metadata.TypeVideo:
				videoAsset = a
			}
		}
		app.Jnl.Record(ctx, fileevent.LivePhoto, fileevent.AsFileAndName(imageAsset.FSys, imageAsset.FileName), "video", videoAsset.FileName)
		_, err := app.Server.Immich.UpdateAsset(ctx, imageAsset.ID, immich.UpdAssetField{LivePhotoVideoID: videoAsset.ID})
		if err != nil {
			app.Jnl.Record(ctx, fileevent.Error, fileevent.FileAndName{}, "error", err.Error())
		}
	}

	// Manage albums
	if len(g.Albums) > 0 {
		app.manageGroupAlbums(ctx, g)
	}
	return nil
}

func (app *UpCmd) handleAsset(ctx context.Context, g *adapters.AssetGroup, a *adapters.LocalAssetFile) error {
	defer func() {
		a.Close() // Close and clean resources linked to the local asset
	}()

	advice, err := app.AssetIndex.ShouldUpload(a)
	if err != nil {
		return err
	}

	switch advice.Advice {
	case NotOnServer: // Upload and manage albums
		err = app.uploadAsset(ctx, a)
		return err
	case SmallerOnServer: // Upload, manage albums and delete the server's asset
		app.Jnl.Record(ctx, fileevent.UploadUpgraded, fileevent.AsFileAndName(a.FSys, a.FileName), "reason", advice.Message)

		// Remember existing asset's albums, if any
		for _, al := range advice.ServerAsset.Albums {
			g.AddAlbum(adapters.LocalAlbum{
				Title:       al.AlbumName,
				Description: al.Description,
			})
		}

		// Upload the superior asset
		err = app.uploadAsset(ctx, a)
		if err != nil {
			return err
		}

		// delete the existing lower quality asset
		err = app.Server.Immich.DeleteAssets(ctx, []string{advice.ServerAsset.ID}, true)
		if err != nil {
			app.Jnl.Record(ctx, fileevent.Error, fileevent.FileAndName{}, "error", err.Error())
		}
		return err

	case SameOnServer:
		a.ID = advice.ServerAsset.ID
		for _, al := range advice.ServerAsset.Albums {
			g.AddAlbum(adapters.LocalAlbum{
				Title:       al.AlbumName,
				Description: al.Description,
			})
		}
		app.Jnl.Record(ctx, fileevent.UploadServerDuplicate, fileevent.AsFileAndName(a.FSys, a.FileName), "reason", advice.Message)

	case BetterOnServer: // and manage albums
		app.Jnl.Record(ctx, fileevent.UploadServerBetter, fileevent.AsFileAndName(a.FSys, a.FileName), "reason", advice.Message)
	}

	return nil
}

func (app *UpCmd) uploadAsset(ctx context.Context, a *adapters.LocalAssetFile) error {
	ar, err := app.Server.Immich.AssetUpload(ctx, a)
	if err != nil {
		app.Jnl.Record(ctx, fileevent.UploadServerError, fileevent.AsFileAndName(a.FSys, a.FileName), "error", err.Error())
		return err // Must signal the error to the caller
	}
	if ar.Status == immich.UploadDuplicate {
		app.Jnl.Record(ctx, fileevent.UploadServerDuplicate, fileevent.AsFileAndName(a.FSys, a.FileName), "info", "the server has this file")
	} else {
		app.Jnl.Record(ctx, fileevent.Uploaded, fileevent.AsFileAndName(a.FSys, a.FileName))
	}
	a.ID = ar.ID
	return nil
}

// manageGroupAlbums add the assets to the albums listed in the group.
// If an album does not exist, it is created.
// Errors are logged.
func (app *UpCmd) manageGroupAlbums(ctx context.Context, g *adapters.AssetGroup) {
	assetIDs := []string{}
	for _, a := range g.Assets {
		assetIDs = append(assetIDs, a.ID)
	}

	for _, album := range g.Albums {
		title := album.Title
		l, exist := app.albums[title]
		if !exist {
			newAl, err := app.Server.Immich.CreateAlbum(ctx, title, album.Description, assetIDs)
			if err != nil {
				app.Jnl.Record(ctx, fileevent.Error, fileevent.FileAndName{}, err)
				return
			}
			app.albums[title] = &newAl
			l = &newAl
		} else {
			_, err := app.Server.Immich.AddAssetToAlbum(ctx, l.ID, assetIDs)
			if err != nil {
				app.Jnl.Record(ctx, fileevent.Error, fileevent.FileAndName{}, err)
				return
			}
		}

		// Log the action
		for _, a := range g.Assets {
			app.Jnl.Record(ctx, fileevent.UploadAddToAlbum, fileevent.AsFileAndName(a.FSys, a.FileName), "Album", title)
		}
	}
}

func (app *UpCmd) DeleteServerAssets(ctx context.Context, ids []string) error {
	app.Root.Message(fmt.Sprintf("%d server assets to delete.", len(ids)))
	return app.Server.Immich.DeleteAssets(ctx, ids, false)
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
