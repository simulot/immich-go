package archive

import (
	"context"
	"errors"

	"github.com/simulot/immich-go/adapters"
	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/internal/fileevent"
)

func run(ctx context.Context, jnl *fileevent.Recorder, _ *app.Application, source adapters.Reader, dest adapters.AssetWriter) error {
	gChan := source.Browse(ctx)
	errCount := 0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case g, ok := <-gChan:
			if !ok {
				return nil
			}
			for _, a := range g.Assets {
				err := dest.WriteAsset(ctx, a)
				if err == nil {
					err = a.Close()
				}
				if err != nil {
					jnl.Log().Error(err.Error())
					errCount++
					if errCount > 5 {
						err := errors.New("too many errors, aborting")
						jnl.Log().Error(err.Error())
						return err
					}
				} else {
					jnl.Record(ctx, fileevent.Written, a)
				}
			}
		}
	}
}
