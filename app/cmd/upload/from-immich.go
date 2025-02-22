package upload

import (
	"context"

	"github.com/simulot/immich-go/adapters/fromimmich"
	"github.com/simulot/immich-go/app"
	"github.com/spf13/cobra"
)

func NewFromImmichCommand(ctx context.Context, parent *cobra.Command, app *app.Application, upOptions *UploadOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "from-immich [flags]",
		Short: "Upload photos from another Immich server",
		Args:  cobra.MaximumNArgs(0),
	}
	cmd.SetContext(ctx)
	options := &fromimmich.FromImmichFlags{}
	options.AddFromImmichFlags(cmd, parent)

	cmd.RunE = func(cmd *cobra.Command, args []string) error { //nolint:contextcheck
		// ready to run
		ctx := cmd.Context()

		source, err := fromimmich.NewFromImmich(ctx, app, app.Jnl(), options)
		if err != nil {
			return err
		}

		return newUpload(UpModeFolder, app, upOptions).run(ctx, source, app, nil)
	}

	return cmd
}
