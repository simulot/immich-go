package upload

import (
	"context"
	"fmt"
	"time"

	"github.com/simulot/immich-go/helpers/fileevent"
	"golang.org/x/sync/errgroup"
)

func (app *UpCmd) runNoUI(ctx context.Context) error {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	stopProgress := make(chan any)
	var maxImmich, currImmich int
	spinner := []rune{' ', ' ', '.', ' ', ' '}
	spinIdx := 0

	immichUpdate := func(value, total int) {
		currImmich, maxImmich = value, total
	}

	progressString := func() string {
		var s string
		counts := app.Jnl.GetCounts()
		immichPct := 0
		if maxImmich > 0 {
			immichPct = 100 * currImmich / maxImmich
		} else {
			immichPct = 100
		}
		AssetsToUpload := counts[fileevent.DiscoveredImage] + counts[fileevent.DiscoveredVideo] - counts[fileevent.AnalysisLocalDuplicate]

		ProcessedAssets := counts[fileevent.Uploaded] +
			counts[fileevent.UploadServerError] +
			counts[fileevent.UploadNotSelected] +
			counts[fileevent.UploadUpgraded] +
			counts[fileevent.UploadServerDuplicate] +
			counts[fileevent.UploadServerBetter] +
			counts[fileevent.DiscoveredDiscarded]
		if !app.ForceUploadWhenNoJSON {
			ProcessedAssets += counts[fileevent.AnalysisMissingAssociatedMetadata]
		}

		if app.GooglePhotos {
			gpPct := 0
			upPct := 0

			if AssetsToUpload > 0 {
				gpPct = int(100 * (counts[fileevent.AnalysisAssociatedMetadata] + counts[fileevent.AnalysisMissingAssociatedMetadata]) / (counts[fileevent.DiscoveredImage] + counts[fileevent.DiscoveredVideo]))

				upPct = int(100 * ProcessedAssets / AssetsToUpload)
			}

			s = fmt.Sprintf("\rImmich read %d%%, Google Photos Analysis: %d%%, Upload errors: %d, Uploaded %d%% %s", immichPct, gpPct, counts[fileevent.UploadServerError], upPct, string(spinner[spinIdx]))
		} else {
			s = fmt.Sprintf("\rImmich read %d%%, Processed %d, Upload errors: %d, Uploaded %d %s", immichPct, ProcessedAssets, counts[fileevent.UploadServerError], counts[fileevent.Uploaded], string(spinner[spinIdx]))
		}
		spinIdx++
		if spinIdx == len(spinner) {
			spinIdx = 0
		}
		return s
	}
	uiGrp := errgroup.Group{}

	uiGrp.Go(func() error {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer func() {
			ticker.Stop()
			fmt.Println(progressString())
		}()
		for {
			select {
			case <-stopProgress:
				fmt.Print(progressString())
				return nil
			case <-ctx.Done():
				fmt.Print(progressString())
				return ctx.Err()
			case <-ticker.C:
				fmt.Print(progressString())
			}
		}
	})

	uiGrp.Go(func() error {
		processGrp := errgroup.Group{}

		processGrp.Go(func() error {
			// Get immich asset
			err := app.getImmichAssets(ctx, immichUpdate)
			if err != nil {
				cancel(err)
			}
			return err
		})
		processGrp.Go(func() error {
			return app.getImmichAlbums(ctx)
		})
		processGrp.Go(func() error {
			// Run Prepare
			err := app.browser.Prepare(ctx)
			if err != nil {
				cancel(err)
			}
			return err
		})
		err := processGrp.Wait()
		if err != nil {
			err := context.Cause(ctx)
			if err != nil {
				cancel(err)
				return err
			}
		}
		err = app.uploadLoop(ctx)
		if err != nil {
			cancel(err)
		}
		close(stopProgress)
		return err
	})

	err := uiGrp.Wait()
	if err != nil {
		err = context.Cause(ctx)
	}
	app.Jnl.Report()
	return err
}
