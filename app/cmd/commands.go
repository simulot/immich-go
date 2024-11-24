package cmd

import (
	"context"

	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/app/cmd/archive"
	"github.com/simulot/immich-go/app/cmd/upload"
	"github.com/spf13/cobra"
)

func AddCommands(cmd *cobra.Command, ctx context.Context, app *app.Application) {
	cmd.AddCommand(
		upload.NewUploadCommand(ctx, app),
		archive.NewArchiveCommand(ctx, app),
	)
}
