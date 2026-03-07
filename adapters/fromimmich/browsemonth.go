package fromimmich

import (
	"context"
	"fmt"
	"sort"

	cliflags "github.com/simulot/immich-go/internal/cliFlags"

	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/assets"
)

// PreScan queries the source Immich server for all matching assets and returns
// a sorted list of YYYY-MM strings representing months that have assets.
// It uses a lightweight callback that only collects dates without fetching
// full asset details.
func (fic *FromImmichCmd) PreScan(ctx context.Context) ([]string, error) {
	fic.MinimalRating = min(max(0, fic.MinimalRating), 5)

	so := immich.SearchOptions()

	if !fic.OnlyArchived && !fic.OnlyTrashed && !fic.OnlyFavorite {
		so.All()
	} else {
		if fic.OnlyArchived {
			so.WithOnlyArchived()
		}
		if fic.OnlyTrashed {
			so.WithOnlyTrashed()
		}
		if fic.OnlyFavorite {
			so.WithOnlyFavorite()
		}
	}

	if fic.Make != "" {
		so.WithOnlyMake(fic.Make)
	}
	if fic.Model != "" {
		so.WithOnlyMake(fic.Model)
	}
	if fic.Country != "" {
		so.WithOnlyCountry(fic.Country)
	}
	if fic.State != "" {
		so.WithOnlyState(fic.State)
	}
	if fic.City != "" {
		so.WithOnlyCity(fic.City)
	}

	if fic.OnlyNoAlbum {
		so.WithNotInAlbum()
	} else {
		so.WithAlbums(fic.albumIDs...)
	}

	if fic.InclusionFlags.DateRange.IsSet() {
		so.WithDateRange(fic.InclusionFlags.DateRange)
	}

	if fic.MinimalRating > 1 {
		so.WithMinimalRate(fic.MinimalRating)
	}

	if len(fic.albumIDs) > 0 {
		so.WithAlbums(fic.albumIDs...)
	}

	if len(fic.tagIDs) > 0 {
		so.WithTags(fic.tagIDs...)
	}

	if len(fic.peopleIDs) > 0 {
		so.WithPeople(fic.peopleIDs...)
	}

	monthSet := make(map[string]struct{})
	err := fic.client.Immich.GetFilteredAssetsFn(ctx, so, func(a *immich.Asset) error {
		if !fic.IncludePartners && a.OwnerID != fic.client.User.ID {
			return nil
		}
		d := a.LocalDateTime.Time
		if d.IsZero() {
			d = a.FileCreatedAt.Time
		}
		if !d.IsZero() {
			month := fmt.Sprintf("%04d-%02d", d.Year(), d.Month())
			monthSet[month] = struct{}{}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	months := make([]string, 0, len(monthSet))
	for m := range monthSet {
		months = append(months, m)
	}
	sort.Strings(months)
	return months, nil
}

// BrowseMonth yields only assets whose date falls within the target month.
// It temporarily overrides the date range filter and delegates to Browse.
func (fic *FromImmichCmd) BrowseMonth(ctx context.Context, month string) chan *assets.Group {
	if month == "no-date" {
		// Immich assets always have dates from the server, return empty channel
		ch := make(chan *assets.Group)
		close(ch)
		return ch
	}

	// Save and override the date range using the month string (e.g. "2022-06")
	savedRange := fic.InclusionFlags.DateRange
	fic.InclusionFlags.DateRange = cliflags.InitDateRange(nil, month)

	gOut := make(chan *assets.Group)
	go func() {
		defer func() {
			close(gOut)
			fic.InclusionFlags.DateRange = savedRange
		}()
		err := fic.getAssets(ctx, gOut)
		if err != nil {
			fic.app.ProcessError(err) //nolint:errcheck
		}
	}()
	return gOut
}
