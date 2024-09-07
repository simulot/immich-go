package tool

import (
	"context"
	"fmt"

	"github.com/simulot/immich-go/cmd"
	"github.com/simulot/immich-go/cmd/album"
)

func CommandTool(ctx context.Context, common *cmd.RootImmichFlags, args []string) error {
	if len(args) > 0 {
		cmd := args[0]
		args = args[1:]

		if cmd == "album" {
			return album.AlbumCommand(ctx, common, args)
		}
	}

	return fmt.Errorf("the tool command need a sub command: album")
}
