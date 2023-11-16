package cmdtool

import (
	"context"
	"fmt"
	"immich-go/cmdtool/cmdalbum"
	"immich-go/immich"
	"immich-go/logger"
)

func CommandTool(ctx context.Context, ic *immich.ImmichClient, logger *logger.Logger, args []string) error {
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
