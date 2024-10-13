package commands

import (
	"context"

	"github.com/simulot/immich-go/commands/application"
	"github.com/simulot/immich-go/commands/upload"
	"github.com/spf13/cobra"
)

func AddCommands(cmd *cobra.Command, ctx context.Context, app *application.Application) {
	cmd.AddCommand(
		upload.NewUploadCommand(ctx, app),
	)
}
