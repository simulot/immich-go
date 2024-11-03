package upload

import (
	"context"
	"errors"
	"strings"

	"github.com/simulot/immich-go/adapters/folder"
	"github.com/simulot/immich-go/commands/application"
	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/filters"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/spf13/cobra"
)

func NewFromFolderCommand(ctx context.Context, app *application.Application, upOptions *UploadOptions) *cobra.Command {
	upOptions.ManageHEICJPG = filters.HeicJpgKeepHeic
	upOptions.ManageRawJPG = filters.RawJPGKeepRaw
	upOptions.ManageBurst = filters.BurstkKeepRaw

	cmd := &cobra.Command{
		Use:   "from-folder [flags] <path>...",
		Short: "Upload photos from a folder",
		Args:  cobra.MinimumNArgs(1),
	}
	cmd.SetContext(ctx)
	options := &folder.ImportFolderOptions{}
	options.AddFromFolderFlags(cmd)

	cmd.RunE = func(cmd *cobra.Command, args []string) error { //nolint:contextcheck
		// ready to run
		ctx := cmd.Context()
		log := app.Log()
		client := app.Client()

		// parse arguments
		fsyss, err := fshelper.ParsePath(args)
		if err != nil {
			return err
		}
		if len(fsyss) == 0 {
			log.Message("No file found matching the pattern: %s", strings.Join(args, ","))
			return errors.New("No file found matching the pattern: " + strings.Join(args, ","))
		}

		// create the adapter for folders
		options.SupportedMedia = client.Immich.SupportedMedia()
		upOptions.Filters = append(upOptions.Filters, upOptions.ManageBurst.GroupFilter(), upOptions.ManageRawJPG.GroupFilter(), upOptions.ManageHEICJPG.GroupFilter())

		options.InfoCollector = filenames.NewInfoCollector(app.GetTZ(), options.SupportedMedia)
		adapter, err := folder.NewLocalFiles(ctx, app.Jnl(), options, fsyss...)
		if err != nil {
			return err
		}

		return newUpload(UpModeFolder, app, upOptions).run(ctx, adapter, app)
	}

	return cmd
}
