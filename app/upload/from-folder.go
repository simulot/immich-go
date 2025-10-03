package upload

import (
	"context"

	"github.com/simulot/immich-go/app"
	"github.com/spf13/cobra"
)

func NewFromFolderCommand(ctx context.Context, parent *cobra.Command, app *app.Application, upOptions *UploadOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "from-folder [flags] <path>...",
		Short: "Upload photos from a folder",
		Args:  cobra.MinimumNArgs(1),
	}
	cmd.SetContext(ctx)
	flags := cmd.Flags()
	uo := &UploadOptions{}
	uo.ImportFolderOptions.RegisterFlags(flags, cmd)

	cmd.RunE = uo.runE

	return cmd
}
