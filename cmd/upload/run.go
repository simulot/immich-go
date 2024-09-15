package upload

import (
	"context"
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/simulot/immich-go/adapters"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/stacking"
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
	assetChan, err := app.browser.Browse(ctx)
	if err != nil {
		return err
	}
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

	if app.StackBurstPhotos || app.StackJpgWithRaw {
		stacks := app.stacks.Stacks()
		if len(stacks) > 0 {
			app.Root.Log.Info("Creating stacks")
		nextStack:
			for _, s := range stacks {
				switch {
				case !app.StackBurstPhotos && s.StackType == stacking.StackBurst:
					continue nextStack
				case !app.StackJpgWithRaw && s.StackType == stacking.StackRawJpg:
					continue nextStack
				}
				app.Root.Message(fmt.Sprintf("Stacking %s...", strings.Join(s.Names, ", ")))
				err = app.Server.Immich.StackAssets(ctx, s.CoverID, s.IDs)
				if err != nil {
					app.Root.Log.Error(fmt.Sprintf("Can't stack images: %s", err))
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

	// if len(app.deleteLocalList) > 0 {
	// 	err = app.DeleteLocalAssets()
	// }

	return err
}

func (app *UpCmd) handleAsset(ctx context.Context, a *adapters.LocalAssetFile) error {
	defer func() {
		a.Close() // Close and clean resources linked to the local asset
	}()

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
		app.manageAssetAlbum(ctx, ID, a)

	case SmallerOnServer: // Upload, manage albums and delete the server's asset
		app.Jnl.Record(ctx, fileevent.UploadUpgraded, a, a.FileName, "reason", advice.Message)

		// Get existing asset's albums, if any
		for _, al := range advice.ServerAsset.Albums {
			a.Albums = append(a.Albums, adapters.LocalAlbum{
				Title:       al.AlbumName,
				Description: al.Description,
			})
		}

		// Upload the superior asset
		ID, err := app.UploadAsset(ctx, a)
		if err != nil {
			return nil
		}
		app.manageAssetAlbum(ctx, ID, a)

		// delete the existing lower quality asset
		err = app.Server.Immich.DeleteAssets(ctx, []string{ID}, true)
		if err != nil {
			app.Jnl.Record(ctx, fileevent.Error, a, a.FileName, "error", err.Error())
		}

	case SameOnServer: // manage albums
		// Set add the server asset into albums determined locally
		if !advice.ServerAsset.JustUploaded {
			app.Jnl.Record(ctx, fileevent.UploadServerDuplicate, a, a.FileName, "reason", advice.Message)
		} else {
			app.Jnl.Record(ctx, fileevent.AnalysisLocalDuplicate, a, a.FileName)
		}
		app.manageAssetAlbum(ctx, advice.ServerAsset.ID, a)

	case BetterOnServer: // and manage albums
		app.Jnl.Record(ctx, fileevent.UploadServerBetter, a, a.FileName, "reason", advice.Message)
		app.manageAssetAlbum(ctx, advice.ServerAsset.ID, a)
	}

	return nil
}

// manageAssetAlbum adds the asset to all albums it should be in.
// If an album does not exist, it is created.
// Errors are logged.
func (app *UpCmd) manageAssetAlbum(ctx context.Context, id string, a *adapters.LocalAssetFile) {
	addedTo := map[string]bool{}

	for _, album := range a.Albums {
		if addedTo[album.Title] {
			continue
		}
		title := album.Title
		l, exist := app.albums[title]
		if !exist {
			newAl, err := app.Server.Immich.CreateAlbum(ctx, title, album.Description, []string{id})
			if err != nil {
				app.Jnl.Record(ctx, fileevent.Error, a, a.FileName, err)
				continue
			}
			app.Jnl.Record(ctx, fileevent.UploadAddToAlbum, a, a.FileName, "Album", newAl.AlbumName)
			app.albums[title] = &newAl
			l = &newAl
		} else {
			_, err := app.Server.Immich.AddAssetToAlbum(ctx, l.ID, []string{id})
			if err != nil {
				app.Jnl.Record(ctx, fileevent.Error, a, a.FileName, err)
				continue
			}
			app.Jnl.Record(ctx, fileevent.UploadAddToAlbum, a, a.FileName, "Album", l.AlbumName)
		}
		addedTo[l.AlbumName] = true
	}
}

// func (app *UpCmd) isInAlbum(a *adapters.LocalAssetFile, album string) bool {
// 	for _, al := range a.Albums {
// 		if app.albumName(al) == album {
// 			return true
// 		}
// 	}
// 	return false
// }

// UploadAsset upload the asset on the server
// Add the assets into listed albums
// return ID of the asset
func (app *UpCmd) UploadAsset(ctx context.Context, a *adapters.LocalAssetFile) (string, error) {
	var resp, liveResp immich.AssetResponse
	var err error

	if a.LivePhoto != nil {
		liveResp, err = app.Server.Immich.AssetUpload(ctx, a.LivePhoto)
		if err == nil {
			if liveResp.Status == immich.UploadDuplicate {
				app.Jnl.Record(ctx, fileevent.UploadServerDuplicate, a.LivePhoto, a.LivePhoto.FileName, "info", "the server has this file")
			} else {
				app.Jnl.Record(ctx, fileevent.Uploaded, a.LivePhoto, a.LivePhoto.FileName)
			}
			a.LivePhotoID = liveResp.ID
		} else {
			app.Jnl.Record(ctx, fileevent.UploadServerError, a.LivePhoto, a.LivePhoto.FileName, "error", err.Error())
		}
	}
	b := *a // Keep a copy of the asset to log errors specifically on the image
	resp, err = app.Server.Immich.AssetUpload(ctx, a)
	if err == nil {
		if resp.Status == immich.UploadDuplicate {
			app.Jnl.Record(ctx, fileevent.UploadServerDuplicate, a, a.FileName, "info", "the server has this file")
		} else {
			b.LivePhoto = nil
			app.Jnl.Record(ctx, fileevent.Uploaded, &b, b.FileName, "capture date", b.Metadata.DateTaken.String())
		}
	} else {
		app.Jnl.Record(ctx, fileevent.UploadServerError, a, a.FileName, "error", err.Error())
		return "", err
	}

	return resp.ID, nil
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

func (app *UpCmd) DeleteServerAssets(ctx context.Context, ids []string) error {
	app.Root.Message(fmt.Sprintf("%d server assets to delete.", len(ids)))
	return app.Server.Immich.DeleteAssets(ctx, ids, false)
}
