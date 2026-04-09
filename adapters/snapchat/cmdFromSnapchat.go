package snapchat

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
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
	"github.com/simulot/immich-go/internal/groups"
	"github.com/simulot/immich-go/internal/groups/burst"
	"github.com/simulot/immich-go/internal/groups/epsonfastfoto"
	"github.com/simulot/immich-go/internal/groups/series"
	"github.com/simulot/immich-go/internal/namematcher"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type Command struct {
	InclusionFlags cliflags.InclusionFlags
	BannedFiles    namematcher.List
	shared.StackOptions

	app            *app.Application
	processor      *fileprocessor.FileProcessor
	tz             *time.Location
	infoCollector  *filenames.InfoCollector
	supportedMedia filetypes.SupportedMedia
	fsyss          []fs.FS
	groupers       []groups.Grouper
}

func (sc *Command) RegisterFlags(flags *pflag.FlagSet, cmd *cobra.Command) {
	sc.BannedFiles, _ = namematcher.New(shared.DefaultBannedFiles...)
	flags.Var(&sc.BannedFiles, "ban-file", "Exclude a file based on a pattern (case-insensitive). Can be specified multiple times.")
	sc.InclusionFlags.RegisterFlags(flags, "")
	if cmd.Parent() != nil && cmd.Parent().Name() == "upload" {
		sc.StackOptions.RegisterFlags(flags)
	}
}

func NewFromSnapchatCommand(ctx context.Context, _ *cobra.Command, app *app.Application, runner adapters.Runner) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "from-snapchat [flags] <mydata-*.zip> | <takeout-folder>",
		Short: "Upload photos and videos from a Snapchat memories export",
		Args:  cobra.MinimumNArgs(1),
	}
	cmd.SetContext(ctx)

	sc := &Command{app: app}
	sc.RegisterFlags(cmd.Flags(), cmd)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		var err error

		sc.processor = app.FileProcessor()
		sc.tz = app.GetTZ()
		sc.supportedMedia = app.GetSupportedMedia()

		sc.fsyss, err = fshelper.ParsePath(args)
		if err != nil {
			return err
		}
		if len(sc.fsyss) == 0 {
			app.Log().Message("No file found matching the pattern: %s", strings.Join(args, ","))
			return errors.New("No file found matching the pattern: " + strings.Join(args, ","))
		}

		defer func() {
			if err := fshelper.CloseFSs(sc.fsyss); err != nil {
				app.Log().Error("error closing file systems", "error", err)
			}
		}()

		if sc.InclusionFlags.DateRange.IsSet() {
			sc.InclusionFlags.DateRange.SetTZ(sc.tz)
		}

		if sc.ManageEpsonFastFoto {
			sc.groupers = append(sc.groupers, epsonfastfoto.Group{}.Group)
		}
		if sc.ManageBurst != filters.BurstNothing {
			sc.groupers = append(sc.groupers, burst.Group)
		}
		sc.groupers = append(sc.groupers, series.Group)

		sc.infoCollector = filenames.NewInfoCollector(sc.tz, sc.supportedMedia)

		return runner.Run(cmd, sc)
	}

	return cmd
}

func (sc *Command) makeAssetFromMediaFile(file mediaFile) *assets.Asset {
	a := &assets.Asset{
		File:             fshelper.FSName(file.fsys, file.name),
		OriginalFileName: file.base,
		FileDate:         file.modTime,
		FileSize:         int(file.size),
	}
	a.SetNameInfo(sc.infoCollector.GetInfo(a.OriginalFileName))
	if file.metadata != nil {
		a.FromApplication = a.UseMetadata(file.metadata)
	}
	if file.overlay != nil {
		ext := strings.ToLower(filepath.Ext(a.OriginalFileName))
		if file.mainType() == mediaTypeImage && ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
			a.OriginalFileName = strings.TrimSuffix(a.OriginalFileName, filepath.Ext(a.OriginalFileName)) + ".jpg"
		}
		a.File = fshelper.FSName(newMergedFS(file, sc.app), mergedAssetName)
	}
	return a
}
