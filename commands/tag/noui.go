// package tag

// import (
// 	"context"
// 	"errors"
// 	"fmt"
// 	"strings"
// 	"sync/atomic"
// 	"time"

// 	"github.com/simulot/immich-go/internal/fileevent"
// 	"golang.org/x/sync/errgroup"
// )

// func (app *TagCmd) runNoUI(ctx context.Context) error {
// 	ctx, cancel := context.WithCancelCause(ctx)
// 	defer cancel(nil)

// 	var preparationDone atomic.Bool

// 	stopProgress := make(chan any)
// 	var maxImmich, currImmich int
// 	spinner := []rune{' ', ' ', '.', ' ', ' '}
// 	spinIdx := 0

// 	immichUpdate := func(value, total int) {
// 		currImmich, maxImmich = value, total
// 	}

// 	progressString := func() string {
// 		counts := app.Jnl.GetCounts()
// 		defer func() {
// 			spinIdx++
// 			if spinIdx == len(spinner) {
// 				spinIdx = 0
// 			}
// 		}()
// 		immichPct := 0
// 		if maxImmich > 0 {
// 			immichPct = 100 * currImmich / maxImmich
// 		} else {
// 			immichPct = 100
// 		}

// 		return fmt.Sprintf(
// 			"\rImmich read %d%%, Assets found: %d, Tag errors: %d, Tagged %d %s",
// 			immichPct,
// 			app.Jnl.TotalAssets(),
// 			counts[fileevent.UploadServerError],
// 			counts[fileevent.Uploaded],
// 			string(spinner[spinIdx]),
// 		)
// 	}
// 	uiGrp := errgroup.Group{}

// 	uiGrp.Go(func() error {
// 		ticker := time.NewTicker(500 * time.Millisecond)
// 		defer func() {
// 			ticker.Stop()
// 			fmt.Println(progressString())
// 		}()
// 		for {
// 			select {
// 			case <-stopProgress:
// 				fmt.Print(progressString())
// 				return nil
// 			case <-ctx.Done():
// 				fmt.Print(progressString())
// 				return ctx.Err()
// 			case <-ticker.C:
// 				fmt.Print(progressString())
// 			}
// 		}
// 	})

// 	uiGrp.Go(func() error {
// 		processGrp := errgroup.Group{}

// 		processGrp.Go(func() error {
// 			// Get immich asset
// 			err := app.getImmichAssets(ctx, immichUpdate)
// 			if err != nil {
// 				cancel(err)
// 			}
// 			return err
// 		})
// 		processGrp.Go(func() error {
// 			// Run Prepare
// 			err := app.browser.Prepare(ctx)
// 			if err != nil {
// 				cancel(err)
// 			}
// 			return err
// 		})
// 		err := processGrp.Wait()
// 		if err != nil {
// 			err := context.Cause(ctx)
// 			if err != nil {
// 				cancel(err)
// 				return err
// 			}
// 		}
// 		preparationDone.Store(true)
// 		err = app.uploadLoop(ctx)
// 		if err != nil {
// 			cancel(err)
// 		}

// 		counts := app.Jnl.GetCounts()
// 		messages := strings.Builder{}
// 		if counts[fileevent.Error]+counts[fileevent.UploadServerError] > 0 {
// 			messages.WriteString("Some errors have occurred. Look at the log file for details\n")
// 		}
// 		if messages.Len() > 0 {
// 			cancel(errors.New(messages.String()))
// 		}
// 		close(stopProgress)
// 		return err
// 	})

// 	err := uiGrp.Wait()
// 	if err != nil {
// 		err = context.Cause(ctx)
// 	}
// 	app.Jnl.ReportTags()
// 	return err
// }
