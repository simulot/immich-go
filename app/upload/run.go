package upload

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/simulot/immich-go/adapters"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/assets/cache"
	cliflags "github.com/simulot/immich-go/internal/cliFlags"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/filters"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/simulot/immich-go/internal/worker"
	"golang.org/x/sync/errgroup"
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
		uc.albumsCache = nil
	}
	if uc.tagsCache != nil {
		uc.tagsCache.Close()
		uc.tagsCache = nil
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

	uc.adapter = adapter

	// Check if adapter supports batched mode
	drp, hasBatching := adapter.(adapters.DateRangeProvider)
	if hasBatching {
		return uc.uploadBatched(ctx, adapter, drp)
	}

	// Fall back to existing behavior (unchanged)
	return uc.uploadUnbatched(ctx, adapter)
}

// uploadUnbatched is the original upload path: fetch all server assets, then upload everything.
// Used when the adapter does not implement DateRangeProvider.
func (uc *UpCmd) uploadUnbatched(ctx context.Context, adapter adapters.Reader) error {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	uc.albumsCache = cache.NewCollectionCache(50, func(album assets.Album, ids []string) (assets.Album, error) {
		return uc.saveAlbum(ctx, album, ids)
	})
	uc.tagsCache = cache.NewCollectionCache(50, func(tag assets.Tag, ids []string) (assets.Tag, error) {
		return uc.saveTags(ctx, tag, ids)
	})

	runner := uc.runUI
	uc.assetIndex = newAssetIndex()
	uc.immichAssetsReady = make(chan struct{})

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

// uploadBatched processes uploads month by month, scoping server queries
// to each month's date range for memory efficiency.
func (uc *UpCmd) uploadBatched(ctx context.Context, adapter adapters.Reader, drp adapters.DateRangeProvider) error {
	// 1. PreScan to get month list
	uc.app.Log().Info("Pre-scanning to determine month list...")
	months, err := drp.PreScan(ctx)
	if err != nil {
		return fmt.Errorf("pre-scan failed: %w", err)
	}
	uc.app.Log().Info(fmt.Sprintf("Pre-scan found %d months", len(months)))

	// Check if there are no-date files; if so, add a "no-date" batch at the end
	if provider, ok := adapter.(interface{ HasNoDateFiles() bool }); ok && provider.HasNoDateFiles() {
		months = append(months, "no-date")
	}

	// 2. Load state
	serverURL := uc.client.Server
	archivePath := "" // derive from adapter if possible
	if named, ok := adapter.(interface{ ArchivePath() string }); ok {
		archivePath = named.ArchivePath()
	}

	stateDir := uc.StateDir
	if stateDir == "" {
		stateDir = DefaultStateDir(serverURL, archivePath)
	}

	uc.state, err = LoadState(stateDir, serverURL, archivePath)
	if err != nil {
		return fmt.Errorf("can't load state: %w", err)
	}

	if uc.ResetState {
		uc.state.ResetState()
		if err := uc.state.SaveState(); err != nil {
			return fmt.Errorf("can't save reset state: %w", err)
		}
		fmt.Printf("State has been reset for %s -> %s\n", archivePath, serverURL)
	}

	if uc.ShowState {
		completed := len(uc.state.CompletedMonths)
		total := len(months)
		remaining := 0
		for _, m := range months {
			if !uc.state.IsMonthCompleted(m) {
				remaining++
			}
		}
		nextMonth := ""
		for _, m := range months {
			if !uc.state.IsMonthCompleted(m) {
				nextMonth = m
				break
			}
		}
		fmt.Printf("Upload state for %s -> %s\n", archivePath, serverURL)
		fmt.Printf("  Total months: %d\n", total)
		fmt.Printf("  Completed:    %d\n", completed)
		fmt.Printf("  Remaining:    %d\n", remaining)
		if nextMonth != "" {
			fmt.Printf("  Next month:   %s\n", nextMonth)
		}
		fmt.Printf("  Last updated: %s\n", uc.state.UpdatedAt.Format("2006-01-02 15:04"))
		return nil
	}

	// 3. Filter months: remove completed, apply batch-limit
	var remaining []string
	var skippedMonths []string
	for _, m := range months {
		if uc.state.IsMonthCompleted(m) {
			skippedMonths = append(skippedMonths, m)
		} else {
			remaining = append(remaining, m)
		}
	}

	if len(skippedMonths) > 0 {
		uc.app.Log().Info(fmt.Sprintf("Skipping %d already completed months: %s", len(skippedMonths), strings.Join(skippedMonths, ", ")))
	}

	if len(remaining) == 0 {
		uc.app.Log().Info("All months already completed. Nothing to do.")
		return nil
	}

	if uc.BatchLimit > 0 && len(remaining) > uc.BatchLimit {
		remaining = remaining[:uc.BatchLimit]
	}

	uc.app.Log().Info(fmt.Sprintf("Processing %d months (of %d remaining): %s", len(remaining), len(months)-len(skippedMonths), strings.Join(remaining, ", ")))

	// 4. Build the month-loop function that will be called from within either UI or NoUI
	uc.batchTotal = len(remaining)
	monthLoop := func(ctx context.Context) error {
		for i, month := range remaining {
			select {
			case <-ctx.Done():
				return context.Cause(ctx)
			default:
			}
			uc.batchCurrent = i + 1
			err := uc.processMonth(ctx, adapter, drp, month)
			if err != nil {
				return err
			}
		}
		return nil
	}

	// 5. Run with UI or NoUI
	useUI := !uc.NoUI
	if useUI {
		_, err := tcell.NewScreen()
		if err != nil {
			uc.app.Log().Warn("can't initialize the screen for the UI mode. Falling back to no-gui mode", "err", err)
			fmt.Println("can't initialize the screen for the UI mode. Falling back to no-gui mode")
			useUI = false
		}
	}

	if useUI {
		err = uc.runBatchedUI(ctx, monthLoop)
	} else {
		err = monthLoop(ctx)
	}
	if err != nil {
		return err
	}

	// Print batch summary
	counts := uc.app.FileProcessor().Logger().GetCounts()
	uploaded := counts[fileevent.ProcessedUploadSuccess]
	skipped := counts[fileevent.DiscardedServerDuplicate]
	errCount := counts[fileevent.ErrorUploadFailed] + counts[fileevent.ErrorServerError] + counts[fileevent.ErrorFileAccess]

	firstMonth := remaining[0]
	lastMonth := remaining[len(remaining)-1]
	fmt.Printf("\nBatch complete: processed %s through %s (%d months)\n", firstMonth, lastMonth, len(remaining))
	fmt.Printf("  Uploaded: %d assets\n", uploaded)
	fmt.Printf("  Skipped:  %d (already on server)\n", skipped)
	fmt.Printf("  Errors:   %d\n", errCount)

	completedTotal := len(uc.state.CompletedMonths)
	totalMonths := len(months)
	nextMonth := ""
	for _, m := range months {
		if !uc.state.IsMonthCompleted(m) {
			nextMonth = m
			break
		}
	}
	if nextMonth != "" {
		fmt.Printf("Progress: %d/%d months complete. Next: %s\n", completedTotal, totalMonths, nextMonth)
	} else {
		fmt.Printf("Progress: %d/%d months complete. All done!\n", completedTotal, totalMonths)
	}

	return nil
}

// processMonth handles uploading all assets for a single month.
// It creates a fresh asset index, fetches server assets scoped to the month's date range,
// browses local files for the month, and runs the upload loop.
func (uc *UpCmd) processMonth(ctx context.Context, adapter adapters.Reader, drp adapters.DateRangeProvider, month string) error {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	uc.app.Log().Info(fmt.Sprintf("Processing month: %s", month))
	uc.currentMonth = month

	// Fresh asset index per month (key for memory efficiency)
	uc.assetIndex = newAssetIndex()
	uc.immichAssetsReady = make(chan struct{})

	// Set in-progress
	uc.state.SetInProgressMonth(month)
	if err := uc.state.SaveState(); err != nil {
		uc.app.Log().Error("can't save state", "err", err)
	}

	// Create fresh caches per month
	uc.albumsCache = cache.NewCollectionCache(50, func(album assets.Album, ids []string) (assets.Album, error) {
		return uc.saveAlbum(ctx, album, ids)
	})
	uc.tagsCache = cache.NewCollectionCache(50, func(tag assets.Tag, ids []string) (assets.Tag, error) {
		return uc.saveTags(ctx, tag, ids)
	})

	// Reset delete list per month
	uc.deleteServerList = nil
	uc.finished = false

	// Determine the browse function for this month
	groupChan := drp.BrowseMonth(ctx, month)

	// Determine the asset fetch function for this month
	var dr *cliflags.DateRange
	if month != "no-date" {
		after, before, err := adapters.MonthToDateRange(month)
		if err != nil {
			return fmt.Errorf("invalid month %q: %w", month, err)
		}
		dr = &cliflags.DateRange{After: after, Before: before}
	}

	// Run the parallel initialization + upload using runMonthNoUI
	// (always NoUI for batched mode to keep it simple and avoid UI flicker per month)
	err := uc.runMonthNoUI(ctx, dr, groupChan)

	// Clean up per-month resources
	if uc.albumsCache != nil {
		uc.albumsCache.Close()
		uc.albumsCache = nil
	}
	if uc.tagsCache != nil {
		uc.tagsCache.Close()
		uc.tagsCache = nil
	}

	if err != nil {
		return err
	}

	// Mark complete
	uc.state.CompleteMonth(month)
	if err := uc.state.SaveState(); err != nil {
		uc.app.Log().Error("can't save state after completing month", "err", err)
	}

	uc.app.Log().Info(fmt.Sprintf("Completed month: %s", month))
	return nil
}

// runMonthNoUI runs the parallel initialization and upload loop for a single month
// in the batched upload flow. It fetches server assets scoped to the date range,
// fetches albums, and runs the upload loop — all in parallel to avoid deadlock
// (BrowseMonth's channel must be drained concurrently with server asset fetching).
func (uc *UpCmd) runMonthNoUI(ctx context.Context, dr *cliflags.DateRange, groupChan chan *assets.Group) error {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	processGrp := errgroup.Group{}

	processGrp.Go(func() error {
		var err error
		if dr != nil {
			err = uc.getImmichAssetsFiltered(ctx, *dr, uc.immichUpdateFn)
		} else {
			// no-date batch: fetch all assets (or skip server fetch)
			err = uc.getImmichAssetsFiltered(ctx, cliflags.DateRange{}, uc.immichUpdateFn)
		}
		if err != nil {
			cancel(err)
		}
		return err
	})
	processGrp.Go(func() error {
		return uc.getImmichAlbums(ctx)
	})

	// uploadLoop must run in parallel so it drains groupChan while server
	// assets are being fetched. uploadLoop already waits for immichAssetsReady
	// before processing each asset.
	var uploadErr error
	processGrp.Go(func() error {
		uploadErr = uc.uploadLoop(ctx, groupChan)
		if uploadErr != nil {
			cancel(uploadErr)
		}
		return uploadErr
	})

	err := processGrp.Wait()
	if err != nil {
		cause := context.Cause(ctx)
		if cause != nil {
			return cause
		}
		return err
	}
	return nil
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

// getImmichAssetsFiltered fetches server assets scoped to a date range
// (used in batched mode). If the date range is not set, it falls back
// to GetAllAssets (used for the "no-date" batch).
func (uc *UpCmd) getImmichAssetsFiltered(ctx context.Context, dr cliflags.DateRange, updateFn progressUpdate) error {
	defer close(uc.immichAssetsReady)
	received := 0

	filter := func(a *immich.Asset) error {
		if updateFn != nil {
			defer func() {
				updateFn(received, received)
			}()
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			received++
			if a.OwnerID != uc.client.User.ID {
				return nil
			}
			if a.LibraryID != "" {
				return nil
			}
			uc.assetIndex.addImmichAsset(a)
			return nil
		}
	}

	var err error
	if dr.After.IsZero() && dr.Before.IsZero() {
		// no-date batch or empty date range: fetch all
		err = uc.client.Immich.GetAllAssets(ctx, filter)
	} else {
		err = uc.client.Immich.GetFilteredAssetsFn(ctx, immich.SearchOptions().All().WithDateRange(dr), filter)
	}
	if err != nil {
		return err
	}
	if updateFn != nil {
		updateFn(received, received)
	}
	uc.app.Log().Info(fmt.Sprintf("Assets on the server (filtered): %d", uc.assetIndex.len()))
	return nil
}

func (uc *UpCmd) uploadLoop(ctx context.Context, groupChan chan *assets.Group) error {
	ctx, cancel := context.WithCancelCause(ctx)

	useAdaptive := uc.app.OnErrors == cliflags.OnErrorsRetry
	throttle := worker.NewThrottle(uc.app.ConcurrentTask)

	var consecutiveSuccesses atomic.Int64

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
				if useAdaptive {
					throttle.Acquire()
				}
				workers.Submit(func() {
					if useAdaptive {
						defer throttle.Release()
					}
					err := uc.handleGroup(ctx, g)
					if err != nil {
						if useAdaptive && immich.IsRetryable(err) {
							consecutiveSuccesses.Store(0)
							cur := throttle.Current()
							newLevel := max(cur/2, 1)
							if newLevel < cur {
								throttle.SetConcurrency(newLevel)
								uc.app.Log().Info("Reducing concurrency due to server errors", "from", cur, "to", newLevel)
							}
							// Brief pause to let server recover
							select {
							case <-time.After(5 * time.Second):
							case <-ctx.Done():
							}
						}
						err = uc.app.ProcessError(err)
						if err != nil {
							cancel(err)
						}
					} else {
						if useAdaptive {
							n := consecutiveSuccesses.Add(1)
							if n%10 == 0 {
								cur := throttle.Current()
								maxLevel := throttle.Max()
								if cur < maxLevel {
									newLevel := min(cur+1, maxLevel)
									throttle.SetConcurrency(newLevel)
									uc.app.Log().Info("Increasing concurrency after consecutive successes", "from", cur, "to", newLevel)
								}
							}
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

	// Wait for server asset index to be ready before checking duplicates
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-uc.immichAssetsReady:
	}

	// In batched mode, check if this file was already uploaded in a previous run
	if uc.state != nil {
		source := ""
		if fs, ok := a.File.FS().(interface{ Name() string }); ok {
			source = fs.Name()
		}
		filePath := a.File.Name()
		if uc.state.IsFileUploaded(source, filePath, int64(a.FileSize)) {
			uc.resumeSkipped.Add(1)
			uc.app.Log().Info("Skipping file already uploaded in previous run", "file", filePath)
			return nil
		}
	}

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

	// Record upload in state file for resume support
	if uc.state != nil {
		source := ""
		if fs, ok := a.File.FS().(interface{ Name() string }); ok {
			source = fs.Name()
		}
		uc.state.RecordFileUploaded(source, a.File.Name(), int64(a.FileSize))
		if err := uc.state.SaveState(); err != nil {
			uc.app.Log().Error("can't save state after upload", "err", err)
		}
	}

	return ar.Status, nil
}

// replaceAsset replaces an asset on the server. It uploads the new asset, copies the metadata from the old one and deletes the old one.
// https://github.com/immich-app/immich/pull/23172#issue-3542430029
func (uc *UpCmd) replaceAsset(ctx context.Context, newAsset, oldAsset *assets.Asset) (string, error) {
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
