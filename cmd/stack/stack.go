package stack

import (
	"context"
	"flag"
	"fmt"
	"sort"
	"strconv"

	"github.com/simulot/immich-go/cmd"
	"github.com/simulot/immich-go/helpers/stacking"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/ui"
)

type StackCmd struct {
	*cmd.RootImmichFlags
	AssumeYes bool
	DateRange immich.DateRange // Set capture date range
}

func initStack(ctx context.Context, common *cmd.RootImmichFlags, args []string) (*StackCmd, error) {
	cmd := flag.NewFlagSet("stack", flag.ExitOnError)
	validRange := immich.DateRange{}

	_ = validRange.Set("1850-01-04,2030-01-01")
	app := StackCmd{
		RootImmichFlags: common,
		DateRange:       validRange,
	}
	app.RootImmichFlags.SetFlags(cmd)
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
	err = app.RootImmichFlags.Start(ctx)
	if err != nil {
		return nil, err
	}
	return &app, err
}

func NewStackCommand(ctx context.Context, common *cmd.RootImmichFlags, args []string) error {
	app, err := initStack(ctx, common, args)
	if err != nil {
		return err
	}

	sb := stacking.NewStackBuilder(app.Immich.SupportedMedia())
	fmt.Println("Get server's assets...")
	assetCount := 0

	err = app.Immich.GetAllAssetsWithFilter(ctx, func(a *immich.Asset) error {
		if a.IsTrashed {
			return nil
		}
		if !app.DateRange.InRange(a.ExifInfo.DateTimeOriginal.Time) {
			return nil
		}
		assetCount += 1
		sb.ProcessAsset(a.ID, a.OriginalFileName, a.ExifInfo.DateTimeOriginal.Time)
		return nil
	})
	if err != nil {
		return err
	}
	stacks := sb.Stacks()
	app.Log.Info(fmt.Sprintf(" %d received, %d stack(s) possible\n", assetCount, len(stacks)))

	for _, s := range stacks {
		fmt.Printf("Stack following images taken on %s\n", s.Date)
		cover := s.CoverID
		names := s.Names
		sort.Strings(names)
		for _, n := range names {
			fmt.Printf("  %s\n", n)
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
				fmt.Printf("Can't stack images: %s\n", err)
			}
		}
	}

	return nil
}
