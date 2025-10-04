package gp

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/simulot/immich-go/adapters"
	"github.com/simulot/immich-go/adapters/shared"
	"github.com/simulot/immich-go/app"
	cliflags "github.com/simulot/immich-go/internal/cliFlags"
	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/filetypes"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/simulot/immich-go/internal/namematcher"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// ImportFlags represents the command-line flags for the Google Photos takeout import command.
type ImportFlags struct {
	// CLI FLags
	CreateAlbums       bool
	ImportFromAlbum    string
	ImportIntoAlbum    string
	PartnerSharedAlbum string
	KeepTrashed        bool
	KeepPartner        bool
	KeepUntitled       bool
	KeepArchived       bool
	KeepJSONLess       bool
	InclusionFlags     cliflags.InclusionFlags
	BannedFiles        namematcher.List
	TakeoutTag         bool
	TakeoutName        string
	PeopleTag          bool

	// internal state
	SupportedMedia filetypes.SupportedMedia
	InfoCollector  *filenames.InfoCollector
	TZ             *time.Location
	fsyss          []fs.FS
}

func (o *ImportFlags) RegisterFlags(flags *pflag.FlagSet, cmd *cobra.Command) {
	o.BannedFiles, _ = namematcher.New(shared.DefaultBannedFiles...)
	o.SupportedMedia = filetypes.DefaultSupportedMedia

	flags.BoolVar(&o.CreateAlbums, "sync-albums", true, "Automatically create albums in Immich that match the albums in your Google Photos takeout")
	flags.StringVar(&o.ImportFromAlbum, "from-album-name", "", "Only import photos from the specified Google Photos album")
	flags.BoolVar(&o.KeepUntitled, "include-untitled-albums", false, "Include photos from albums without a title in the import process")
	flags.BoolVarP(&o.KeepTrashed, "include-trashed", "t", false, "Import photos that are marked as trashed in Google Photos")
	flags.BoolVarP(&o.KeepPartner, "include-partner", "p", true, "Import photos from your partner's Google Photos account")
	flags.StringVar(&o.PartnerSharedAlbum, "partner-shared-album", "", "Add partner's photo to the specified album name")
	flags.BoolVarP(&o.KeepArchived, "include-archived", "a", true, "Import archived Google Photos")
	flags.BoolVarP(&o.KeepJSONLess, "include-unmatched", "u", false, "Import photos that do not have a matching JSON file in the takeout")
	flags.Var(&o.BannedFiles, "ban-file", "Exclude a file based on a pattern (case-insensitive). Can be specified multiple times.")
	flags.BoolVar(&o.TakeoutTag, "takeout-tag", true, "Tag uploaded photos with a tag \"{takeout}/takeout-YYYYMMDDTHHMMSSZ\"")
	flags.BoolVar(&o.PeopleTag, "people-tag", true, "Tag uploaded photos with tags \"people/name\" found in the JSON file")

	o.InclusionFlags.RegisterFlags(flags, "")
}

var _re3digits = regexp.MustCompile(`-\d{3}$`)

func NewFromGooglePhotosCommand(ctx context.Context, parent *cobra.Command, app *app.Application, runner adapters.Runner) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "from-google-photos [flags] <takeout-*.zip> | <takeout-folder>",
		Short: "Upload photos either from a zipped Google Photos takeout or decompressed archive",
		Args:  cobra.MinimumNArgs(1),
	}
	cmd.SetContext(ctx)
	opt := &ImportFlags{}
	opt.RegisterFlags(cmd.Flags(), cmd)

	cmd.RunE = func(cmd *cobra.Command, args []string) error { //nolint:contextcheck
		ctx := cmd.Context()
		log := app.Log()

		opt.TZ = app.GetTZ()

		fsyss, err := fshelper.ParsePath(args)
		if err != nil {
			return err
		}
		if len(fsyss) == 0 {
			log.Message("No file found matching the pattern: %s", strings.Join(args, ","))
			return errors.New("No file found matching the pattern: " + strings.Join(args, ","))
		}

		defer fshelper.CloseFSs(fsyss)

		if opt.TakeoutTag {
			for _, fsys := range fsyss {
				if fsys, ok := fsys.(fshelper.NameFS); ok {
					opt.TakeoutName = fsys.Name()
					break
				}
			}

			if filepath.Ext(opt.TakeoutName) == ".zip" {
				opt.TakeoutName = strings.TrimSuffix(opt.TakeoutName, filepath.Base(opt.TakeoutName))
			}
			if opt.TakeoutName == "" {
				opt.TakeoutTag = false
			}
			opt.TakeoutName = _re3digits.ReplaceAllString(opt.TakeoutName, "")
		}

		adapter, err := NewTakeout(ctx, app.Jnl(), opt, fsyss...)
		if err != nil {
			return err
		}

		// callback the caller
		err = runner.Run(cmd, adapter)
		return err
	}

	return cmd
}
