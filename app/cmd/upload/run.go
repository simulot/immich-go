package upload

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/simulot/immich-go/adapters"
	gp "github.com/simulot/immich-go/adapters/googlePhotos"
	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/assets/cache"
	cliflags "github.com/simulot/immich-go/internal/cliFlags"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/filters"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/simulot/immich-go/internal/gen/syncset"
)

type UpCmd struct {
	Mode UpLoadMode
	*UploadOptions
	app *app.Application

	assetIndex        *immichIndex         // List of assets present on the server
	localAssets       *syncset.Set[string] // List of assets present on the local input by name+size
	immichAssetsReady chan struct{}        // Signal that the asset index is ready
	deleteServerList  []*immich.Asset      // List of server assets to remove

	adapter       adapters.Reader
	DebugCounters bool // Enable CSV action counters per file

	Paths          []string // Path to explore
	takeoutOptions *gp.ImportFlags

	albumsCache *cache.CollectionCache[assets.Album] // List of albums present on the server
	tagsCache   *cache.CollectionCache[assets.Tag]   // List of tags present on the server

	shouldResumeJobs map[string]bool // List of jobs to resume
	finished         bool            // the finish task has been run
}

func newUpload(mode UpLoadMode, app *app.Application, options *UploadOptions) *UpCmd {
	upCmd := &UpCmd{
		UploadOptions:     options,
		app:               app,
		Mode:              mode,
		localAssets:       syncset.New[string](),
		immichAssetsReady: make(chan struct{}),
	}

	return upCmd
}

func (upCmd *UpCmd) setTakeoutOptions(options *gp.ImportFlags) *UpCmd {
	upCmd.takeoutOptions = options
	return upCmd
}

func (upCmd *UpCmd) saveAlbum(ctx context.Context, album assets.Album, ids []string) (assets.Album, error) {
	if len(ids) == 0 {
		return album, nil
	}
	if album.ID == "" {
		r, err := upCmd.app.Client().Immich.CreateAlbum(ctx, album.Title, album.Description, ids)
		if err != nil {
			upCmd.app.Jnl().Log().Error("failed to create album", "err", err, "album", album.Title)
			return album, err
		}
		upCmd.app.Jnl().Log().Info("created album", "album", album.Title, "assets", len(ids))
		album.ID = r.ID
		return album, nil
	}
	_, err := upCmd.app.Client().Immich.AddAssetToAlbum(ctx, album.ID, ids)
	if err != nil {
		upCmd.app.Jnl().Log().Error("failed to add assets to album", "err", err, "album", album.Title, "assets", len(ids))
		return album, err
	}
	upCmd.app.Jnl().Log().Info("updated album", "album", album.Title, "assets", len(ids))
	return album, err
}

func (upCmd *UpCmd) saveTags(ctx context.Context, tag assets.Tag, ids []string) (assets.Tag, error) {
	if len(ids) == 0 {
		return tag, nil
	}
	if tag.ID == "" {
		r, err := upCmd.app.Client().Immich.UpsertTags(ctx, []string{tag.Value})
		if err != nil {
			upCmd.app.Jnl().Log().Error("failed to create tag", "err", err, "tag", tag.Name)
			return tag, err
		}
		upCmd.app.Jnl().Log().Info("created tag", "tag", tag.Value)
		tag.ID = r[0].ID
	}
	_, err := upCmd.app.Client().Immich.TagAssets(ctx, tag.ID, ids)
	if err != nil {
		upCmd.app.Jnl().Log().Error("failed to add assets to tag", "err", err, "tag", tag.Value, "assets", len(ids))
		return tag, err
	}
	upCmd.app.Jnl().Log().Info("updated tag", "tag", tag.Value, "assets", len(ids))
	return tag, err
}

func (UpCmd *UpCmd) pauseJobs(ctx context.Context, app *app.Application) error {
	UpCmd.shouldResumeJobs = make(map[string]bool)
	jobs, err := app.Client().AdminImmich.GetJobs(ctx)
	if err != nil {
		return err
	}
	for name, job := range jobs {
		UpCmd.shouldResumeJobs[name] = !job.QueueStatus.IsPaused
		if UpCmd.shouldResumeJobs[name] {
			_, err = app.Client().AdminImmich.SendJobCommand(ctx, name, "pause", true)
			if err != nil {
				UpCmd.app.Jnl().Log().Error("Immich Job command sent", "pause", name, "err", err.Error())
				return err
			}
			UpCmd.app.Jnl().Log().Info("Immich Job command sent", "pause", name)
		}
	}
	return nil
}

func (UpCmd *UpCmd) resumeJobs(_ context.Context, app *app.Application) error {
	if UpCmd.shouldResumeJobs == nil {
		return nil
	}
	// Start with a context not yet cancelled
	ctx := context.Background() //nolint
	for name, shouldResume := range UpCmd.shouldResumeJobs {
		if shouldResume {
			_, err := app.Client().AdminImmich.SendJobCommand(ctx, name, "resume", true) //nolint:contextcheck
			if err != nil {
				UpCmd.app.Jnl().Log().Error("Immich Job command sent", "resume", name, "err", err.Error())
				return err
			}
			UpCmd.app.Jnl().Log().Info("Immich Job command sent", "resume", name)
		}
	}
	return nil
}

func (UpCmd *UpCmd) finishing(ctx context.Context, app *app.Application) error {
	if UpCmd.finished {
		return nil
	}
	defer func() { UpCmd.finished = true }()
	// do waiting operations
	UpCmd.albumsCache.Close()
	UpCmd.tagsCache.Close()

	// Resume immich background jobs if requested
	err := UpCmd.resumeJobs(ctx, app)
	if err != nil {
		return err
	}
	// Log the journal report
	report := app.Jnl().Report()

	if len(report) > 0 {
		lines := strings.Split(report, "\n")
		for _, s := range lines {
			app.Jnl().Log().Info(s)
		}
	}

	return nil
}

func (upCmd *UpCmd) run(ctx context.Context, adapter adapters.Reader, app *app.Application, fsys []fs.FS) error {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	// Stop immich background jobs if requested
	// will be resumed with a call to finishing()
	if app.Client().PauseImmichBackgroundJobs {
		err := upCmd.pauseJobs(ctx, app)
		if err != nil {
			return fmt.Errorf("can't pause immich background jobs: pass an administrator key with the flag --admin-api-key or disable the jobs pausing with the flag --pause-immich-jobs=FALSE\n%w", err)
		}
	}
	defer func() { _ = upCmd.finishing(ctx, app) }()
	defer func() {
		fmt.Println(app.Jnl().Report())
	}()
	upCmd.albumsCache = cache.NewCollectionCache(50, func(album assets.Album, ids []string) (assets.Album, error) {
		return upCmd.saveAlbum(ctx, album, ids)
	})
	upCmd.tagsCache = cache.NewCollectionCache(50, func(tag assets.Tag, ids []string) (assets.Tag, error) {
		return upCmd.saveTags(ctx, tag, ids)
	})

	upCmd.adapter = adapter

	runner := upCmd.runUI
	upCmd.assetIndex = newAssetIndex()

	if upCmd.NoUI {
		runner = upCmd.runNoUI
	}
	_, err := tcell.NewScreen()
	if err != nil {
		upCmd.app.Log().Warn("can't initialize the screen for the UI mode. Falling back to no-gui mode", "err", err)
		fmt.Println("can't initialize the screen for the UI mode. Falling back to no-gui mode")
		runner = upCmd.runNoUI
	}
	err = runner(ctx, app)

	err = errors.Join(err, fshelper.CloseFSs(fsys))

	return err
}

func (upCmd *UpCmd) getImmichAlbums(ctx context.Context) error {
	// Get the album list from the server, but without assets.
	serverAlbums, err := upCmd.app.Client().Immich.GetAllAlbums(ctx)
	if err != nil {
		return fmt.Errorf("can't get the album list from the server: %w", err)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-upCmd.immichAssetsReady:
		// Wait for the server's assets to be ready.
		for _, a := range serverAlbums {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				// Get the album info from the server, with assets.
				r, err := upCmd.app.Client().Immich.GetAlbumInfo(ctx, a.ID, false)
				if err != nil {
					upCmd.app.Log().Error("can't get the album info from the server", "album", a.AlbumName, "err", err)
					continue
				}
				ids := make([]string, 0, len(r.Assets))
				for _, aa := range r.Assets {
					ids = append(ids, aa.ID)
				}

				album := assets.NewAlbum(a.ID, a.AlbumName, a.Description)
				upCmd.albumsCache.NewCollection(a.AlbumName, album, ids)
				upCmd.app.Log().Info("got album from the server", "album", a.AlbumName, "assets", len(r.Assets))
				upCmd.app.Log().Debug("got album from the server", "album", a.AlbumName, "assets", ids)
				// assign the album to the assets
				for _, id := range ids {
					a := upCmd.assetIndex.getByID(id)
					if a == nil {
						upCmd.app.Log().Debug("processing the immich albums: asset not found in index", "id", id)
						continue
					}
					a.Albums = append(a.Albums, album)
				}
			}
		}
	}
	return nil
}

func (upCmd *UpCmd) getImmichAssets(ctx context.Context, updateFn progressUpdate) error {
	defer close(upCmd.immichAssetsReady)
	statistics, err := upCmd.app.Client().Immich.GetAssetStatistics(ctx)
	if err != nil {
		return err
	}
	totalOnImmich := statistics.Total
	received := 0

	err = upCmd.app.Client().Immich.GetAllAssetsWithFilter(ctx, nil, func(a *immich.Asset) error {
		if updateFn != nil {
			defer func() {
				updateFn(received, totalOnImmich)
			}()
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			received++
			if a.OwnerID != upCmd.app.Client().User.ID {
				upCmd.app.Log().Debug("Skipping asset with different owner", "assetOwnerID", a.OwnerID, "clientUserID", upCmd.app.Client().User.ID, "ID", a.ID, "FileName", a.OriginalFileName, "Capture date", a.ExifInfo.DateTimeOriginal, "CheckSum", a.Checksum, "FileSize", a.ExifInfo.FileSizeInByte, "DeviceAssetID", a.DeviceAssetID, "OwnerID", a.OwnerID, "IsTrashed", a.IsTrashed, "IsArchived", a.IsArchived)
				return nil
			}
			if a.LibraryID != "" {
				upCmd.app.Log().Debug("Skipping asset with external library", "assetLibraryID", a.LibraryID, "ID", a.ID, "FileName", a.OriginalFileName, "Capture date", a.ExifInfo.DateTimeOriginal, "CheckSum", a.Checksum, "FileSize", a.ExifInfo.FileSizeInByte, "DeviceAssetID", a.DeviceAssetID, "OwnerID", a.OwnerID, "IsTrashed", a.IsTrashed, "IsArchived", a.IsArchived)
				return nil
			}
			upCmd.assetIndex.addImmichAsset(a)
			upCmd.app.Log().Debug("Immich asset:", "ID", a.ID, "FileName", a.OriginalFileName, "Capture date", a.ExifInfo.DateTimeOriginal, "CheckSum", a.Checksum, "FileSize", a.ExifInfo.FileSizeInByte, "DeviceAssetID", a.DeviceAssetID, "OwnerID", a.OwnerID, "IsTrashed", a.IsTrashed, "IsArchived", a.IsArchived)
			return nil
		}
	})
	if err != nil {
		return err
	}
	if updateFn != nil {
		updateFn(totalOnImmich, totalOnImmich)
	}
	upCmd.app.Log().Info(fmt.Sprintf("Assets on the server: %d", upCmd.assetIndex.len()))
	return nil
}

func (upCmd *UpCmd) uploadLoop(ctx context.Context, groupChan chan *assets.Group) error {
	var err error
	errorCount := 0
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
				upCmd.app.Log().Error(err.Error())

				switch {
				case upCmd.app.Client().OnServerErrors == cliflags.OnServerErrorsNeverStop:
					// nop
				case upCmd.app.Client().OnServerErrors == cliflags.OnServerErrorsStop:
					return err
				default:
					errorCount++
					if errorCount >= int(upCmd.app.Client().OnServerErrors) {
						err := errors.New("too many errors, aborting")
						upCmd.app.Log().Error(err.Error())
						return err
					}
				}
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

	g = filters.ApplyFilters(g, upCmd.Filters...)

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

	// var status stri g
	advice, err := upCmd.assetIndex.ShouldUpload(a, upCmd)
	if err != nil {
		return err
	}

	switch advice.Advice {
	case NotOnServer: // Upload and manage albums
		serverStatus, err := upCmd.uploadAsset(ctx, a)
		if err != nil {
			return err
		}

		upCmd.processUploadedAsset(ctx, a, serverStatus)
		return nil

	case SmallerOnServer: // Upload, manage albums and delete the server's asset

		// Remember existing asset's albums, if any
		a.Albums = append(a.Albums, advice.ServerAsset.Albums...)

		// Upload the superior asset
		serverStatus, err := upCmd.replaceAsset(ctx, advice.ServerAsset.ID, a, advice.ServerAsset)
		if err != nil {
			return err
		}

		upCmd.processUploadedAsset(ctx, a, serverStatus)
		return nil

	case AlreadyProcessed: // SHA1 already processed
		upCmd.app.Jnl().Record(ctx, fileevent.AnalysisLocalDuplicate, a.File, "reason", "the file is already present in the input", "original name", advice.ServerAsset.OriginalFileName)
		return nil

	case SameOnServer:
		a.ID = advice.ServerAsset.ID
		a.Albums = append(a.Albums, advice.ServerAsset.Albums...)
		upCmd.app.Jnl().Record(ctx, fileevent.UploadServerDuplicate, a.File, "reason", advice.Message)
		upCmd.manageAssetAlbums(ctx, a.File, a.ID, a.Albums)

	case BetterOnServer: // and manage albums
		a.ID = advice.ServerAsset.ID
		upCmd.app.Jnl().Record(ctx, fileevent.UploadServerBetter, a.File, "reason", advice.Message)
		upCmd.manageAssetAlbums(ctx, a.File, a.ID, a.Albums)

	case ForceUpload:
		var serverStatus string
		var err error

		if advice.ServerAsset != nil {
			// Remember existing asset's albums, if any
			a.Albums = append(a.Albums, advice.ServerAsset.Albums...)

			// Upload the superior asset
			serverStatus, err = upCmd.replaceAsset(ctx, advice.ServerAsset.ID, a, advice.ServerAsset)
		} else {
			serverStatus, err = upCmd.uploadAsset(ctx, a)
		}
		if err != nil {
			return err
		}

		upCmd.processUploadedAsset(ctx, a, serverStatus)
		return nil
	}

	return nil
}

// uploadAsset uploads the asset to the server.
// set the server's asset ID to the asset.
// return the duplicate condition and error.
func (upCmd *UpCmd) uploadAsset(ctx context.Context, a *assets.Asset) (string, error) {
	defer upCmd.app.Log().Debug("", "file", a)
	ar, err := upCmd.app.Client().Immich.AssetUpload(ctx, a)
	if err != nil {
		upCmd.app.Jnl().Record(ctx, fileevent.UploadServerError, a.File, "error", err.Error())
		return "", err // Must signal the error to the caller
	}
	if ar.Status == immich.UploadDuplicate {
		originalName := "unknown"
		original := upCmd.assetIndex.getByID(ar.ID)
		if original != nil {
			originalName = original.OriginalFileName
		}
		if a.ID == "" {
			upCmd.app.Jnl().Record(ctx, fileevent.AnalysisLocalDuplicate, a.File, "reason", "the file is already present in the input", "original name", originalName)
		} else {
			upCmd.app.Jnl().Record(ctx, fileevent.UploadServerDuplicate, a.File, "reason", "the server already has this file", "original name", originalName)
		}
	} else {
		upCmd.app.Jnl().Record(ctx, fileevent.Uploaded, a.File)
	}
	a.ID = ar.ID

	// // DEBGUG
	//  if theID, ok := upCmd.assetIndex.byI

	if a.FromApplication != nil && ar.Status != immich.StatusDuplicate {
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
			return "", err
		}
	}
	upCmd.assetIndex.addLocalAsset(a)
	return ar.Status, nil
}

func (upCmd *UpCmd) replaceAsset(ctx context.Context, ID string, a, old *assets.Asset) (string, error) {
	defer upCmd.app.Log().Debug("replaced by", "ID", ID, "file", a)
	ar, err := upCmd.app.Client().Immich.ReplaceAsset(ctx, ID, a)
	if err != nil {
		upCmd.app.Jnl().Record(ctx, fileevent.UploadServerError, a.File, "error", err.Error())
		return "", err // Must signal the error to the caller
	}
	if ar.Status == immich.UploadDuplicate {
		originalName := "unknown"
		original := upCmd.assetIndex.getByID(ar.ID)
		if original != nil {
			originalName = original.OriginalFileName
		}
		if a.ID == "" {
			upCmd.app.Jnl().Record(ctx, fileevent.AnalysisLocalDuplicate, a.File, "reason", "the file is already present in the input", "original name", originalName)
		} else {
			upCmd.app.Jnl().Record(ctx, fileevent.UploadServerDuplicate, a.File, "reason", "the server already has this file", "original name", originalName)
		}
	} else {
		a.ID = ID
		upCmd.app.Jnl().Record(ctx, fileevent.UploadUpgraded, a.File)
		upCmd.assetIndex.replaceAsset(a, old)
	}
	return ar.Status, nil
}

// manageAssetAlbums add the assets to the albums listed.
// If an album does not exist, it is created.
// If the album already has the asset, it is not added.
// Errors are logged.
func (upCmd *UpCmd) manageAssetAlbums(ctx context.Context, f fshelper.FSAndName, ID string, albums []assets.Album) {
	if len(albums) == 0 {
		return
	}

	for _, album := range albums {
		al := assets.NewAlbum("", album.Title, album.Description)
		if upCmd.albumsCache.AddIDToCollection(al.Title, album, ID) {
			upCmd.app.Jnl().Record(ctx, fileevent.UploadAddToAlbum, f, "album", al.Title)
		}
	}
}

func (upCmd *UpCmd) manageAssetTags(ctx context.Context, a *assets.Asset) {
	if len(a.Tags) == 0 {
		return
	}

	tags := make([]string, len(a.Tags))
	for i := range a.Tags {
		tags[i] = a.Tags[i].Name
	}
	for _, t := range a.Tags {
		if upCmd.tagsCache.AddIDToCollection(t.Name, t, a.ID) {
			upCmd.app.Jnl().Record(ctx, fileevent.Tagged, a.File, "tag", t.Value)
		}
	}
}

func (upCmd *UpCmd) DeleteServerAssets(ctx context.Context, ids []string) error {
	upCmd.app.Log().Message("%d server assets to delete.", len(ids))
	return upCmd.app.Client().Immich.DeleteAssets(ctx, ids, false)
}

func (upCmd *UpCmd) processUploadedAsset(ctx context.Context, a *assets.Asset, serverStatus string) {
	if serverStatus != immich.StatusDuplicate {
		// TODO: current version of Immich doesn't allow to add same tag to an asset already tagged.
		//       there is no mean to go the list of tagged assets for a given tag.
		upCmd.manageAssetAlbums(ctx, a.File, a.ID, a.Albums)
		upCmd.manageAssetTags(ctx, a)
	}
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
