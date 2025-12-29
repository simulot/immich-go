package upload

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/simulot/immich-go/adapters"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/assets/cache"
	"github.com/simulot/immich-go/internal/exif/sidecars/xmpsidecar"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/filters"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/simulot/immich-go/internal/fshelper/osfs"
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

func (uc *UpCmd) pauseJobs(ctx context.Context) error {
	jobs := []string{"thumbnailGeneration", "metadataExtraction", "videoConversion", "faceDetection", "smartSearch"}
	for _, name := range jobs {
		_, err := uc.client.AdminImmich.SendJobCommand(ctx, name, "pause", true)
		if err != nil {
			uc.app.Log().Error("Immich Job command sent", "pause", name, "err", err.Error())
			return err
		}
		uc.app.Log().Info("Immich Job command sent", "pause", name)
	}
	return nil
}

func (uc *UpCmd) resumeJobs(_ context.Context) error {
	jobs := []string{"thumbnailGeneration", "metadataExtraction", "videoConversion", "faceDetection", "smartSearch"}

	// Start with a context not yet cancelled
	ctx := context.Background() //nolint
	for _, name := range jobs {
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
	if uc.albumsCache != nil {
		uc.albumsCache.Close()
	}
	if uc.tagsCache != nil {
		uc.tagsCache.Close()
	}

	// Resume immich background jobs if requested
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

	if uc.tagSidecarDir != "" {
		if remErr := os.RemoveAll(uc.tagSidecarDir); remErr != nil {
			uc.app.Log().Warn("failed to clean temporary tag sidecar directory", "dir", uc.tagSidecarDir, "err", remErr)
		}
		uc.tagSidecarDir = ""
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
	if !uc.TagViaSidecar {
		uc.tagsCache = cache.NewCollectionCache(50, func(tag assets.Tag, ids []string) (assets.Tag, error) {
			return uc.saveTags(ctx, tag, ids)
		})
	} else {
		uc.tagsCache = nil
	}

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
				// Get the album info from the server, with assets.
				r, err := uc.client.Immich.GetAlbumInfo(ctx, a.ID, false)
				if err != nil {
					uc.app.Log().Error("can't get the album info from the server", "album", a.AlbumName, "err", err)
					continue
				}
				ids := make([]string, 0, len(r.Assets))
				for _, aa := range r.Assets {
					ids = append(ids, aa.ID)
				}

				album := assets.NewAlbum(a.ID, a.AlbumName, a.Description)
				uc.albumsCache.NewCollection(a.AlbumName, album, ids)
				uc.app.Log().Info("got album from the server", "album", a.AlbumName, "assets", len(r.Assets))
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
		ids := []string{}
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

	// Upload assets from the group
	for _, a := range g.Assets {
		err := uc.handleAsset(ctx, a)
		errGroup = errors.Join(err)
	}

	// Manage groups
	// after the filtering and the upload, we can stack the assets

	if len(g.Assets) > 1 && g.Grouping != assets.GroupByNone {
		client := uc.client.Immich.(immich.ImmichStackInterface)
		ids := []string{g.Assets[g.CoverIndex].ID}
		for i, a := range g.Assets {
			// Record stacking event
			uc.app.FileProcessor().RecordNonAsset(ctx, g.Assets[i].File, 0, fileevent.ProcessedStacked)
			if i != g.CoverIndex && a.ID != "" {
				ids = append(ids, a.ID)
			}
		}
		if len(ids) > 1 {
			_, err := client.CreateStack(ctx, ids)
			if err != nil {
				uc.app.Log().Error("Can't create stack", "error", err)
			}
		}
	}

	return errGroup
}

func (uc *UpCmd) handleAsset(ctx context.Context, a *assets.Asset) error {
	defer func() {
		a.Close() // Close and clean resources linked to the local asset
	}()

	// var status stri g
	advice, err := uc.assetIndex.ShouldUpload(a, uc)
	if err != nil {
		return err
	}

	switch advice.Advice {
	case NotOnServer: // Upload and manage albums
		serverStatus, err := uc.uploadAsset(ctx, a)
		if err != nil {
			return err
		}

		uc.processUploadedAsset(ctx, a, serverStatus)
		return nil

	case SmallerOnServer: // Upload, manage albums and delete the server's asset

		// Remember existing asset's albums, if any
		a.Albums = append(a.Albums, advice.ServerAsset.Albums...)

		// Upload the superior asset
		serverStatus, err := uc.replaceAsset(ctx, a, advice.ServerAsset)
		if err != nil {
			return err
		}

		uc.processUploadedAsset(ctx, a, serverStatus)
		uc.app.FileProcessor().RecordAssetProcessed(ctx, a.File, int64(a.FileSize), fileevent.ProcessedUploadUpgraded)

		return nil

	case AlreadyProcessed: // SHA1 already processed
		// Record as discarded - duplicate in input
		uc.app.FileProcessor().RecordNonAsset(ctx, a.File, int64(a.FileSize), fileevent.DiscardedLocalDuplicate)
		uc.app.FileProcessor().RecordAssetProcessed(ctx, a.File, int64(a.FileSize), fileevent.ProcessedMetadataUpdated)
		uc.manageAssetAlbums(ctx, a.File, a.ID, a.Albums)
		return nil

	case SameOnServer:
		a.ID = advice.ServerAsset.ID
		a.Albums = append(a.Albums, advice.ServerAsset.Albums...)
		// Record as processed - duplicate on server
		uc.app.FileProcessor().RecordNonAsset(ctx, a.File, int64(a.FileSize), fileevent.DiscardedServerDuplicate)
		uc.app.FileProcessor().RecordAssetProcessed(ctx, a.File, int64(a.FileSize), fileevent.ProcessedMetadataUpdated)
		uc.manageAssetAlbums(ctx, a.File, a.ID, a.Albums)

	case BetterOnServer: // and manage albums
		a.ID = advice.ServerAsset.ID
		// Record as discarded - server has better version
		uc.app.FileProcessor().RecordAssetDiscarded(ctx, a.File, int64(a.FileSize), fileevent.ProcessedMetadataUpdated, advice.Message)
		uc.manageAssetAlbums(ctx, a.File, a.ID, a.Albums)

	case ForceUpload:
		var serverStatus string
		var err error

		if advice.ServerAsset != nil {
			// Remember existing asset's albums, if any
			a.Albums = append(a.Albums, advice.ServerAsset.Albums...)

			// Upload the superior asset
			serverStatus, err = uc.replaceAsset(ctx, a, advice.ServerAsset)
		} else {
			serverStatus, err = uc.uploadAsset(ctx, a)
		}
		if err != nil {
			return err
		}

		uc.processUploadedAsset(ctx, a, serverStatus)
		return nil
	}

	return nil
}

// uploadAsset uploads the asset to the server.
// set the server's asset ID to the asset.
// return the duplicate condition and error.
func (uc *UpCmd) uploadAsset(ctx context.Context, a *assets.Asset) (string, error) {
	defer uc.app.Log().Debug("upload asset", "file", a)

	if uc.SessionTag {
		a.AddTag(uc.session)
	}
	for _, tag := range uc.Tags {
		a.AddTag(tag)
	}
	if err := uc.prepareTagsSidecar(ctx, a); err != nil {
		if uc.app.FileProcessor() != nil {
			uc.app.FileProcessor().RecordAssetError(ctx, a.File, int64(a.FileSize), fileevent.ErrorFileAccess, err)
		} else if uc.app.Log() != nil {
			uc.app.Log().Error("prepare sidecar failed", "file", a.File, "error", err)
		}
		return "", err
	}

	ar, err := uc.client.Immich.AssetUpload(ctx, a)
	if err != nil {
		// Record upload error
		uc.app.FileProcessor().RecordAssetError(ctx, a.File, int64(a.FileSize), fileevent.ErrorServerError, err)
		return "", err // Must signal the error to the caller
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
			return "", err
		}
		// Record successful metadata update
		uc.app.FileProcessor().Logger().Record(ctx, fileevent.ProcessedMetadataUpdated, a.File)
	}
	uc.assetIndex.addLocalAsset(a)
	return ar.Status, nil
}

// replaceAsset replaces an asset on the server. It uploads the new asset, copies the metadata from the old one and deletes the old one.
// https://github.com/immich-app/immich/pull/23172#issue-3542430029
func (uc *UpCmd) replaceAsset(ctx context.Context, newAsset, oldAsset *assets.Asset) (string, error) {
	if err := uc.prepareTagsSidecar(ctx, newAsset); err != nil {
		if uc.app.FileProcessor() != nil {
			uc.app.FileProcessor().RecordAssetError(ctx, newAsset.File, int64(newAsset.FileSize), fileevent.ErrorFileAccess, err)
		} else if uc.app.Log() != nil {
			uc.app.Log().Error("prepare sidecar failed", "file", newAsset.File, "error", err)
		}
		return "", err
	}
	// 1. Upload the new asset
	ar, err := uc.client.Immich.AssetUpload(ctx, newAsset)
	if err != nil {
		// Record upload error
		uc.app.FileProcessor().RecordAssetError(ctx, newAsset.File, int64(newAsset.FileSize), fileevent.ErrorServerError, err)
		return "", err // Must signal the error to the caller
	}
	newAsset.ID = ar.ID
	if ar.Status == immich.UploadDuplicate {
		// Record as processed - server duplicate
		uc.app.FileProcessor().RecordAssetProcessed(ctx, newAsset.File, int64(newAsset.FileSize), fileevent.DiscardedServerDuplicate)
		return immich.UploadDuplicate, nil
	}

	// 2. copy metadata from existing asset to the new asset
	err = uc.client.Immich.CopyAsset(ctx, oldAsset.ID, ar.ID)
	if err != nil {
		// Record copy error
		uc.app.FileProcessor().RecordAssetError(ctx, newAsset.File, int64(newAsset.FileSize), fileevent.ErrorServerError, err)
		return "", err // Must signal the error to the caller
	}

	// 3. Delete the existing asset
	err = uc.client.Immich.DeleteAssets(ctx, []string{oldAsset.ID}, true)
	if err != nil {
		// Record delete error
		uc.app.FileProcessor().RecordAssetError(ctx, newAsset.File, int64(newAsset.FileSize), fileevent.ErrorServerError, err)
		return "", err // Must signal the error to the caller
	}
	uc.assetIndex.replaceAsset(newAsset, oldAsset)
	// Record successful upgrade
	// uc.app.FileProcessor().RecordAssetProcessed(ctx, newAsset.File, int64(newAsset.FileSize), fileevent.ProcessedUploadUpgraded)
	return "", nil
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
	if uc.TagViaSidecar {
		return
	}
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

func (uc *UpCmd) prepareTagsSidecar(ctx context.Context, a *assets.Asset) error {
	if !uc.TagViaSidecar {
		return nil
	}

	loggedTags := make(map[string]struct{})
	fp := uc.app.FileProcessor()
	for _, t := range a.Tags {
		if t.Value == "" {
			continue
		}
		if _, seen := loggedTags[t.Value]; seen {
			continue
		}
		loggedTags[t.Value] = struct{}{}
		if fp != nil && fp.Logger() != nil {
			fp.Logger().Record(ctx, fileevent.ProcessedTagged, a.File, "tag", t.Value, "method", "sidecar")
		}
	}

	tagSet := make(map[string]struct{})
	for value := range loggedTags {
		tagSet[value] = struct{}{}
	}
	for _, t := range a.Tags {
		if t.Value != "" {
			tagSet[t.Value] = struct{}{}
		}
	}
	if a.FromSideCar != nil {
		for _, t := range a.FromSideCar.Tags {
			if t.Value != "" {
				tagSet[t.Value] = struct{}{}
			}
		}
	}

	tagValues := make([]string, 0, len(tagSet))
	for value := range tagSet {
		tagValues = append(tagValues, value)
	}
	sort.Strings(tagValues)

	if len(tagValues) == 0 {
		uc.app.Log().Debug("tag via sidecar skipped", "file", a.OriginalFileName, "reason", "no tags")
		return nil
	}

	tagList := strings.Join(tagValues, ", ")
	if len(loggedTags) > 0 {
		uc.app.Log().Info("tag via sidecar applied", "file", a.OriginalFileName, "tags", tagList)
	} else {
		uc.app.Log().Debug("tag via sidecar preserved existing tags", "file", a.OriginalFileName, "tags", tagList)
	}

	var source []byte
	if a.FromSideCar != nil && a.FromSideCar.File.FS() != nil && a.FromSideCar.File.Name() != "" {
		f, err := a.FromSideCar.File.Open()
		if err != nil {
			return fmt.Errorf("open existing sidecar: %w", err)
		}
		source, err = io.ReadAll(f)
		_ = f.Close()
		if err != nil {
			return fmt.Errorf("read existing sidecar: %w", err)
		}
	}

	if len(source) == 0 && len(tagValues) == 0 {
		return nil
	}

	updated, err := xmpsidecar.UpdateTags(source, tagValues)
	if err != nil {
		return fmt.Errorf("update sidecar tags: %w", err)
	}

	if uc.tagSidecarDir == "" {
		dir, dirErr := os.MkdirTemp("", "immich-go-tag-sidecars-*")
		if dirErr != nil {
			return fmt.Errorf("create temporary sidecar directory: %w", dirErr)
		}
		uc.tagSidecarDir = dir
	}

	fileName := path.Base(a.OriginalFileName) + ".xmp"
	fullPath := filepath.Join(uc.tagSidecarDir, fileName)
	if writeErr := os.WriteFile(fullPath, updated, 0o600); writeErr != nil {
		return fmt.Errorf("write temporary sidecar: %w", writeErr)
	}

	uc.app.Log().Debug("tag via sidecar wrote temporary sidecar", "file", a.OriginalFileName, "path", fullPath)

	md := &assets.Metadata{}
	if err = xmpsidecar.ReadXMP(bytes.NewReader(updated), md); err != nil {
		return fmt.Errorf("reload updated sidecar metadata: %w", err)
	}

	fs := osfs.DirFS(uc.tagSidecarDir)
	md.File = fshelper.FSName(fs, fileName)
	a.FromSideCar = a.UseMetadata(md)

	return nil
}

func (uc *UpCmd) DeleteServerAssets(ctx context.Context, ids []string) error {
	uc.app.Log().Message("%d server assets to delete.", len(ids))
	return uc.client.Immich.DeleteAssets(ctx, ids, false)
}

func (uc *UpCmd) processUploadedAsset(ctx context.Context, a *assets.Asset, serverStatus string) {
	if serverStatus != immich.StatusDuplicate {
		// TODO: current version of Immich doesn't allow to add same tag to an asset already tagged.
		//       there is no mean to go the list of tagged assets for a given tag.
		uc.manageAssetAlbums(ctx, a.File, a.ID, a.Albums)
		uc.manageAssetTags(ctx, a)
	}
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
