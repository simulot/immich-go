package archive

import (
	"context"

	"github.com/simulot/immich-go/adapters"
	"github.com/simulot/immich-go/commands/application"
)

func run(ctx context.Context, app *application.Application, adapter adapters.Adapter) error {
	groupChan := adapter.Browse(ctx)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case g, ok := <-groupChan:
			if !ok {
				return nil
			}

		}
	}
}
