package upload

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	gp "github.com/simulot/immich-go/adapters/googlePhotos"
	"github.com/simulot/immich-go/application"
	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/spf13/cobra"
)

func NewFromGooglePhotosCommand(ctx context.Context, parent *cobra.Command, app *application.Application, upOptions *UploadOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "from-google-photos [flags] <takeout-*.zip> | <takeout-folder>",
		Short: "Upload photos either from a zipped Google Photos takeout or decompressed archive",
		Args:  cobra.MinimumNArgs(1),
	}
	cmd.SetContext(ctx)
	options := &gp.ImportFlags{}
	options.AddFromGooglePhotosFlags(cmd, parent)

	cmd.RunE = func(cmd *cobra.Command, args []string) error { //nolint:contextcheck
		ctx := cmd.Context()
		log := app.Log()
		client := app.Client()

		fsyss, err := fshelper.ParsePath(args)
		if err != nil {
			return err
		}
		if len(fsyss) == 0 {
			log.Message("No file found matching the pattern: %s", strings.Join(args, ","))
			return errors.New("No file found matching the pattern: " + strings.Join(args, ","))
		}

		if options.TakeoutTag {
			gotIt := false
			for _, a := range args {
				if filepath.Ext(a) == ".zip" {
					options.TakeoutName = filepath.Base(a)
					if len(options.TakeoutName) > 4+4 {
						options.TakeoutName = "{takeout}/" + options.TakeoutName[:len(options.TakeoutName)-4-4]
						gotIt = true
						break
					}
				}
			}
			if !gotIt {
				log.Message("Can't set the takeout tag: no .zip file in the arguments")
				options.TakeoutTag = false
			}
		}

		upOptions.Filters = append(upOptions.Filters, options.ManageBurst.GroupFilter(), options.ManageRawJPG.GroupFilter(), options.ManageHEICJPG.GroupFilter())

		options.SupportedMedia = client.Immich.SupportedMedia()
		options.InfoCollector = filenames.NewInfoCollector(app.GetTZ(), options.SupportedMedia)
		adapter, err := gp.NewTakeout(ctx, app.Jnl(), options, fsyss...)
		if err != nil {
			return err
		}
		return newUpload(UpModeGoogleTakeout, app, upOptions).setTakeoutOptions(options).run(ctx, adapter, app)
	}

	return cmd
}
