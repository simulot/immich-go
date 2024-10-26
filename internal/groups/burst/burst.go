package burst

import (
	"context"
	"time"

	"github.com/simulot/immich-go/internal/groups"
	"github.com/simulot/immich-go/internal/groups/groupby"
	"golang.org/x/exp/constraints"
)

const frameInterval = 250 * time.Millisecond

// Group groups assets taken within a period of less than 1 second.
// The in channel receives assets sorted by date taken.
func Group(ctx context.Context, in <-chan groups.Asset, out chan<- groups.Asset, gOut chan<- *groups.AssetGroup) {
	var currentGroup []groups.Asset
	var lastTimestamp time.Time

	for {
		select {
		case <-ctx.Done():
			return
		case a, ok := <-in:
			if !ok {
				if len(currentGroup) > 0 {
					sendBurstGroup(ctx, out, gOut, currentGroup)
				}
				return
			}
			if a.DateTaken().IsZero() {
				select {
				case out <- a:
				case <-ctx.Done():
				}
				continue
			}

			if len(currentGroup) == 0 || abs(a.DateTaken().Sub(lastTimestamp)) < frameInterval {
				currentGroup = append(currentGroup, a)
				lastTimestamp = a.DateTaken()
			} else {
				sendBurstGroup(ctx, out, gOut, currentGroup)
				currentGroup = []groups.Asset{a}
				lastTimestamp = a.DateTaken()
			}
		}
	}
}

// abs returns the absolute value of a given integer.
func abs[T constraints.Integer](x T) T {
	if x < 0 {
		return -x
	}
	return x
}

func sendBurstGroup(ctx context.Context, out chan<- groups.Asset, outg chan<- *groups.AssetGroup, assets []groups.Asset) {
	if len(assets) < 2 {
		select {
		case out <- assets[0]:
		case <-ctx.Done():
		}
		return
	}

	g := groups.NewAssetGroup(groupby.GroupByBurst, assets...)
	g.CoverIndex = 0 // Assuming the first asset is the cover

	select {
	case <-ctx.Done():
		return
	case outg <- g:
	}
}
