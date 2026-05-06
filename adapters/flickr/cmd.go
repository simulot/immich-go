package flickr

import (
	"context"
	"errors"
	"strings"

	"github.com/simulot/immich-go/adapters"
	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// NewFromFlickrCommand creates the "from-flickr" cobra subcommand and wires it into
// the upload pipeline via the provided runner.  It follows the same construction
// pattern as NewFromGooglePhotosCommand:
//   - cmd.SetContext is called before returning so context propagates correctly.
//   - processor is read inside RunE (after PersistentPreRunE has populated it).
//   - fsyss is populated and validated inside RunE; CloseFSs is deferred there.
func NewFromFlickrCommand(ctx context.Context, parent *cobra.Command, app *app.Application, runner adapters.Runner) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "from-flickr [flags] <metadata.zip> <images.zip>...",
		Short: "Upload photos from a Flickr export (metadata archive + image archives)",
		Args:  cobra.MinimumNArgs(1),
	}
	cmd.SetContext(ctx)

	f := &FlickrCmd{
		app: app,
	}
	f.RegisterFlags(cmd.Flags(), cmd)

	cmd.RunE = func(cmd *cobra.Command, args []string) error { //nolint:contextcheck
		var err error

		// processor is populated by PersistentPreRunE which runs before RunE.
		f.processor = app.FileProcessor()

		f.fsyss, err = fshelper.ParsePath(args)
		if err != nil {
			return err
		}
		if len(f.fsyss) == 0 {
			return errors.New("no file found matching the pattern: " + strings.Join(args, ","))
		}

		defer func() {
			if err := fshelper.CloseFSs(f.fsyss); err != nil {
				app.Log().Error("error closing file systems", "error", err)
			}
		}()

		return runner.Run(cmd, f)
	}

	return cmd
}

// RegisterFlags registers the Flickr-specific CLI flags onto the provided FlagSet.
// Only flags relevant to a Flickr export import are included; GP-specific flags
// (TakeoutTag, PeopleTag, StackOptions, InclusionFlags) are intentionally omitted.
func (f *FlickrCmd) RegisterFlags(flags *pflag.FlagSet, cmd *cobra.Command) {
	flags.BoolVar(&f.CreateAlbums, "sync-albums", true, "Automatically create albums in Immich that match the albums in your Flickr export")
}
