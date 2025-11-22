package folder

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func TestRegisterFlagsSetsAlbumModeNone(t *testing.T) {
	var ifc ImportFolderCmd
	cmd := &cobra.Command{Use: "from-folder"}
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)

	ifc.RegisterFlags(flags, cmd)

	if ifc.UsePathAsAlbumName != FolderModeNone {
		t.Fatalf("expected default album mode %q, got %q", FolderModeNone, ifc.UsePathAsAlbumName)
	}
}

func TestAlbumFolderModeSetAcceptsNone(t *testing.T) {
	var mode AlbumFolderMode
	if err := mode.Set("none"); err != nil {
		t.Fatalf("unexpected error setting mode: %v", err)
	}
	if mode != FolderModeNone {
		t.Fatalf("expected mode %q, got %q", FolderModeNone, mode)
	}
}
