package series

/* This package implements a group builder for series of images.
A series is a group of images with the same radical part in their name.
*/

import (
	"context"
	"time"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/filetypes"
	"golang.org/x/exp/constraints"
)

const (
	threshold = 1 * time.Second
)

// Group groups assets by series, based on the radical part of the name.
// the in channel receives assets sorted by radical, then by date taken.
func Group(ctx context.Context, in <-chan *assets.Asset, out chan<- *assets.Asset, gOut chan<- *assets.Group) {
	currentRadical := ""
	currentCaptureDate := time.Time{}
	currentGroup := []*assets.Asset{}

	for {
		select {
		case <-ctx.Done():
			return
		case a, ok := <-in:
			if !ok {
				if len(currentGroup) > 0 {
					sendGroup(ctx, out, gOut, currentGroup)
				}
				return
			}
			r := a.Radical
			cd := getAssetCaptureDate(a)
			if r != currentRadical || a.Type != filetypes.TypeImage || cd.IsZero() || abs(cd.Sub(currentCaptureDate)) > threshold {
				if len(currentGroup) > 0 {
					sendGroup(ctx, out, gOut, currentGroup)
					currentGroup = []*assets.Asset{}
				}
				currentRadical = r
				currentCaptureDate = cd
			}
			currentGroup = append(currentGroup, a)
		}
	}
}

func getAssetCaptureDate(a *assets.Asset) time.Time {
	if !a.CaptureDate.IsZero() {
		return a.CaptureDate
	}
	return a.FileDate
}

func sendGroup(ctx context.Context, out chan<- *assets.Asset, outg chan<- *assets.Group, as []*assets.Asset) {
	if len(as) < 2 {
		// Not a series
		sendAsset(ctx, out, as)
		return
	}
	grouping := assets.GroupByOther

	gotJPG := false
	gotRAW := false
	gotHEIC := false
	gotMP4 := false
	gotMOV := false

	cover := 0
	// determine if the group is a burst
	for i, a := range as {
		gotMP4 = gotMP4 || a.Ext == ".mp4"
		gotMOV = gotMOV || a.Ext == ".mov"
		gotJPG = gotJPG || a.Ext == ".jpg"
		gotRAW = gotRAW || filetypes.IsRawFile(a.Ext)
		gotHEIC = gotHEIC || a.Ext == ".heic" || a.Ext == ".heif"
		if grouping == assets.GroupByOther {
			switch a.Kind {
			case assets.KindBurst:
				grouping = assets.GroupByBurst
			}
		}
		if a.IsCover {
			cover = i
		}
	}

	// If we have only two assets, we can try to group them as raw/jpg or heic/jpg
	if len(as) == 2 {
		if grouping == assets.GroupByOther {
			if gotJPG && gotRAW && !gotHEIC {
				grouping = assets.GroupByRawJpg
			} else if gotJPG && !gotRAW && gotHEIC {
				grouping = assets.GroupByHeicJpg
			} else if (gotMP4 || gotMOV) && (gotJPG || gotHEIC) {
				grouping = assets.GroupByNone
			}
		}
		if grouping == assets.GroupByNone {
			for _, a := range as {
				select {
				case out <- a:
				case <-ctx.Done():
					return
				}
			}
			return
		}
	}

	// good to go
	g := assets.NewGroup(grouping, as...)
	g.CoverIndex = cover

	select {
	case <-ctx.Done():
		return
	case outg <- g:
	}
}

// sendAsset sends assets of the group as individual assets to the output channel
func sendAsset(ctx context.Context, out chan<- *assets.Asset, assets []*assets.Asset) {
	for _, a := range assets {
		select {
		case out <- a:
		case <-ctx.Done():
			return
		}
	}
}

func abs[T constraints.Integer](x T) T {
	if x < 0 {
		return -x
	}
	return x
}
