package burst

import (
	"context"
	"time"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/filetypes"
	"golang.org/x/exp/constraints"
)

const frameInterval = 900 * time.Millisecond

// Group groups photos taken within a period of less than 1 second with a digital camera.
// This addresses photos taken with a digital camera when there isn't any burst indication in the file namee
//
// Ex: IMG_0001.JPG, IMG_0002.JPG, etc. and the date taken is different by a fraction of second
// Ex: IMG_0001.JPG, IMG_0001.RAW, IMG_0002.JPG, IMG_0002.RAW, etc.
//
// Edited images, images identified as as burst already are not considered.
// The in channel receives assets sorted by date taken.
func Group(ctx context.Context, in <-chan *assets.Asset, out chan<- *assets.Asset, gOut chan<- *assets.Group) {
	var currentGroup []*assets.Asset
	var lastTaken time.Time
	var lastRadical string

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

			// exclude movies, edited or burst images
			// exclude images without a date taken
			// exclude images taken more than frameInterval apart
			ni := a.NameInfo

			d := getAssetCaptureDate(a)
			dontGroupMe := ni.Type != filetypes.TypeImage ||
				d.IsZero() ||
				ni.Kind == assets.KindBurst ||
				ni.Kind == assets.KindEdited

			if ni.Radical != lastRadical && abs(d.Sub(lastTaken)) > frameInterval {
				dontGroupMe = true
			}

			if dontGroupMe {
				if len(currentGroup) > 0 {
					sendBurstGroup(ctx, out, gOut, currentGroup)
				}
				currentGroup = []*assets.Asset{a}
			} else {
				currentGroup = append(currentGroup, a)
			}
			lastRadical = ni.Radical
			lastTaken = d
		}
	}
}

func getAssetCaptureDate(a *assets.Asset) time.Time {
	if !a.CaptureDate.IsZero() {
		return a.CaptureDate
	}
	return a.FileDate
}

// abs returns the absolute value of a given integer.
func abs[T constraints.Integer](x T) T {
	if x < 0 {
		return -x
	}
	return x
}

func sendBurstGroup(ctx context.Context, out chan<- *assets.Asset, outg chan<- *assets.Group, as []*assets.Asset) {
	if len(as) == 0 {
		return
	}
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
