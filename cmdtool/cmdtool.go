package cmdtool

import (
	"context"
	"fmt"

	"github.com/simulot/immich-go/cmdtool/cmdalbum"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/logger"
)

func CommandTool(ctx context.Context, ic *immich.ImmichClient, logger *logger.Log, args []string) error {
	if len(args) > 0 {
		cmd := args[0]
		args = args[1:]

		switch cmd {
		case "album":
			return cmdalbum.AlbumCommand(ctx, ic, logger, args)
		}
	}

	return fmt.Errorf("the tool command need a sub command: album")
}
