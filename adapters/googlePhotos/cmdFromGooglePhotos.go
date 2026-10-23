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
	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/fileprocessor"
	"github.com/simulot/immich-go/internal/filetypes"
	"github.com/simulot/immich-go/internal/filters"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/simulot/immich-go/internal/gen"
	"github.com/simulot/immich-go/internal/groups"
	"github.com/simulot/immich-go/internal/groups/burst"
	"github.com/simulot/immich-go/internal/groups/epsonfastfoto"
	"github.com/simulot/immich-go/internal/groups/series"
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
	KeepFailedVideos   bool
	InclusionFlags     cliflags.InclusionFlags
	BannedFiles        namematcher.List
	TakeoutTag         bool
	TakeoutName        string
	PeopleTag          bool
	shared.StackOptions

	// internal state
	app            *app.Application
	processor      *fileprocessor.FileProcessor
	supportedMedia filetypes.SupportedMedia
	infoCollector  *filenames.InfoCollector
	tz             *time.Location
	fsyss          []fs.FS
	catalogs       map[string]directoryCatalog                // file catalogs by directory in the set of the all takeout parts
	albums         map[string]assets.Album                    // track album names by folder
	fileTracker    *gen.SyncMap[fileKeyTracker, trackingInfo] // map[fileKeyTracker]trackingInfo // key is base name + file size,  value is list of file paths
	groupers       []groups.Grouper
	// filters        []filters.Filter
}

func (toc *TakeoutCmd) RegisterFlags(flags *pflag.FlagSet, cmd *cobra.Command) {
	toc.BannedFiles, _ = namematcher.New(shared.DefaultBannedFiles...)
	toc.supportedMedia = filetypes.DefaultSupportedMedia

	flags.BoolVar(&toc.CreateAlbums, "sync-albums", true, "Automatically create albums in Immich that match the albums in your Google Photos takeout")
	flags.StringVar(&toc.ImportFromAlbum, "from-album-name", "", "Only import photos from the specified Google Photos album")
	flags.BoolVar(&toc.KeepUntitled, "include-untitled-albums", false, "Include photos from albums without a title in the import process")
	flags.BoolVarP(&toc.KeepTrashed, "include-trashed", "t", false, "Import photos that are marked as trashed in Google Photos")
	flags.BoolVarP(&toc.KeepPartner, "include-partner", "p", true, "Import photos from your partner's Google Photos account")
	flags.StringVar(&toc.PartnerSharedAlbum, "partner-shared-album", "", "Add partner's photo to the specified album name")
	flags.BoolVarP(&toc.KeepArchived, "include-archived", "a", true, "Import archived Google Photos")
	flags.BoolVarP(&toc.KeepJSONLess, "include-unmatched", "u", false, "Import photos that do not have a matching JSON file in the takeout")
	flags.BoolVar(&toc.KeepFailedVideos, "include-failed-videos", false, "Import videos that have been marked by Google Photos as failed")
	flags.Var(&toc.BannedFiles, "ban-file", "Exclude a file based on a pattern (case-insensitive). Can be specified multiple times.")
	flags.BoolVar(&toc.TakeoutTag, "takeout-tag", true, "Tag uploaded photos with a tag \"{takeout}/takeout-YYYYMMDDTHHMMSSZ\"")
	flags.BoolVar(&toc.PeopleTag, "people-tag", true, "Tag uploaded photos with tags \"people/name\" found in the JSON file")
	if cmd.Parent() != nil && cmd.Parent().Name() == "upload" {
		toc.StackOptions.RegisterFlags(flags)
	}

	toc.InclusionFlags.RegisterFlags(flags, "")
}

var _re3digits = regexp.MustCompile(`-\d{3}$`)

func NewFromGooglePhotosCommand(ctx context.Context, parent *cobra.Command, app *app.Application, runner adapters.Runner) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "from-google-photos [flags] <takeout-*.zip> | <takeout-folder>",
		Short: "Upload photos either from a zipped Google Photos takeout or decompressed archive",
		Args:  cobra.MinimumNArgs(1),
	}
	cmd.SetContext(ctx)
	toc := &TakeoutCmd{
		app:         app,
		catalogs:    map[string]directoryCatalog{},
		albums:      map[string]assets.Album{},
		fileTracker: gen.NewSyncMap[fileKeyTracker, trackingInfo](), // map[fileKeyTracker]trackingInfo{},
	}
	toc.RegisterFlags(cmd.Flags(), cmd)

	cmd.RunE = func(cmd *cobra.Command, args []string) error { //nolint:contextcheck
		var err error

		log := app.Log()
		toc.processor = app.FileProcessor()
		toc.tz = app.GetTZ()

		// make an fs.FS per zip file or folder given on the CLI
		toc.fsyss, err = fshelper.ParsePath(args)
		if err != nil {
			return err
		}
		if len(toc.fsyss) == 0 {
			log.Message("No file found matching the pattern: %s", strings.Join(args, ","))
			return errors.New("No file found matching the pattern: " + strings.Join(args, ","))
		}

		defer func() {
			if err := fshelper.CloseFSs(toc.fsyss); err != nil {
				// Handle the error - log it, since we can't return it
				log.Error("error closing file systems", "error", err)
			}
		}()

		if toc.TakeoutTag {
			for _, fsys := range toc.fsyss {
				if fsys, ok := fsys.(fshelper.NameFS); ok {
					toc.TakeoutName = fsys.Name()
					break
				}
			}

			if filepath.Ext(toc.TakeoutName) == ".zip" {
				toc.TakeoutName = strings.TrimSuffix(toc.TakeoutName, filepath.Base(toc.TakeoutName))
			}
			if toc.TakeoutName == "" {
				toc.TakeoutTag = false
			}
			toc.TakeoutName = _re3digits.ReplaceAllString(toc.TakeoutName, "")
		}
		if toc.ManageEpsonFastFoto {
			g := epsonfastfoto.Group{}
			toc.groupers = append(toc.groupers, g.Group)
		}
		if toc.ManageBurst != filters.BurstNothing {
			toc.groupers = append(toc.groupers, burst.Group)
		}
		toc.groupers = append(toc.groupers, series.Group)

		// toc.filters = append(toc.filters, toc.StackOptions.ManageBurst.GroupFilter(), toc.StackOptions.ManageRawJPG.GroupFilter(), toc.StackOptions.ManageHEICJPG.GroupFilter())

		toc.supportedMedia = toc.app.GetSupportedMedia()
		toc.infoCollector = filenames.NewInfoCollector(toc.tz, toc.supportedMedia)

		// callback the caller
		return runner.Run(cmd, toc)
	}

	return cmd
}
