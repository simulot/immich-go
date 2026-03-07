package adapters

import (
	"context"
	"fmt"
	"time"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/spf13/cobra"
)

type Reader interface {
	Browse(cxt context.Context) chan *assets.Group
}

// DateRangeProvider is an optional interface for adapters that support
// date-aware browsing (batched upload mode). If an adapter does not
// implement DateRangeProvider, the upload command falls back to the
// current behavior (full index, no batching).
type DateRangeProvider interface {
	PreScan(ctx context.Context) (months []string, err error)
	BrowseMonth(ctx context.Context, month string) chan *assets.Group
}

type AssetWriter interface {
	WriteAsset(context.Context, *assets.Asset) error
	// WriteGroup(ctx context.Context, group *assets.Group) error
}

type Runner interface {
	Run(cmd *cobra.Command, adapter Reader) error
}

// MonthToDateRange converts a "YYYY-MM" string to a date range with
// ±1 day padding for timezone safety.
// For example, "2022-06" returns:
//
//	after  = 2022-05-31 00:00:00 UTC
//	before = 2022-07-02 00:00:00 UTC
func MonthToDateRange(month string) (after, before time.Time, err error) {
	t, err := time.Parse("2006-01", month)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid month %q: %w", month, err)
	}
	after = t.AddDate(0, 0, -1)                // first of month minus 1 day
	before = t.AddDate(0, 1, 0).AddDate(0, 0, 1) // first of next month plus 1 day
	return after, before, nil
}
