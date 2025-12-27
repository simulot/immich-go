package upload

import (
	"context"
	"errors"
	"strings"

	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/fileevent"
	"golang.org/x/sync/errgroup"
)

func (uc *UpCmd) runNoUI(ctx context.Context, app *app.Application) error {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	consoleSink := newConsoleSink(!app.UIExperimental)
	if processor := app.FileProcessor(); processor != nil {
		processor.Logger().RegisterSink(consoleSink)
		defer processor.Logger().UnregisterSink(consoleSink)
	}
	consoleSink.Start(ctx)
	defer consoleSink.Stop()
	uiGrp := errgroup.Group{}

	immichUpdate := func(value, total int) {
		consoleSink.SetImmichProgress(value, total)
	}

	uiGrp.Go(func() error {
		processGrp := errgroup.Group{}
		var groupChan chan *assets.Group
		var err error

		processGrp.Go(func() error {
			// Get immich asset
			err := uc.getImmichAssets(ctx, immichUpdate)
			if err != nil {
				cancel(err)
			}
			return err
		})
		processGrp.Go(func() error {
			return uc.getImmichAlbums(ctx)
		})
		processGrp.Go(func() error {
			// Run Prepare
			groupChan = uc.adapter.Browse(ctx)
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
		err = uc.uploadLoop(ctx, groupChan)
		if err != nil {
			cancel(err)
		}

		counts := app.FileProcessor().Logger().GetCounts()
		messages := strings.Builder{}
		if counts[fileevent.ErrorUploadFailed]+counts[fileevent.ErrorServerError]+counts[fileevent.ErrorFileAccess]+counts[fileevent.ErrorIncomplete] > 0 {
			messages.WriteString("Some errors have occurred. Look at the log file for details\n")
		}

		if messages.Len() > 0 {
			cancel(errors.New(messages.String()))
		}
		err = errors.Join(err, uc.finishing(ctx))
		return err
	})

	err := uiGrp.Wait()
	if err != nil {
		err = context.Cause(ctx)
	}
	return err
}
