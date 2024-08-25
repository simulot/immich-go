package stack

import (
	"fmt"
	"sort"
	"time"

	"github.com/simulot/immich-go/cmd"
	"github.com/simulot/immich-go/helpers/stacking"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/ui"
	"github.com/spf13/cobra"
)

type StackCmd struct {
	Command *cobra.Command
	*cmd.RootImmichFlags
	cmd.ImmichServerFlags
	AssumeYes bool
	DateRange immich.DateRange // Set capture date range
}

func AddCommand(root *cmd.RootImmichFlags) {
	stackCmd := StackCmd{
		RootImmichFlags: root,
		Command: &cobra.Command{
			Use:   "stack",
			Short: "Stack photos",
			Long:  `Stack photos taken in the short periode of time.`,
		},
	}
	now := time.Now().Add(24 * time.Hour)

	_ = stackCmd.DateRange.Set(time.Date(now.Year()-10, 1, 1, 0, 0, 0, 0, time.Local).Format("2006-01-02") + "," + now.Format("2006-01-02"))
	stackCmd.Command.RunE = stackCmd.run
	stackCmd.Command.Flags().Var(&stackCmd.DateRange, "date-range", "photos must be in the date range")
	stackCmd.Command.Flags().Bool("force-yes", false, "Assume YES to all questions")

	cmd.NewImmichServerFlagSet(stackCmd.Command, &stackCmd.ImmichServerFlags)
	root.Command.AddCommand(stackCmd.Command)
}

func (app *StackCmd) run(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	err := app.RootImmichFlags.Initialize()
	if err != nil {
		return err
	}
	err = app.ImmichServerFlags.Initialize(app.RootImmichFlags)
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
