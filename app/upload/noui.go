package upload

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/fileevent"
	"golang.org/x/sync/errgroup"
)

func (upCmd *UpCmd) runNoUI(ctx context.Context, app *app.Application) error {
	ctx, cancel := context.WithCancelCause(ctx)
	lock := sync.RWMutex{}
	defer cancel(nil)

	var preparationDone atomic.Bool

	stopProgress := make(chan any)
	var maxImmich, currImmich int
	spinner := []rune{' ', ' ', '.', ' ', ' '}
	spinIdx := 0

	immichUpdate := func(value, total int) {
		lock.Lock()
		currImmich, maxImmich = value, total
		lock.Unlock()
	}

	progressString := func() string {
		counts := app.Jnl().GetCounts()
		defer func() {
			spinIdx++
			if spinIdx == len(spinner) {
				spinIdx = 0
			}
		}()
		lock.Lock()
		immichPct := 0
		if maxImmich > 0 {
			immichPct = 100 * currImmich / maxImmich
		} else {
			immichPct = 100
		}
		lock.Unlock()

		return fmt.Sprintf("\rImmich read %d%%, Assets found: %d, Upload errors: %d, Uploaded %d %s", immichPct, app.Jnl().TotalAssets(), counts[fileevent.UploadServerError], counts[fileevent.Uploaded], string(spinner[spinIdx]))
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
		var groupChan chan *assets.Group
		var err error

		processGrp.Go(func() error {
			// Get immich asset
			err := upCmd.getImmichAssets(ctx, immichUpdate)
			if err != nil {
				cancel(err)
			}
			return err
		})
		processGrp.Go(func() error {
			return upCmd.getImmichAlbums(ctx)
		})
		processGrp.Go(func() error {
			// Run Prepare
			groupChan = upCmd.adapter.Browse(ctx)
			return err
		})
		err = processGrp.Wait()
		if err != nil {
			err := context.Cause(ctx)
			if err != nil {
				cancel(err)
				return err
			}
		}
		preparationDone.Store(true)
		err = upCmd.uploadLoop(ctx, groupChan)
		if err != nil {
			cancel(err)
		}

		counts := app.Jnl().GetCounts()
		messages := strings.Builder{}
		if counts[fileevent.Error]+counts[fileevent.UploadServerError] > 0 {
			messages.WriteString("Some errors have occurred. Look at the log file for details\n")
		}

		if messages.Len() > 0 {
			cancel(errors.New(messages.String()))
		}
		err = errors.Join(err, upCmd.finishing(ctx, app))
		close(stopProgress)
		return err
	})

	err := uiGrp.Wait()
	if err != nil {
		err = context.Cause(ctx)
	}
	return err
}
