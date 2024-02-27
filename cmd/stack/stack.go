package stack

import (
	"context"
	"flag"
	"path"
	"sort"
	"strconv"

	"github.com/simulot/immich-go/cmd"
	"github.com/simulot/immich-go/helpers/stacking"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/logger"
	"github.com/simulot/immich-go/ui"
)

type StackCmd struct {
	*cmd.SharedFlags
	AssumeYes bool
	DateRange immich.DateRange // Set capture date range
}

func initSack(ctx context.Context, common *cmd.SharedFlags, args []string) (*StackCmd, error) {
	cmd := flag.NewFlagSet("stack", flag.ExitOnError)
	validRange := immich.DateRange{}

	_ = validRange.Set("1850-01-04,2030-01-01")
	app := StackCmd{
		SharedFlags: common,
		DateRange:   validRange,
	}
	app.SharedFlags.SetFlags(cmd)
	cmd.BoolFunc("yes", "When true, assume Yes to all actions", func(s string) error {
		var err error
		app.AssumeYes, err = strconv.ParseBool(s)
		return err
	})
	cmd.Var(&app.DateRange, "date", "Process only documents having a capture date in that range.")
	err := cmd.Parse(args)
	if err != nil {
		return nil, err
	}
	err = app.SharedFlags.Start(ctx)
	if err != nil {
		return nil, err
	}
	return &app, err
}

func NewStackCommand(ctx context.Context, common *cmd.SharedFlags, args []string) error {
	app, err := initSack(ctx, common, args)
	if err != nil {
		return err
	}

	sb := stacking.NewStackBuilder(app.Immich.SupportedMedia())
	app.Jnl.Log.MessageContinue(logger.OK, "Get server's assets...")
	assetCount := 0

	err = app.Immich.GetAllAssetsWithFilter(ctx, func(a *immich.Asset) {
		if a.IsTrashed {
			return
		}
		if !app.DateRange.InRange(a.ExifInfo.DateTimeOriginal.Time) {
			return
		}
		assetCount += 1
		sb.ProcessAsset(a.ID, a.OriginalFileName+path.Ext(a.OriginalPath), a.ExifInfo.DateTimeOriginal.Time)
	})
	if err != nil {
		return err
	}
	stacks := sb.Stacks()
	app.Jnl.Log.MessageTerminate(logger.OK, " %d received, %d stack(s) possible", assetCount, len(stacks))

	for _, s := range stacks {
		app.Jnl.Log.OK("Stack following images taken on %s", s.Date)
		cover := s.CoverID
		names := s.Names
		sort.Strings(names)
		for _, n := range names {
			app.Jnl.Log.OK("  %s", n)
		}
		yes := app.AssumeYes
		if !app.AssumeYes {
			r, err := ui.ConfirmYesNo(ctx, "Proceed?", "n")
			if err != nil {
				return err
			}
			if r == "y" {
				yes = true
			}
		}
		if yes {
			err := app.Immich.StackAssets(ctx, cover, s.IDs)
			if err != nil {
				app.Jnl.Log.Warning("Can't stack images: %s", err)
			}
		}
	}

	return nil
}
