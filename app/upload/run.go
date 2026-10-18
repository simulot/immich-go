package upload

import (
	"context"
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/simulot/immich-go/adapters"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/assets/cache"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/filters"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/simulot/immich-go/internal/worker"
)

func (uc *UpCmd) saveAlbum(ctx context.Context, album assets.Album, ids []string) (assets.Album, error) {
	if len(ids) == 0 {
		return album, nil
	}
	if album.ID == "" {
		r, err := uc.client.Immich.CreateAlbum(ctx, album.Title, album.Description, ids)
		if err != nil {
			uc.app.Log().Error("failed to create album", "err", err, "album", album.Title)
			return album, err
		}
		uc.app.Log().Info("created album", "album", album.Title, "assets", len(ids))
		album.ID = r.ID
		return album, nil
	}
	_, err := uc.client.Immich.AddAssetToAlbum(ctx, album.ID, ids)
	if err != nil {
		uc.app.Log().Error("failed to add assets to album", "err", err, "album", album.Title, "assets", len(ids))
		return album, err
	}
	uc.app.Log().Info("updated album", "album", album.Title, "assets", len(ids))
	return album, err
}

func (uc *UpCmd) saveTags(ctx context.Context, tag assets.Tag, ids []string) (assets.Tag, error) {
	if len(ids) == 0 {
		return tag, nil
	}
	if tag.ID == "" {
		r, err := uc.client.Immich.UpsertTags(ctx, []string{tag.Value})
		if err != nil {
			uc.app.Log().Error("failed to create tag", "err", err, "tag", tag.Name)
			return tag, err
		}
		uc.app.Log().Info("created tag", "tag", tag.Value)
		tag.ID = r[0].ID
	}
	_, err := uc.client.Immich.TagAssets(ctx, tag.ID, ids)
	if err != nil {
		uc.app.Log().Error("failed to add assets to tag", "err", err, "tag", tag.Value, "assets", len(ids))
		return tag, err
	}
	uc.app.Log().Info("updated tag", "tag", tag.Value, "assets", len(ids))
	return tag, err
}

// backgroundJobs lists the Immich job queues paused during the upload.
var backgroundJobs = []string{"thumbnailGeneration", "metadataExtraction", "videoConversion", "faceDetection", "smartSearch"}

// pauseJobs pauses the Immich background jobs that aren't already paused, and records them in
// uc.pausedJobs so that resumeJobs resumes only those. Queues paused by the user before the run
// are left alone. If pausing a queue fails, the queues paused so far are resumed (best effort)
// before returning the error, since the upload is then aborted and finishing() never runs.
func (uc *UpCmd) pauseJobs(ctx context.Context) error {
	status, err := uc.client.AdminImmich.GetJobs(ctx)
	if err != nil {
		uc.app.Log().Error("Immich Job status", "err", err.Error())
		return err
	}
	for _, name := range backgroundJobs {
		if status[name].QueueStatus.IsPaused {
			uc.app.Log().Info("Immich Job already paused, leaving it as is", "job", name)
			continue
		}
		_, err := uc.client.AdminImmich.SendJobCommand(ctx, name, "pause", true)
		if err != nil {
			uc.app.Log().Error("Immich Job command sent", "pause", name, "err", err.Error())
			_ = uc.resumeJobs(ctx)
			return err
		}
		uc.pausedJobs = append(uc.pausedJobs, name)
		uc.app.Log().Info("Immich Job command sent", "pause", name)
	}
	return nil
}

// resumeJobs resumes the jobs paused by pauseJobs during this run. It does nothing when the
// jobs weren't paused (--pause-immich-jobs=false), or for queues that were already paused before.
func (uc *UpCmd) resumeJobs(_ context.Context) error {
	// Start with a context not yet cancelled
	ctx := context.Background() //nolint
	for _, name := range uc.pausedJobs {
		_, err := uc.client.AdminImmich.SendJobCommand(ctx, name, "resume", true) //nolint:contextcheck
		if err != nil {
			uc.app.Log().Error("Immich Job command sent", "resume", name, "err", err.Error())
			return err
		}
		uc.app.Log().Info("Immich Job command sent", "resume", name)
	}
	return nil
}

func (uc *UpCmd) finishing(ctx context.Context) error {
	if uc.finished {
		return nil
	}
	defer func() { uc.finished = true }()
	// do waiting operations
	uc.albumsCache.Close()
	uc.tagsCache.Close()

	// Resume the immich background jobs paused by this run, if any
	err := uc.resumeJobs(ctx)
	if err != nil {
		return err
	}

	// Generate FileProcessor report
	if uc.app.FileProcessor() != nil {
		report := uc.app.FileProcessor().GenerateReport()
		if len(report) > 0 {
			lines := strings.Split(report, "\n")
			for _, s := range lines {
				uc.app.Log().Info(s)
			}
		}
	}

	return nil
}

func (uc *UpCmd) upload(ctx context.Context, adapter adapters.Reader) error {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	// Stop immich background jobs if requested
	// will be resumed with a call to finishing()
	if uc.client.PauseImmichBackgroundJobs {
		err := uc.pauseJobs(ctx)
		if err != nil {
			return fmt.Errorf("can't pause immich background jobs: pass an administrator key with the flag --admin-api-key or disable the jobs pausing with the flag --pause-immich-jobs=FALSE\n%w", err)
		}
	}
	defer func() { _ = uc.finishing(ctx) }()
	defer func() {
		if uc.app.FileProcessor() != nil {
			fmt.Println(uc.app.FileProcessor().GenerateReport())
		}
	}()
	uc.albumsCache = cache.NewCollectionCache(50, func(album assets.Album, ids []string) (assets.Album, error) {
		return uc.saveAlbum(ctx, album, ids)
	})
	uc.tagsCache = cache.NewCollectionCache(50, func(tag assets.Tag, ids []string) (assets.Tag, error) {
		return uc.saveTags(ctx, tag, ids)
	})

	uc.adapter = adapter

	runner := uc.runUI
	uc.assetIndex = newAssetIndex()

	if uc.NoUI {
		runner = uc.runNoUI
	} else {
		_, err := tcell.NewScreen()
		if err != nil {
			uc.app.Log().Warn("can't initialize the screen for the UI mode. Falling back to no-gui mode", "err", err)
			fmt.Println("can't initialize the screen for the UI mode. Falling back to no-gui mode")
			runner = uc.runNoUI
		}
	}
	err := runner(ctx, uc.app)
	return err
}

func (uc *UpCmd) getImmichAlbums(ctx context.Context) error {
	// Get the album list from the server, but without assets.
	serverAlbums, err := uc.client.Immich.GetAllAlbums(ctx)
	if err != nil {
		return fmt.Errorf("can't get the album list from the server: %w", err)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-uc.immichAssetsReady:
		// Wait for the server's assets to be ready.
		for _, a := range serverAlbums {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				ids, err := uc.client.Immich.GetAlbumAssetIDs(ctx, a.ID)
				if err != nil {
					uc.app.Log().Error("can't get the album assets from the server", "album", a.AlbumName, "err", err)
					continue
				}

				album := assets.NewAlbum(a.ID, a.AlbumName, a.Description)
				uc.albumsCache.NewCollection(a.AlbumName, album, ids)
				uc.app.Log().Info("got album from the server", "album", a.AlbumName, "assets", len(ids))
				uc.app.Log().Debug("got album from the server", "album", a.AlbumName, "assets", ids)
				// assign the album to the assets
				for _, id := range ids {
					a := uc.assetIndex.getByID(id)
					if a == nil {
						uc.app.Log().Debug("processing the immich albums: asset not found in index", "id", id)
						continue
					}
					a.Albums = append(a.Albums, album)
				}
			}
		}
	}
	return nil
}

func (uc *UpCmd) getImmichAssets(ctx context.Context, updateFn progressUpdate) error {
	defer close(uc.immichAssetsReady)
	statistics, err := uc.client.Immich.GetAssetStatistics(ctx)
	if err != nil {
		return err
	}
	totalOnImmich := statistics.Total
	received := 0

	err = uc.client.Immich.GetAllAssets(ctx, func(a *immich.Asset) error {
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
			if a.OwnerID != uc.client.User.ID {
				uc.app.Log().Debug("Skipping asset with different owner", "assetOwnerID", a.OwnerID, "clientUserID", uc.client.User.ID, "ID", a.ID, "FileName", a.OriginalFileName, "Capture date", a.ExifInfo.DateTimeOriginal, "CheckSum", a.Checksum, "FileSize", a.ExifInfo.FileSizeInByte, "DeviceAssetID", a.DeviceAssetID, "OwnerID", a.OwnerID, "IsTrashed", a.IsTrashed, "IsArchived", a.IsArchived)
				return nil
			}
			if a.LibraryID != "" {
				uc.app.Log().Debug("Skipping asset with external library", "assetLibraryID", a.LibraryID, "ID", a.ID, "FileName", a.OriginalFileName, "Capture date", a.ExifInfo.DateTimeOriginal, "CheckSum", a.Checksum, "FileSize", a.ExifInfo.FileSizeInByte, "DeviceAssetID", a.DeviceAssetID, "OwnerID", a.OwnerID, "IsTrashed", a.IsTrashed, "IsArchived", a.IsArchived)
				return nil
			}
			uc.assetIndex.addImmichAsset(a)
			uc.app.Log().Debug("Immich asset:", "ID", a.ID, "FileName", a.OriginalFileName, "Capture date", a.ExifInfo.DateTimeOriginal, "CheckSum", a.Checksum, "FileSize", a.ExifInfo.FileSizeInByte, "DeviceAssetID", a.DeviceAssetID, "OwnerID", a.OwnerID, "IsTrashed", a.IsTrashed, "IsArchived", a.IsArchived)
			return nil
		}
	})
	if err != nil {
		return err
	}
	if updateFn != nil {
		updateFn(totalOnImmich, totalOnImmich)
	}
	uc.app.Log().Info(fmt.Sprintf("Assets on the server: %d", uc.assetIndex.len()))
	return nil
}

func (uc *UpCmd) uploadLoop(ctx context.Context, groupChan chan *assets.Group) error {
	ctx, cancel := context.WithCancelCause(ctx)

	// the goroutine submits the groups, and stops when then number of error is higher than tolerated
	var wg sync.WaitGroup
	wg.Go(func() {
		workers := worker.NewPool(uc.app.ConcurrentTask)
		defer workers.Stop()
		for {
			select {
			case <-ctx.Done():
				cancel(ctx.Err())
				return
			case g, ok := <-groupChan:
				if !ok {
					return
				}
				workers.Submit(func() {
					err := uc.handleGroup(ctx, g)
					if err != nil {
						err = uc.app.ProcessError(err)
						if err != nil {
							cancel(err)
						}
					}
				})
			}
		}
	})

	wg.Wait()
	err := context.Cause(ctx)

	// Cleanup: delete server assets if needed
	if len(uc.deleteServerList) > 0 {
		ids := make([]string, 0, len(uc.deleteServerList))
		for _, da := range uc.deleteServerList {
			ids = append(ids, da.ID)
		}
		err := uc.DeleteServerAssets(ctx, ids)
		if err != nil {
			return fmt.Errorf("can't delete server's assets: %w", err)
		}
	}

	return err
}

func (uc *UpCmd) handleGroup(ctx context.Context, g *assets.Group) error {
	var errGroup error

	g = filters.ApplyFilters(g, uc.Filters...)

	// discard rejected assets
	for _, a := range g.Removed {
		a.Asset.Close()
		// Record asset as discarded with reason
		uc.app.FileProcessor().RecordAssetDiscarded(ctx, a.Asset.File, int64(a.Asset.FileSize), fileevent.DiscardedNotSelected, a.Reason)
	}

	// A standalone edited file still needs a stack when the server holds the original.
	editedAlone := len(g.Assets) == 1 && g.Assets[0].Kind == assets.KindEdited

	// Classify the server assets that match an edit pair's files before uploading, so
	// ShouldUpload can skip versions and stackIDs can merge an existing stack.
	stackBlocked := false
	groupSet := make(map[*assets.Asset]bool, len(g.Assets))
	hasEdit := false
	for _, a := range g.Assets {
		groupSet[a] = true
		if a.Kind == assets.KindEdited {
			hasEdit = true
		}
	}
	if hasEdit {
		for _, a := range g.Assets {
			name := path.Base(a.File.Name())
			if a.Kind == assets.KindEdited {
				name = a.OriginalFileName
			}
			for _, sa := range uc.assetIndex.nameDateCandidates(a, name) {
				if groupSet[sa] {
					continue // uploaded in this run; cannot be in a stack yet
				}
				if uc.assetIndex.isAlreadyProcessed(sa.Checksum) {
					continue // uploaded by this run; its ID can be a dry-run UUID, it is not a server stack
				}
				if uc.assetIndex.serverVersions.Contains(sa.ID) {
					continue
				}
				if _, known := uc.assetIndex.stackPrimaryByID.Load(sa.ID); known {
					continue
				}
				info, err := uc.client.Immich.GetAssetInfo(ctx, sa.ID)
				if err != nil {
					uc.app.Log().Warn("Can't classify the server asset's version", "asset", sa.ID, "error", err)
					stackBlocked = true
					continue
				}
				uc.assetIndex.recordVersion(info)
			}
		}
	}

	// Upload assets from the group
	for _, a := range g.Assets {
		err := uc.handleAsset(ctx, a, g)
		errGroup = errors.Join(errGroup, err)
	}

	// Manage groups
	// after the filtering and the upload, we can stack the assets
	if (len(g.Assets) > 1 && g.Grouping != assets.GroupByNone) || editedAlone {
		client := uc.client.Immich.(immich.ImmichStackInterface)
		ids, blocked := stackIDs(g, uc.assetIndex)
		if blocked {
			for _, a := range g.Assets {
				if a.Kind == assets.KindEdited {
					uc.app.Log().Warn("Ambiguous server version for an edited asset; not stacking", "asset", a.File.Name())
				}
			}
			return errGroup
		}
		if stackBlocked {
			return errGroup // could not classify a version; do not risk splitting a stack
		}
		if len(ids) > 1 {
			for _, a := range g.Assets {
				if slices.Contains(ids, uc.assetIndex.replacement(a).ID) {
					// Record stacking event
					uc.app.FileProcessor().RecordNonAsset(ctx, a.File, 0, fileevent.ProcessedStacked)
				}
			}
			_, err := client.CreateStack(ctx, ids)
			if err != nil {
				uc.app.Log().Error("Can't create stack", "error", err)
			}
		}
	}
	return errGroup
}

// stackIDs returns the distinct, non-empty server IDs of the group's assets, cover first, each
// resolved through ii.replacement (an asset matched to a server asset that a later asset of the
// group replaced is represented by the replacement). An asset that was discarded, or that failed
// to upload, has no ID and is left out. The same server asset can back several assets of the
// group (a local duplicate), and must be listed once.
// The second result reports an ambiguity: several server copies match an edited asset, or they
// belong to different stacks. The caller must not stack then, because picking one would split an
// existing stack.
func stackIDs(g *assets.Group, ii *immichIndex) ([]string, bool) {
	ids := make([]string, 0, len(g.Assets)+1)
	addID := func(id string) {
		if id != "" && !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}
	add := func(a *assets.Asset) {
		addID(ii.replacement(a).ID)
	}
	if g.CoverIndex >= 0 && g.CoverIndex < len(g.Assets) {
		add(g.Assets[g.CoverIndex])
	}
	for _, a := range g.Assets {
		add(a)
	}
	blocked := false
	for _, a := range g.Assets {
		if a.Kind != assets.KindEdited || ii.replacement(a).ID == "" {
			continue // the edit has no server asset; nothing to stack it with
		}
		var baseID, primaryID string
		for _, sa := range ii.editBaseCandidates(a) {
			if slices.Contains(g.Assets, sa) {
				continue // the group's own original; already added
			}
			p, inStack := ii.stackPrimaryByID.Load(sa.ID)
			if inStack && p != "" {
				if primaryID == "" {
					primaryID = p
				} else if primaryID != p {
					blocked = true
					break
				}
			} else {
				if baseID == "" {
					baseID = sa.ID
				} else if baseID != sa.ID {
					blocked = true
					break
				}
			}
		}
		if blocked {
			break
		}
		if primaryID != "" {
			addID(primaryID) // the server merges the whole existing stack
		}
		if baseID != "" {
			addID(baseID)
		}
	}
	return ids, blocked
}

// handleAsset uploads the asset a, or updates the server's copy of it, as advised by the asset
// index. g is the group a belongs to; its other assets are never mistaken for server-side variants
// of a.
func (uc *UpCmd) handleAsset(ctx context.Context, a *assets.Asset, g *assets.Group) error {
	defer func() {
		a.Close() // Close and clean resources linked to the local asset
	}()

	var siblings []*assets.Asset
	if g != nil {
		siblings = g.Assets
	}
	advice, err := uc.assetIndex.ShouldUpload(a, uc, siblings...)
	if err != nil {
		return err
	}

	switch advice.Advice {
	case NotOnServer: // Upload and manage albums
		err = uc.uploadAsset(ctx, a)
		if err != nil {
			return err
		}

		uc.processUploadedAsset(ctx, a)
		return nil

	case SmallerOnServer: // Upload, manage albums and delete the server's asset

		// Remember existing asset's albums, if any
		a.Albums = append(a.Albums, advice.ServerAsset.Albums...)

		// Upload the superior asset
		err = uc.replaceAsset(ctx, a, advice.ServerAsset)
		if err != nil {
			return err
		}

		uc.processUploadedAsset(ctx, a)
		uc.app.FileProcessor().RecordAssetProcessed(ctx, a.File, int64(a.FileSize), fileevent.ProcessedUploadUpgraded)

		return nil

	case AlreadyProcessed: // SHA1 already processed
		a.ID = advice.ServerAsset.ID
		// Record as discarded - duplicate in input
		uc.app.FileProcessor().RecordNonAsset(ctx, a.File, int64(a.FileSize), fileevent.DiscardedLocalDuplicate)
		uc.app.FileProcessor().RecordAssetProcessed(ctx, a.File, int64(a.FileSize), fileevent.ProcessedMetadataUpdated)
		uc.manageAssetAlbums(ctx, a.File, a.ID, a.Albums)
		uc.manageAssetTags(ctx, a)
		return nil

	case SameOnServer:
		a.ID = advice.ServerAsset.ID
		a.Albums = append(a.Albums, advice.ServerAsset.Albums...)
		// Record as processed - duplicate on server
		uc.app.FileProcessor().RecordNonAsset(ctx, a.File, int64(a.FileSize), fileevent.DiscardedServerDuplicate)
		uc.app.FileProcessor().RecordAssetProcessed(ctx, a.File, int64(a.FileSize), fileevent.ProcessedMetadataUpdated)
		uc.manageAssetAlbums(ctx, a.File, a.ID, a.Albums)
		uc.manageAssetTags(ctx, a)

	case BetterOnServer: // and manage albums
		a.ID = advice.ServerAsset.ID
		// Record as discarded - server has better version
		uc.app.FileProcessor().RecordAssetDiscarded(ctx, a.File, int64(a.FileSize), fileevent.ProcessedMetadataUpdated, advice.Message)
		uc.manageAssetAlbums(ctx, a.File, a.ID, a.Albums)
		uc.manageAssetTags(ctx, a)
	case ForceUpload:
		if advice.ServerAsset != nil {
			// Remember existing asset's albums, if any
			a.Albums = append(a.Albums, advice.ServerAsset.Albums...)

			// Upload the superior asset
			err = uc.replaceAsset(ctx, a, advice.ServerAsset)
		} else {
			err = uc.uploadAsset(ctx, a)
		}
		if err != nil {
			return err
		}

		uc.processUploadedAsset(ctx, a)
		return nil
	}

	return nil
}

// addCommandLineTags adds the --tag values and, if --session-tag is set, the session tag to the
// asset, so that manageAssetTags applies them on the server.
func (uc *UpCmd) addCommandLineTags(a *assets.Asset) {
	if uc.SessionTag {
		a.AddTag(uc.session)
	}
	for _, tag := range uc.Tags {
		a.AddTag(tag)
	}
}

// uploadAsset uploads the asset to the server.
// set the server's asset ID to the asset.
func (uc *UpCmd) uploadAsset(ctx context.Context, a *assets.Asset) error {
	defer uc.app.Log().Debug("upload asset", "file", a)

	ar, err := uc.client.Immich.AssetUpload(ctx, a)
	if err != nil {
		// Record upload error
		uc.app.FileProcessor().RecordAssetError(ctx, a.File, int64(a.FileSize), fileevent.ErrorServerError, err)
		return err // Must signal the error to the caller
	}
	if ar.Status == immich.UploadDuplicate {
		originalName := "unknown"
		original := uc.assetIndex.getByID(ar.ID)
		if original != nil {
			originalName = original.OriginalFileName
		}
		if a.ID == "" {
			// Record as discarded - local duplicate
			uc.app.FileProcessor().RecordAssetDiscarded(ctx, a.File, int64(a.FileSize), fileevent.DiscardedLocalDuplicate,
				fmt.Sprintf("already present in input as %s", originalName))
		} else {
			// Record as processed - server duplicate
			uc.app.FileProcessor().RecordAssetProcessed(ctx, a.File, int64(a.FileSize), fileevent.DiscardedServerDuplicate)
		}
	} else {
		// Record successful upload
		uc.app.FileProcessor().RecordAssetProcessed(ctx, a.File, int64(a.FileSize), fileevent.ProcessedUploadSuccess)
	}
	a.ID = ar.ID

	// // DEBGUG
	//  if theID, ok := uc.assetIndex.byI

	if a.FromApplication != nil && ar.Status != immich.StatusDuplicate {
		// metadata from application (immich or google photos) are forced.
		// if a.Description != "" || (a.Latitude != 0 && a.Longitude != 0) || a.Rating != 0 || !a.CaptureDate.IsZero() {
		a.UseMetadata(a.FromApplication)
		_, err := uc.client.Immich.UpdateAsset(ctx, a.ID, immich.UpdAssetField{
			Description:      a.Description,
			Latitude:         a.Latitude,
			Longitude:        a.Longitude,
			Rating:           a.Rating,
			DateTimeOriginal: a.CaptureDate,
		})
		if err != nil {
			// Record metadata update error
			uc.app.FileProcessor().RecordAssetError(ctx, a.File, int64(a.FileSize), fileevent.ErrorServerError, err)
			return err
		}
		// Record successful metadata update
		uc.app.FileProcessor().Logger().Record(ctx, fileevent.ProcessedMetadataUpdated, a.File)
	}
	uc.assetIndex.addLocalAsset(a)
	return nil
}

// replaceAsset replaces an asset on the server. It uploads the new asset, copies the metadata from the old one and deletes the old one.
// https://github.com/immich-app/immich/pull/23172#issue-3542430029
func (uc *UpCmd) replaceAsset(ctx context.Context, newAsset, oldAsset *assets.Asset) error {
	// 1. Upload the new asset
	ar, err := uc.client.Immich.AssetUpload(ctx, newAsset)
	if err != nil {
		// Record upload error
		uc.app.FileProcessor().RecordAssetError(ctx, newAsset.File, int64(newAsset.FileSize), fileevent.ErrorServerError, err)
		return err // Must signal the error to the caller
	}
	newAsset.ID = ar.ID
	if ar.Status == immich.UploadDuplicate {
		// Record as processed - server duplicate
		uc.app.FileProcessor().RecordAssetProcessed(ctx, newAsset.File, int64(newAsset.FileSize), fileevent.DiscardedServerDuplicate)
		return nil
	}

	// 2. copy metadata from existing asset to the new asset
	err = uc.client.Immich.CopyAsset(ctx, oldAsset.ID, ar.ID)
	if err != nil {
		// Record copy error
		uc.app.FileProcessor().RecordAssetError(ctx, newAsset.File, int64(newAsset.FileSize), fileevent.ErrorServerError, err)
		return err // Must signal the error to the caller
	}

	// 3. Delete the existing asset
	err = uc.client.Immich.DeleteAssets(ctx, []string{oldAsset.ID}, true)
	if err != nil {
		// Record delete error
		uc.app.FileProcessor().RecordAssetError(ctx, newAsset.File, int64(newAsset.FileSize), fileevent.ErrorServerError, err)
		return err // Must signal the error to the caller
	}
	uc.assetIndex.replaceAsset(newAsset, oldAsset)
	// Record successful upgrade
	// uc.app.FileProcessor().RecordAssetProcessed(ctx, newAsset.File, int64(newAsset.FileSize), fileevent.ProcessedUploadUpgraded)
	return nil
}

// manageAssetAlbums add the assets to the albums listed.
// If an album does not exist, it is created.
// If the album already has the asset, it is not added.
// Errors are logged.
func (uc *UpCmd) manageAssetAlbums(ctx context.Context, f fshelper.FSAndName, ID string, albums []assets.Album) {
	if len(albums) == 0 {
		return
	}

	for _, album := range albums {
		al := assets.NewAlbum("", album.Title, album.Description)
		if uc.albumsCache.AddIDToCollection(al.Title, album, ID) {
			// Record album addition event
			uc.app.FileProcessor().Logger().Record(ctx, fileevent.ProcessedAlbumAdded, f, "album", al.Title)
		}
	}
}

func (uc *UpCmd) manageAssetTags(ctx context.Context, a *assets.Asset) {
	if len(a.Tags) == 0 {
		return
	}

	tags := make([]string, len(a.Tags))
	for i := range a.Tags {
		tags[i] = a.Tags[i].Name
	}
	for _, t := range a.Tags {
		if uc.tagsCache.AddIDToCollection(t.Name, t, a.ID) {
			// Record tag event
			uc.app.FileProcessor().Logger().Record(ctx, fileevent.ProcessedTagged, a.File, "tag", t.Value)
		}
	}
}

func (uc *UpCmd) DeleteServerAssets(ctx context.Context, ids []string) error {
	uc.app.Log().Message("%d server assets to delete.", len(ids))
	return uc.client.Immich.DeleteAssets(ctx, ids, false)
}

func (uc *UpCmd) processUploadedAsset(ctx context.Context, a *assets.Asset) {
	// TODO: current version of Immich doesn't allow to add same tag to an asset already tagged.
	//       there is no mean to go the list of tagged assets for a given tag.
	uc.manageAssetAlbums(ctx, a.File, a.ID, a.Albums)
	uc.addCommandLineTags(a)
	uc.manageAssetTags(ctx, a)
}

/*
func (upCmd *UpCmd) DeleteLocalAssets() error {
	upCmd.RootImmichFlags.Message(fmt.Sprintf("%d local assets to delete.", len(upCmd.deleteLocalList)))

	for _, a := range upCmd.deleteLocalList {
		if !upCmd.DryRun {
			upCmd.Log.Info(fmt.Sprintf("delete file %q", a.Title))
			err := a.Remove()
			if err != nil {
				return err
			}
		} else {
			upCmd.Log.Info(fmt.Sprintf("file %q not deleted, dry run mode.", a.Title))
		}
	}
	return nil
}
*/
