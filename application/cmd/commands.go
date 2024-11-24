package cmd

import (
	"context"

	"github.com/simulot/immich-go/application"
	"github.com/simulot/immich-go/application/cmd/archive"
	"github.com/simulot/immich-go/application/cmd/upload"
	"github.com/spf13/cobra"
)

func AddCommands(cmd *cobra.Command, ctx context.Context, app *application.Application) {
	cmd.AddCommand(
		upload.NewUploadCommand(ctx, app),
		archive.NewArchiveCommand(ctx, app),
	)
}
