package filters

import (
	"fmt"
	"strings"

	"github.com/simulot/immich-go/internal/assets"
)

type BurstFlag int

const (
	BurstNothing  BurstFlag = iota
	BurstStack              // Stack burst photos, all the photos in the burst are kept
	BurstkKeepRaw           // Stack burst, keep raw photos when when have JPEG and raw
	BurstKeepJPEG           // Stack burst, keep JPEG photos when when have JPEG and raw
)

func (flg BurstFlag) GroupFilter() Filter {
	switch flg {
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
	selectedAssets := []*assets.Asset{}
	for _, a := range g.Assets {
		if a.NameInfo().Ext == ".jpg" || a.NameInfo().Ext == ".jpeg" {
			g.Removed = append(g.Removed, a)
		} else {
			selectedAssets = append(selectedAssets, a)
		}
	}
	g.Assets = selectedAssets
	return g
}

func stackBurstKeepJPEG(g *assets.Group) *assets.Group {
	if g.Grouping != assets.GroupByBurst {
		return g
	}
	// Keep only jpe files
	selectedAssets := []*assets.Asset{}
	for _, a := range g.Assets {
		if a.NameInfo().Ext == ".jpg" || a.NameInfo().Ext == ".jpeg" {
			selectedAssets = append(selectedAssets, a)
		} else {
			g.Removed = append(g.Removed, a)
		}
	}
	g.Assets = selectedAssets
	return g
}

// Implement spf13 flag.Value interface

func (b *BurstFlag) Set(value string) error {
	switch strings.ToLower(value) {
	case "":
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
		return ""
	case BurstStack:
		return "Stack"
	case BurstkKeepRaw:
		return "StackKeepRaw"
	case BurstKeepJPEG:
		return "StackKeepJPEG"
	default:
		return "Unknown"
	}
}

func (b BurstFlag) Type() string {
	return "BurstFlag"
}
