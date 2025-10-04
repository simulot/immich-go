package stack

import (
	"context"
	"sort"
	"time"

	"github.com/simulot/immich-go/adapters/shared"
	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/assets"
	cliflags "github.com/simulot/immich-go/internal/cliFlags"
	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/filetypes"
	"github.com/simulot/immich-go/internal/filters"
	"github.com/simulot/immich-go/internal/groups"
	"github.com/simulot/immich-go/internal/groups/burst"
	"github.com/simulot/immich-go/internal/groups/epsonfastfoto"
	"github.com/simulot/immich-go/internal/groups/series"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

/*
TODO
- [X] dry-run mode
- [X] existing stack --> apparently correctly handled by the server
- [X] Take sub second exif time into account
*/
type StackCmd struct {
	// CLI flags
	StackOptions shared.StackOptions
	DateRange    cliflags.DateRange

	// internal state
	SupportedMedia filetypes.SupportedMedia
	InfoCollector  *filenames.InfoCollector
	TZ             *time.Location
	assets         []*assets.Asset
	client         app.Client
	groupers       []groups.Grouper // groups are used to group assets
	filters        []filters.Filter // filters are used to filter assets in groups
}

func (sc *StackCmd) RegisterFlags(flags *pflag.FlagSet) {
	sc.StackOptions.RegisterFlags(flags)
	flags.Var(&sc.DateRange, "date-range", "photos must be taken in the date range")
}

// const timeFormat = "2006-01-02T15:04:05.000Z"

func NewStackCommand(ctx context.Context, a *app.Application) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stack [flags]",
		Short: "Update Immich for stacking related photos",
		Long:  `Stack photos related to each other according to the options`,
	}

	o := &StackCmd{}
	o.RegisterFlags(cmd.Flags())
	o.client.RegisterFlags(cmd.Flags(), "")
	cmd.TraverseChildren = true

	cmd.RunE = func(cmd *cobra.Command, args []string) error { //nolint:contextcheck
		// ready to run
		ctx := cmd.Context()
		err := o.client.Open(ctx, a)
		if err != nil {
			return err
		}
		o.TZ = a.GetTZ()
		o.DateRange.SetTZ(a.GetTZ())

		o.InfoCollector = filenames.NewInfoCollector(o.TZ, o.client.Immich.SupportedMedia())
		o.filters = append(o.filters,
			o.StackOptions.ManageBurst.GroupFilter(),
			o.StackOptions.ManageRawJPG.GroupFilter(),
			o.StackOptions.ManageHEICJPG.GroupFilter())

		if o.StackOptions.ManageEpsonFastFoto {
			o.groupers = append(o.groupers, epsonfastfoto.Group{}.Group)
		}
		if o.StackOptions.ManageBurst != filters.BurstNothing {
			o.groupers = append(o.groupers, burst.Group)
		}
		o.groupers = append(o.groupers, series.Group)

		so := immich.SearchOptions().WithExif().WithDateRange(o.DateRange)

		err = o.client.Immich.GetFilteredAssetsFn(ctx, so,
			func(a *immich.Asset) error {
				if a.IsTrashed {
					return nil
				}

				asset := a.AsAsset()
				asset.SetNameInfo(o.InfoCollector.GetInfo(asset.OriginalFileName))
				asset.FromApplication = &assets.Metadata{
					FileName:    a.OriginalFileName,
					Latitude:    a.ExifInfo.Latitude,
					Longitude:   a.ExifInfo.Longitude,
					Description: a.ExifInfo.Description,
					DateTaken:   a.ExifInfo.DateTimeOriginal.Time,
					Trashed:     a.IsTrashed,
					Archived:    a.IsArchived,
					Favorited:   a.IsFavorite,
					Rating:      byte(a.Rating),
					Tags:        asset.Tags,
				}

				o.assets = append(o.assets, asset)
				return nil
			})
		if err != nil {
			return err
		}
		err = o.ProcessAssets(ctx, a)
		return err
	}
	return cmd
}

func (s *StackCmd) ProcessAssets(ctx context.Context, app *app.Application) error {
	log := app.Log()

	in := make(chan *assets.Asset)

	go func() {
		defer close(in)
		// Sort assets by radical, then date
		sort.Slice(s.assets, func(i, j int) bool {
			r1, r2 := s.assets[i].Radical, s.assets[j].Radical
			if r1 != r2 {
				return r1 < r2
			}
			return s.assets[i].CaptureDate.Before(s.assets[j].CaptureDate)
		})
		for _, a := range s.assets {
			select {
			case in <- a:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Group assets
	gChan := groups.NewGrouperPipeline(ctx, s.groupers...).PipeGrouper(ctx, in)

	for g := range gChan {
		g = filters.ApplyFilters(g, s.filters...)
		// Delete filtered assets
		if len(g.Removed) > 0 {
			for _, r := range g.Removed {
				if err := s.client.Immich.DeleteAssets(ctx, []string{r.Asset.ID}, false); err != nil {
					log.Error("can't delete asset %s: %s", r.Asset.OriginalFileName, err)
				} else {
					log.Info("Asset %s deleted: %s", r.Asset.OriginalFileName, r.Reason)
				}
			}
		}

		if len(g.Assets) > 1 && g.Grouping != assets.GroupByNone {
			client := s.client.Immich.(immich.ImmichStackInterface)
			ids := []string{g.Assets[g.CoverIndex].ID}
			for _, a := range g.Assets {
				log.Info("Stacking", "file", a.OriginalFileName)
				if a.ID != ids[0] {
					ids = append(ids, a.ID)
				}
			}
			if len(ids) > 1 {
				if _, err := client.CreateStack(ctx, ids); err != nil {
					log.Error("Can't create stack", "error", err)
				}
			}
		}
	}
	return nil
}

// 	gChan := make(chan *assets.Group)
// 	go func() {
// 		defer close(gChan)
// 		g := assets.NewGroup()
// 		for _, a := range s.assets {
// 			if !g.Add(a) {
// 				gChan <- g
// 				g = assets.NewGroup()
// 				g.Add(a)
// 			}
// 		}
// 		gChan <- g
// 	}
// 	gs := groups.NewGrouperPipeline(ctx, la.groupers...).PipeGrouper(ctx, in)
// 	g = filters.ApplyFilters(g, upCmd.UploadOptions.Filters...)

// filters := 	append( []filters.Filter,)
