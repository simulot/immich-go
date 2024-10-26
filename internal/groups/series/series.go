package series

/* This package implements a group builder for series of images.
A series is a group of images with the same radical part in their name.
*/

import (
	"context"
	"time"

	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/groups"
	"github.com/simulot/immich-go/internal/groups/groupby"
	"github.com/simulot/immich-go/internal/metadata"
	"golang.org/x/exp/constraints"
)

// Group groups assets by series, based on the radical part of the name.
// the in channel receives assets sorted by radical, then by date taken.
func Group(ctx context.Context, in <-chan groups.Asset, out chan<- groups.Asset, gOut chan<- *groups.AssetGroup) {
	currentRadical := ""
	currentGroup := []groups.Asset{}

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

			if r := a.NameInfo().Radical; r != currentRadical {
				if len(currentGroup) > 0 {
					sendGroup(ctx, out, gOut, currentGroup)
					currentGroup = []groups.Asset{}
				}
				currentRadical = r
			}
			currentGroup = append(currentGroup, a)
		}
	}
}

func sendGroup(ctx context.Context, out chan<- groups.Asset, outg chan<- *groups.AssetGroup, assets []groups.Asset) {
	if len(assets) < 2 {
		// Not a series
		sendAsset(ctx, out, assets)
		return
	}
	grouping := groupby.GroupByOther

	gotJPG := false
	gotRAW := false
	gotHEIC := false

	cover := 0
	// determine if the group is a burst
	for i, a := range assets {
		gotJPG = gotJPG || a.NameInfo().Ext == ".jpg"
		gotRAW = gotRAW || metadata.IsRawFile(a.NameInfo().Ext)
		gotHEIC = gotHEIC || a.NameInfo().Ext == ".heic" || a.NameInfo().Ext == ".heif"
		if grouping == groupby.GroupByOther {
			switch a.NameInfo().Kind {
			case filenames.KindBurst:
				grouping = groupby.GroupByBurst
			}
		}
		if a.NameInfo().IsCover {
			cover = i
		}
	}

	// If we have only two assets, we can try to group them as raw/jpg or heic/jpg
	if len(assets) == 2 {
		if grouping == groupby.GroupByOther {
			if gotJPG && gotRAW && !gotHEIC {
				grouping = groupby.GroupByRawJpg
			} else if gotJPG && !gotRAW && gotHEIC {
				grouping = groupby.GroupByHeicJpg
			}
		}
		// check the delay between the two assets, if it's too long, we don't group them
		if grouping == groupby.GroupByRawJpg || grouping == groupby.GroupByHeicJpg {
			d := assets[0].DateTaken()
			if abs(d.Sub(assets[1].DateTaken())) > 500*time.Millisecond {
				sendAsset(ctx, out, assets)
				return
			}
		}
	}

	// good to go
	g := groups.NewAssetGroup(grouping, assets...)
	g.CoverIndex = cover

	select {
	case <-ctx.Done():
		return
	case outg <- g:
	}
}

// sendAsset sends assets of the group as individual assets to the output channel
func sendAsset(ctx context.Context, out chan<- groups.Asset, assets []groups.Asset) {
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
