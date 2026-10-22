package shared

import (
	"testing"

	"github.com/simulot/immich-go/internal/filters"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// A sub-command of upload reads every --manage-* value given to upload.
func TestReadFlagsTakesUploadsValues(t *testing.T) {
	var up StackOptions
	upload := &cobra.Command{Use: "upload"}
	up.RegisterFlags(upload.PersistentFlags())
	var got StackOptions
	sub := &cobra.Command{Use: "from-folder", RunE: func(cmd *cobra.Command, _ []string) error {
		return got.ReadFlags(cmd.Flags())
	}}
	upload.AddCommand(sub)
	upload.SetArgs([]string{
		"from-folder",
		"--manage-heic-jpeg=StackCoverHeic", "--manage-raw-jpeg=KeepRaw",
		"--manage-burst=StackKeepJPEG", "--manage-epson-fastfoto",
	})
	if err := upload.Execute(); err != nil {
		t.Fatal(err)
	}
	if got.ManageHEICJPG != filters.HeicJpgStackHeic || got.ManageRawJPG != filters.RawJPGKeepRaw ||
		got.ManageBurst != filters.BurstKeepJPEG || !got.ManageEpsonFastFoto {
		t.Errorf("got %+v", got)
	}
}

// Without the flags (archive registers none), the options stay as they are.
func TestReadFlagsWithoutTheFlags(t *testing.T) {
	so := StackOptions{ManageBurst: filters.BurstStack, ManageEpsonFastFoto: true}
	if err := so.ReadFlags(pflag.NewFlagSet("archive", pflag.ContinueOnError)); err != nil {
		t.Fatal(err)
	}
	if so.ManageBurst != filters.BurstStack || !so.ManageEpsonFastFoto {
		t.Errorf("got %+v", so)
	}
}
