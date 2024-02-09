package tool

import (
	"context"
	"fmt"

	"github.com/simulot/immich-go/cmd/album"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/logger"
)

func CommandTool(ctx context.Context, ic *immich.ImmichClient, logger *logger.Log, args []string) error {
	if len(args) > 0 {
		cmd := args[0]
		args = args[1:]

		if cmd == "album" {
			return album.AlbumCommand(ctx, ic, logger, args)
		}
	}

	return fmt.Errorf("the tool command need a sub command: album")
}
