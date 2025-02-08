package stack

import (
	"context"
	"sort"
	"time"

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
)

/*
TODO
- [X] dry-run mode
- [X] existing stack --> apparently correctly handled by the server
- [X] Take sub second exif time into account
*/
type StackCmd struct {
	DateRange cliflags.DateRange // Set capture date range

	// Stack jpg/raw
	StackJpgWithRaw bool

	// Stack burst
	StackBurstPhotos bool

	// SupportedMedia is the server's actual list of supported media types.
	SupportedMedia filetypes.SupportedMedia

	// InfoCollector is used to extract information from the file name.
	InfoCollector *filenames.InfoCollector

	// ManageHEICJPG determines whether to manage HEIC to JPG conversion options.
	ManageHEICJPG filters.HeicJpgFlag

	// ManageRawJPG determines how to manage raw and JPEG files.
	ManageRawJPG filters.RawJPGFlag

	// BurstFlag determines how to manage burst photos.
	ManageBurst filters.BurstFlag

	// ManageEpsonFastFoto enables the management of Epson FastFoto files.
	ManageEpsonFastFoto bool

	TZ *time.Location

	assets []*assets.Asset

	groupers []groups.Grouper // groups are used to group assets
	filters  []filters.Filter // filters are used to filter assets in groups
}

const timeFormat = "2006-01-02T15:04:05.000Z"

func NewStackCommand(ctx context.Context, a *app.Application) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stack [flags]",
		Short: "Update Immich for stacking related photos",
		Long:  `Stack photos related to each other according to the options`,
	}

	o := &StackCmd{}
	app.AddClientFlags(ctx, cmd, a, false)
	cmd.TraverseChildren = true
	cmd.Flags().Var(&o.ManageHEICJPG, "manage-heic-jpeg", "Manage coupled HEIC and JPEG files. Possible values: KeepHeic, KeepJPG, StackCoverHeic, StackCoverJPG")
	cmd.Flags().Var(&o.ManageRawJPG, "manage-raw-jpeg", "Manage coupled RAW and JPEG files. Possible values: KeepRaw, KeepJPG, StackCoverRaw, StackCoverJPG")
	cmd.Flags().Var(&o.ManageBurst, "manage-burst", "Manage burst photos. Possible values: Stack, StackKeepRaw, StackKeepJPEG")
	cmd.Flags().BoolVar(&o.ManageEpsonFastFoto, "manage-epson-fastfoto", false, "Manage Epson FastFoto file (default: false)")
	cmd.Flags().Var(&o.DateRange, "date-range", "photos must be taken in the date range")

	cmd.RunE = func(cmd *cobra.Command, args []string) error { //nolint:contextcheck
		// ready to run
		ctx := cmd.Context()
		client := a.Client()
		o.TZ = a.GetTZ()

		o.InfoCollector = filenames.NewInfoCollector(o.TZ, client.Immich.SupportedMedia())
		o.filters = append(o.filters,
			o.ManageBurst.GroupFilter(),
			o.ManageRawJPG.GroupFilter(),
			o.ManageHEICJPG.GroupFilter())

		if o.ManageEpsonFastFoto {
			o.groupers = append(o.groupers, epsonfastfoto.Group{}.Group)
		}
		if o.ManageBurst != filters.BurstNothing {
			o.groupers = append(o.groupers, burst.Group)
		}
		o.groupers = append(o.groupers, series.Group)

		query := &immich.SearchMetadataQuery{
			WithExif: true,
		}

		if o.DateRange.IsSet() {
			query.TakenAfter = o.DateRange.After.Format(timeFormat)
			query.TakenBefore = o.DateRange.Before.Format(timeFormat)
		}
		err := client.Immich.GetAllAssetsWithFilter(ctx, query,
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
			r1, r2 := s.assets[i].NameInfo.Radical, s.assets[j].NameInfo.Radical
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
				if err := app.Client().Immich.DeleteAssets(ctx, []string{r.Asset.ID}, false); err != nil {
					log.Error("can't delete asset %s: %s", r.Asset.OriginalFileName, err)
				} else {
					log.Info("Asset %s deleted: %s", r.Asset.OriginalFileName, r.Reason)
				}
			}
		}

		if len(g.Assets) > 1 && g.Grouping != assets.GroupByNone {
			client := app.Client().Immich.(immich.ImmichStackInterface)
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
