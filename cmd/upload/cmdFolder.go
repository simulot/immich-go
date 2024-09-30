package upload

import (
	"errors"
	"strings"

	"github.com/simulot/immich-go/adapters/folder"
	"github.com/simulot/immich-go/cmd"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/spf13/cobra"
)

func addFromFolderCommand(uploadCmd *cobra.Command, rootFlags *cmd.RootImmichFlags) {
	cmdFolder := &cobra.Command{
		Use:   "from-folder [flags] <folder> [<folder>...]",
		Short: "import files from a folder structure",
		Args:  cobra.MinimumNArgs(1),
	}
	cmdFolder.Flags().SortFlags = false

	cmdUpServerFlags := cmd.AddImmichServerFlagSet(cmdFolder, rootFlags)
	CommonFlags := addCommonFlags(cmdFolder)

	UploadFolderFlags := folder.ImportFlags{}
	folder.AddUploadFolderFlags(cmdFolder, &UploadFolderFlags)

	// TODO: provide the mapping between old and new flags

	// TODO: add missing options
	// -stack-burst
	// -stack-jpg-raw

	// TODO: Add Examples
	// immich-go upload from-folder --ban-file @eaDir/ --ban-file @__thumb/ --ban-file SYNOFILE_THUMB_*

	uploadCmd.AddCommand(cmdFolder)

	cmdFolder.RunE = func(cmd *cobra.Command, args []string) error {
		UpCmd := &UpCmd{
			Root:              rootFlags,
			Server:            cmdUpServerFlags,
			CommonFlags:       CommonFlags,
			UploadFolderFlags: &UploadFolderFlags,
		}

		ctx := cmd.Context()

		err := UpCmd.Root.Open(cmdFolder)
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

		err = UpCmd.Server.Open(UpCmd.Root)
		if err != nil {
			return err
		}
		UpCmd.UploadFolderFlags.SupportedMedia = UpCmd.Server.Immich.SupportedMedia()

		adapter, err := folder.NewLocalFiles(ctx, UpCmd.Jnl, UpCmd.UploadFolderFlags, fsyss...)
		if err != nil {
			return err
		}

		return UpCmd.run(ctx, adapter)
	}
}
