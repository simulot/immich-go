package burst

import (
	"context"
	"time"

	"github.com/simulot/immich-go/internal/assets"
	"golang.org/x/exp/constraints"
)

const frameInterval = 250 * time.Millisecond

// Group groups assets taken within a period of less than 1 second.
// The in channel receives assets sorted by date taken.
func Group(ctx context.Context, in <-chan *assets.Asset, out chan<- *assets.Asset, gOut chan<- *assets.Group) {
	var currentGroup []*assets.Asset
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
				currentGroup = []*assets.Asset{a}
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

func sendBurstGroup(ctx context.Context, out chan<- *assets.Asset, outg chan<- *assets.Group, as []*assets.Asset) {
	if len(as) < 2 {
		select {
		case out <- as[0]:
		case <-ctx.Done():
		}
		return
	}

	g := assets.NewGroup(assets.GroupByBurst, as...)
	g.CoverIndex = 0 // Assuming the first asset is the cover

	select {
	case <-ctx.Done():
		return
	case outg <- g:
	}
}
