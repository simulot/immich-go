package upload

import (
	"context"

	"github.com/simulot/immich-go/app"
	"github.com/spf13/cobra"
)

func NewFromICloudCommand(ctx context.Context, parent *cobra.Command, app *app.Application, upOptions *UploadOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "from-icloud [flags] <path>...",
		Short: "Upload photos from an iCloud takeout folder or zip file",
		Args:  cobra.MinimumNArgs(1),
	}
	cmd.SetContext(ctx)
	flags := cmd.Flags()
	uo := &UploadOptions{}
	uo.ImportFolderOptions.RegisterFlags(flags, cmd)
	cmd.RunE = uo.runE
	return cmd
}
