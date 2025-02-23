package filters

import (
	"fmt"
	"strings"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/filetypes"
)

type BurstFlag int

const (
	BurstNothing  BurstFlag = iota
	BurstStack              // Stack burst photos, all the photos in the burst are kept
	BurstkKeepRaw           // Stack burst, keep raw photos when when have JPEG and raw
	BurstKeepJPEG           // Stack burst, keep JPEG photos when when have JPEG and raw
)

func (b BurstFlag) GroupFilter() Filter {
	switch b {
	case BurstNothing:
		return unGroupBurst
	case BurstStack:
		return groupBurst
	case BurstkKeepRaw:
		return groupBurstKeepRaw
	case BurstKeepJPEG:
		return stackBurstKeepJPEG
	default:
		return nil
	}
}

func unGroupBurst(g *assets.Group) *assets.Group {
	if g.Grouping != assets.GroupByBurst {
		return g
	}
	g.Grouping = assets.GroupByNone
	return g
}

func groupBurst(g *assets.Group) *assets.Group {
	return g
}

func groupBurstKeepRaw(g *assets.Group) *assets.Group {
	if g.Grouping != assets.GroupByBurst {
		return g
	}
	// Keep only raw files
	removedAssets := []*assets.Asset{}
	keep := 0
	for _, a := range g.Assets {
		if filetypes.IsRawFile(a.Ext) {
			keep++
		} else {
			removedAssets = append(removedAssets, a)
		}
	}
	if keep > 0 {
		for _, a := range removedAssets {
			g.RemoveAsset(a, "Keep only RAW files in burst")
		}
	}
	if len(g.Assets) < 2 {
		g.Grouping = assets.GroupByNone
	}
	return g
}

func stackBurstKeepJPEG(g *assets.Group) *assets.Group {
	if g.Grouping != assets.GroupByBurst {
		return g
	}
	// Keep only jpe files
	removedAssets := []*assets.Asset{}
	keep := 0
	for _, a := range g.Assets {
		if a.Ext == ".jpg" || a.Ext == ".jpeg" { // nolint: goconst
			keep++
		} else {
			removedAssets = append(removedAssets, a)
		}
	}
	if keep > 0 {
		for _, a := range removedAssets {
			g.RemoveAsset(a, "Keep only JPEG files in burst")
		}
	}
	if len(g.Assets) < 2 {
		g.Grouping = assets.GroupByNone
	}
	return g
}

// Implement spf13 flag.Value interface

func (b *BurstFlag) Set(value string) error {
	switch strings.ToLower(value) {
	case "", "nostack": // nolint: goconst
		*b = BurstNothing
	case "stack":
		*b = BurstStack
	case "stackkeepraw":
		*b = BurstkKeepRaw
	case "stackkeepjpeg":
		*b = BurstKeepJPEG
	default:
		return fmt.Errorf("invalid value %q for BurstFlag", value)
	}
	return nil
}

func (b BurstFlag) String() string {
	switch b {
	case BurstNothing:
		return "NoStack" // nolint: goconst
	case BurstStack:
		return "Stack"
	case BurstkKeepRaw:
		return "StackKeepRaw"
	case BurstKeepJPEG:
		return "StackKeepJPEG"
	default:
		return "Unknown" // nolint: goconst
	}
}

func (b BurstFlag) Type() string {
	return "BurstFlag"
}
