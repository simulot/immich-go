package upload

import (
	"errors"
	"strings"

	gp "github.com/simulot/immich-go/adapters/googlePhotos"
	"github.com/simulot/immich-go/cmd"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/spf13/cobra"
)

func addFromGooglePhotosCommand(uploadCmd *cobra.Command, rootFlags *cmd.RootImmichFlags) {
	cmdGP := &cobra.Command{
		Use:   "from-google-photos [flags] <takeout-*.zip> | <takeout-folder>",
		Short: "Import files either from a Google Photos takeout zipped archive or decompressed folder",
		Args:  cobra.MinimumNArgs(1),
	}
	cmdGP.Flags().SortFlags = false

	cmdUpServerFlags := cmd.AddImmichServerFlagSet(cmdGP, rootFlags)
	CommonFlags := addCommonFlags(cmdGP)
	gpFlags := gp.ImportFlags{}
	gp.AddGoogleTakeoutFlags(cmdGP, &gpFlags)
	uploadCmd.AddCommand(cmdGP)

	cmdGP.RunE = func(cmd *cobra.Command, args []string) error {
		UpCmd := &UpCmd{
			Root:              rootFlags,
			Server:            cmdUpServerFlags,
			CommonFlags:       CommonFlags,
			GooglePhotosFlags: &gpFlags,
		}

		ctx := cmdGP.Context()
		err := UpCmd.Root.Open(cmdGP)
		if err != nil {
			return err
		}

		UpCmd.Jnl = fileevent.NewRecorder(rootFlags.Log, false)

		fsyss, err := fshelper.ParsePath(args)
		if err != nil {
			return err
		}
		if len(fsyss) == 0 {
			UpCmd.Root.Message("No file found matching the pattern: %s", strings.Join(args, ","))
			return errors.New("No file found matching the pattern: " + strings.Join(args, ","))
		}

		err = UpCmd.Server.Open(rootFlags)
		if err != nil {
			return err
		}

		UpCmd.GooglePhotosFlags.SupportedMedia = UpCmd.Server.Immich.SupportedMedia()

		adapter, err := gp.NewTakeout(ctx, UpCmd.Jnl, UpCmd.GooglePhotosFlags, fsyss...)
		if err != nil {
			return err
		}

		return UpCmd.run(ctx, adapter)
	}
}
