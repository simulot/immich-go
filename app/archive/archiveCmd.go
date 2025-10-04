package archive

import (
	"context"
	"errors"

	"github.com/simulot/immich-go/adapters/folder"
	gp "github.com/simulot/immich-go/adapters/googlePhotos"
	"github.com/simulot/immich-go/app"
	rootCmd "github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type ArchiveCmd struct {
	// CLI flags
	ArchivePath string `mapstructure:"write-to-folder" yaml:"write-to-folder" json:"write-to-folder" toml:"write-to-folder"`

	// internal state
	app  *app.Application
	jnl  *fileevent.Recorder
	dest *folder.LocalAssetWriter
}

func (ac *ArchiveCmd) RegisterFlags(flags pflag.FlagSet) {
	flags.StringVarP(&ac.ArchivePath, "write-to-folder", "w", "", "Path where to write the archive")
}

func NewArchiveCommand(ctx context.Context, app *rootCmd.Application) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "archive",
		Short: "Archive various sources of photos to a file system",
	}
	ac := &ArchiveCmd{
		app: app,
	}
	ac.RegisterFlags(*cmd.Flags())

	_ = cmd.MarkPersistentFlagRequired("write-to-folder")

	cmd.AddCommand(folder.NewFromFolderCommand(ctx, cmd, app, ac))
	cmd.AddCommand(folder.NewFromICloudCommand(ctx, cmd, app, ac))
	cmd.AddCommand(folder.NewFromPicasaCommand(ctx, cmd, app, ac))
	cmd.AddCommand(gp.NewFromGooglePhotosCommand(ctx, cmd, app, ac))

	cmd.RunE = func(cmd *cobra.Command, args []string) error { //nolint:contextcheck
		return errors.New("you must specify a subcommand to the archive command")
	}
	return cmd
}
