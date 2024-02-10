package stack

import (
	"context"
	"flag"
	"path"
	"sort"
	"strconv"

	"github.com/simulot/immich-go/helpers/stacking"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/logger"
	"github.com/simulot/immich-go/ui"
)

type StackCmd struct {
	Immich *immich.ImmichClient // Immich client
	logger *logger.Log

	AssumeYes bool
	DateRange immich.DateRange // Set capture date range
}

func initSack(ic *immich.ImmichClient, log *logger.Log, args []string) (*StackCmd, error) {
	cmd := flag.NewFlagSet("stack", flag.ExitOnError)
	validRange := immich.DateRange{}

	_ = validRange.Set("1850-01-04,2030-01-01")
	app := StackCmd{
		logger:    log,
		Immich:    ic,
		DateRange: validRange,
	}

	cmd.BoolFunc("yes", "When true, assume Yes to all actions", func(s string) error {
		var err error
		app.AssumeYes, err = strconv.ParseBool(s)
		return err
	})
	cmd.Var(&app.DateRange, "date", "Process only documents having a capture date in that range.")
	err := cmd.Parse(args)
	return &app, err
}

func NewStackCommand(ctx context.Context, ic *immich.ImmichClient, log *logger.Log, args []string) error {
	app, err := initSack(ic, log, args)
	if err != nil {
		return err
	}

	sb := stacking.NewStackBuilder()
	log.MessageContinue(logger.OK, "Get server's assets...")
	assetCount := 0

	err = app.Immich.GetAllAssetsWithFilter(ctx, nil, func(a *immich.Asset) {
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
	log.MessageTerminate(logger.OK, " %d received, %d stack(s) possible", assetCount, len(stacks))

	for _, s := range stacks {
		log.OK("Stack following images taken on %s", s.Date)
		cover := s.CoverID
		names := s.Names
		sort.Strings(names)
		for _, n := range names {
			log.OK("  %s", n)
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
				log.Warning("Can't stack images: %s", err)
			}
		}
	}

	return nil
}
