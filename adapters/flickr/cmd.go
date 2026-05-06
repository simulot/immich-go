package flickr

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
		Use:   "from-flickr [flags] <download-dir | zip-file>...",
		Short: "Upload photos from a Flickr export directory or ZIP files",
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

		// Expand any directory arguments into their constituent ZIPs before
		// ParsePath opens them. This lets users point at a download folder
		// instead of enumerating every ZIP file individually.
		args, err = expandDirArgs(args)
		if err != nil {
			return err
		}
		if len(args) == 0 {
			return errors.New("no ZIP files found in: " + strings.Join(args, ", "))
		}

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

// expandDirArgs replaces any directory path in args with the list of *.zip
// files found inside that directory. Non-directory paths pass through unchanged.
// This allows users to pass a download folder instead of listing every ZIP.
func expandDirArgs(args []string) ([]string, error) {
	out := make([]string, 0, len(args))
	for _, arg := range args {
		info, err := os.Stat(arg)
		if err != nil || !info.IsDir() {
			// Not a directory (or stat failed) — pass through as-is.
			out = append(out, arg)
			continue
		}
		// Directory: find all ZIPs inside.
		matches, err := filepath.Glob(filepath.Join(arg, "*.zip"))
		if err != nil {
			return nil, err
		}
		if len(matches) == 0 {
			return nil, fmt.Errorf("no ZIP files found in directory: %s", arg)
		}
		out = append(out, matches...)
	}
	return out, nil
}

// RegisterFlags registers the Flickr-specific CLI flags onto the provided FlagSet.
// Only flags relevant to a Flickr export import are included; GP-specific flags
// (TakeoutTag, PeopleTag, StackOptions, InclusionFlags) are intentionally omitted.
func (f *FlickrCmd) RegisterFlags(flags *pflag.FlagSet, cmd *cobra.Command) {
	flags.BoolVar(&f.CreateAlbums, "sync-albums", true, "Automatically create albums in Immich that match the albums in your Flickr export")
}
