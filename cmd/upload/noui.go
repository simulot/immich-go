package upload

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/simulot/immich-go/helpers/fileevent"
	"golang.org/x/sync/errgroup"
)

func (app *UpCmd) runNoUI(ctx context.Context) error {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	var preparationDone atomic.Bool

	stopProgress := make(chan any)
	var maxImmich, currImmich int
	spinner := []rune{' ', ' ', '.', ' ', ' '}
	spinIdx := 0

	immichUpdate := func(value, total int) {
		currImmich, maxImmich = value, total
	}

	progressString := func() string {
		counts := app.Jnl.GetCounts()
		defer func() {
			spinIdx++
			if spinIdx == len(spinner) {
				spinIdx = 0
			}
		}()
		immichPct := 0
		if maxImmich > 0 {
			immichPct = 100 * currImmich / maxImmich
		} else {
			immichPct = 100
		}

		if app.GooglePhotos {
			gpTotal := app.Jnl.TotalAssets()
			gpProcessed := app.Jnl.TotalProcessedGP()

			gpPercent := int(100 * gpProcessed / gpTotal)
			upProcessed := int64(0)
			if preparationDone.Load() {
				upProcessed = app.Jnl.TotalProcessed(app.ForceUploadWhenNoJSON)
			}
			upTotal := app.Jnl.TotalAssets()
			upPercent := 100 * upProcessed / upTotal

			return fmt.Sprintf("\rImmich read %d%%, Assets found: %d, Google Photos Analysis: %d%%, Upload errors: %d, Uploaded %d%% %s",
				immichPct, app.Jnl.TotalAssets(), gpPercent, counts[fileevent.UploadServerError], upPercent, string(spinner[spinIdx]))
		}

		return fmt.Sprintf("\rImmich read %d%%, Assets found: %d, Upload errors: %d, Uploaded %d %s", immichPct, app.Jnl.TotalAssets(), counts[fileevent.UploadServerError], counts[fileevent.Uploaded], string(spinner[spinIdx]))
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
		preparationDone.Store(true)
		err = app.uploadLoop(ctx)
		if err != nil {
			cancel(err)
		}

		counts := app.Jnl.GetCounts()
		messages := strings.Builder{}
		if counts[fileevent.Error]+counts[fileevent.UploadServerError] > 0 {
			messages.WriteString("Some errors have occurred. Look at the log file for details\n")
		}
		if app.GooglePhotos && counts[fileevent.AnalysisMissingAssociatedMetadata] > 0 && !app.ForceUploadWhenNoJSON {
			messages.WriteString(fmt.Sprintf("\n%d JSON files are missing.\n", counts[fileevent.AnalysisMissingAssociatedMetadata]))
			messages.WriteString("- Verify if all takeout parts have been included in the processing.\n")
			messages.WriteString("- Request another takeout, either for one year at a time or in smaller increments.\n")
		}
		if messages.Len() > 0 {
			cancel(errors.New(messages.String()))
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
