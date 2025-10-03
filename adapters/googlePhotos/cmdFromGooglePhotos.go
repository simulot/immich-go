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
	// CreateAlbums determines whether to create albums in Immich that match the albums in the Google Photos takeout.
	CreateAlbums bool `mapstructure:"sync-albums" yaml:"sync-albums" json:"sync-albums" toml:"sync-albums"`

	// ImportFromAlbum specifies the name of the Google Photos album to import from. If empty, all albums will be imported.
	ImportFromAlbum string `mapstructure:"from-album-name" yaml:"from-album-name" json:"from-album-name" toml:"from-album-name"`

	// ImportIntoAlbum specifies the name of the album to import assets into.
	ImportIntoAlbum string `mapstructure:"into-album" yaml:"into-album" json:"into-album" toml:"into-album"`

	// PartnerSharedAlbum specifies the name of the album to add partner's photos to.
	PartnerSharedAlbum string `mapstructure:"partner-shared-album" yaml:"partner-shared-album" json:"partner-shared-album" toml:"partner-shared-album"`

	// KeepTrashed determines whether to import photos that are marked as trashed in Google Photos.
	KeepTrashed bool `mapstructure:"include-trashed" yaml:"include-trashed" json:"include-trashed" toml:"include-trashed"`

	// KeepPartner determines whether to import photos from the partner's Google Photos account.
	KeepPartner bool `mapstructure:"include-partner" yaml:"include-partner" json:"include-partner" toml:"include-partner"`

	// KeepUntitled determines whether to include photos from albums without a title in the import process.
	KeepUntitled bool `mapstructure:"include-untitled-albums" yaml:"include-untitled-albums" json:"include-untitled-albums" toml:"include-untitled-albums"`

	// KeepArchived determines whether to import archived Google Photos.
	KeepArchived bool `mapstructure:"include-archived" yaml:"include-archived" json:"include-archived" toml:"include-archived"`

	// KeepJSONLess determines whether to import photos that do not have a matching JSON file in the takeout.
	KeepJSONLess bool `mapstructure:"include-unmatched" yaml:"include-unmatched" json:"include-unmatched" toml:"include-unmatched"`

	// Flags  for controlling the extensions of the files to be uploaded
	InclusionFlags cliflags.InclusionFlags `mapstructure:",squash" yaml:",inline" json:",inline" toml:",inline"`

	// List of banned files
	BannedFiles namematcher.List `mapstructure:"ban-file" yaml:"ban-file" json:"ban-file" toml:"ban-file"` // List of banned file name patterns

	// Add the takeout file name as tag
	TakeoutTag  bool `mapstructure:"takeout-tag" yaml:"takeout-tag" json:"takeout-tag" toml:"takeout-tag"`
	TakeoutName string

	// PeopleTag indicates whether to add a people tag to the imported assets.
	PeopleTag bool `mapstructure:"people-tag" yaml:"people-tag" json:"people-tag" toml:"people-tag"`

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

		err = runner.Run(cmd, adapter)
		return err
	}

	return cmd
}
