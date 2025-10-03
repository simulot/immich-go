package upload

import (
	"context"

	"github.com/simulot/immich-go/app"
	"github.com/spf13/cobra"
)

func NewFromPicasaCommand(ctx context.Context, parent *cobra.Command, app *app.Application, upOptions *UploadOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "from-picasa [flags] <path>...",
		Short: "Upload photos from a Picasa folder or zip file",
		Args:  cobra.MinimumNArgs(1),
	}
	cmd.SetContext(ctx)
	flags := cmd.Flags()
	uo := &UploadOptions{}
	uo.ImportFolderOptions.RegisterFlags(flags, cmd)

	cmd.RunE = uo.runE

	return cmd
}
