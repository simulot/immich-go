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
	"github.com/simulot/immich-go/internal/assets"
	cliflags "github.com/simulot/immich-go/internal/cliFlags"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/filetypes"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/simulot/immich-go/internal/gen"
	"github.com/simulot/immich-go/internal/groups"
	"github.com/simulot/immich-go/internal/namematcher"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// ImportFlags represents the command-line flags for the Google Photos takeout import command.
type TakeoutCmd struct {
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
	catalogs       map[string]directoryCatalog                // file catalogs by directory in the set of the all takeout parts
	albums         map[string]assets.Album                    // track album names by folder
	fileTracker    *gen.SyncMap[fileKeyTracker, trackingInfo] // map[fileKeyTracker]trackingInfo // key is base name + file size,  value is list of file paths
	log            *fileevent.Recorder
	groupers       []groups.Grouper
}

func (toC *TakeoutCmd) RegisterFlags(flags *pflag.FlagSet, cmd *cobra.Command) {
	toC.BannedFiles, _ = namematcher.New(shared.DefaultBannedFiles...)
	toC.SupportedMedia = filetypes.DefaultSupportedMedia

	flags.BoolVar(&toC.CreateAlbums, "sync-albums", true, "Automatically create albums in Immich that match the albums in your Google Photos takeout")
	flags.StringVar(&toC.ImportFromAlbum, "from-album-name", "", "Only import photos from the specified Google Photos album")
	flags.BoolVar(&toC.KeepUntitled, "include-untitled-albums", false, "Include photos from albums without a title in the import process")
	flags.BoolVarP(&toC.KeepTrashed, "include-trashed", "t", false, "Import photos that are marked as trashed in Google Photos")
	flags.BoolVarP(&toC.KeepPartner, "include-partner", "p", true, "Import photos from your partner's Google Photos account")
	flags.StringVar(&toC.PartnerSharedAlbum, "partner-shared-album", "", "Add partner's photo to the specified album name")
	flags.BoolVarP(&toC.KeepArchived, "include-archived", "a", true, "Import archived Google Photos")
	flags.BoolVarP(&toC.KeepJSONLess, "include-unmatched", "u", false, "Import photos that do not have a matching JSON file in the takeout")
	flags.Var(&toC.BannedFiles, "ban-file", "Exclude a file based on a pattern (case-insensitive). Can be specified multiple times.")
	flags.BoolVar(&toC.TakeoutTag, "takeout-tag", true, "Tag uploaded photos with a tag \"{takeout}/takeout-YYYYMMDDTHHMMSSZ\"")
	flags.BoolVar(&toC.PeopleTag, "people-tag", true, "Tag uploaded photos with tags \"people/name\" found in the JSON file")

	toC.InclusionFlags.RegisterFlags(flags, "")
}

var _re3digits = regexp.MustCompile(`-\d{3}$`)

func NewFromGooglePhotosCommand(ctx context.Context, parent *cobra.Command, app *app.Application, runner adapters.Runner) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "from-google-photos [flags] <takeout-*.zip> | <takeout-folder>",
		Short: "Upload photos either from a zipped Google Photos takeout or decompressed archive",
		Args:  cobra.MinimumNArgs(1),
	}
	cmd.SetContext(ctx)
	toC := &TakeoutCmd{
		catalogs:    map[string]directoryCatalog{},
		albums:      map[string]assets.Album{},
		fileTracker: gen.NewSyncMap[fileKeyTracker, trackingInfo](), // map[fileKeyTracker]trackingInfo{},
	}
	toC.RegisterFlags(cmd.Flags(), cmd)

	cmd.RunE = func(cmd *cobra.Command, args []string) error { //nolint:contextcheck
		log := app.Log()
		toC.TZ = app.GetTZ()

		fsyss, err := fshelper.ParsePath(args)
		if err != nil {
			return err
		}
		if len(fsyss) == 0 {
			log.Message("No file found matching the pattern: %s", strings.Join(args, ","))
			return errors.New("No file found matching the pattern: " + strings.Join(args, ","))
		}

		defer fshelper.CloseFSs(fsyss)

		if toC.TakeoutTag {
			for _, fsys := range fsyss {
				if fsys, ok := fsys.(fshelper.NameFS); ok {
					toC.TakeoutName = fsys.Name()
					break
				}
			}

			if filepath.Ext(toC.TakeoutName) == ".zip" {
				toC.TakeoutName = strings.TrimSuffix(toC.TakeoutName, filepath.Base(toC.TakeoutName))
			}
			if toC.TakeoutName == "" {
				toC.TakeoutTag = false
			}
			toC.TakeoutName = _re3digits.ReplaceAllString(toC.TakeoutName, "")
		}

		toC.InfoCollector = filenames.NewInfoCollector(toC.TZ, toC.SupportedMedia)

		// callback the caller
		return runner.Run(cmd, toC)
	}

	return cmd
}
