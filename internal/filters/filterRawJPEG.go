package filters

import (
	"fmt"
	"strings"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/metadata"
)

type RawJPGFlag int

const (
	RawJPGNothing  RawJPGFlag = iota
	RawJPGKeepRaw             // Keep only raw files
	RawJPGKeepJPG             // Keep only JPEG files
	RawJPGStackRaw            // Stack raw and JPEG files, with the raw file as the cover
	RawJPGStackJPG            // Stack raw and JPEG files, with the JPEG file as the cover
)

func (flg RawJPGFlag) GroupFilter() Filter {
	switch flg {
	case RawJPGNothing:
		return unGroupRawJPGNothing
	case RawJPGKeepRaw:
		return groupRawJPGKeepRaw
	case RawJPGKeepJPG:
		return groupRawJPGKeepJPG
	case RawJPGStackRaw:
		return groupRawJPGStackRaw
	case RawJPGStackJPG:
		return groupRawJPGStackJPG
	default:
		return nil
	}
}

func unGroupRawJPGNothing(g *assets.Group) *assets.Group {
	if g.Grouping != assets.GroupByRawJpg {
		return g
	}
	g.Grouping = assets.GroupByNone
	return g
}

func groupRawJPGKeepRaw(g *assets.Group) *assets.Group {
	if g.Grouping != assets.GroupByRawJpg {
		return g
	}
	// Keep only raw files
	KeptAssets := []*assets.Asset{}
	for _, a := range g.Assets {
		if metadata.IsRawFile(a.NameInfo().Ext) {
			KeptAssets = append(KeptAssets, a)
		} else {
			g.Removed = append(g.Removed, a)
		}
	}
	g.Assets = KeptAssets
	if len(g.Assets) < 2 {
		g.Grouping = assets.GroupByNone
	}
	return g
}

func groupRawJPGKeepJPG(g *assets.Group) *assets.Group {
	if g.Grouping != assets.GroupByRawJpg {
		return g
	}
	// Keep only JPEG files
	KeptAssets := []*assets.Asset{}
	for _, a := range g.Assets {
		if a.NameInfo().Ext == ".jpg" || a.NameInfo().Ext == ".jpeg" {
			KeptAssets = append(KeptAssets, a)
		} else {
			g.Removed = append(g.Removed, a)
		}
	}
	g.Assets = KeptAssets
	if len(g.Assets) < 2 {
		g.Grouping = assets.GroupByNone
	}
	return g
}

func groupRawJPGStackRaw(g *assets.Group) *assets.Group {
	if g.Grouping != assets.GroupByRawJpg {
		return g
	}
	// Set the cover index to the first RAW file
	for i, a := range g.Assets {
		if metadata.IsRawFile(a.NameInfo().Ext) {
			g.CoverIndex = i
			break
		}
	}
	return g
}

func groupRawJPGStackJPG(g *assets.Group) *assets.Group {
	if g.Grouping != assets.GroupByRawJpg {
		return g
	}
	// Set the cover index to the first JPEG file
	for i, a := range g.Assets {
		if a.NameInfo().Ext == ".jpg" || a.NameInfo().Ext == ".jpeg" {
			g.CoverIndex = i
			break
		}
	}
	return g
}

func (r *RawJPGFlag) Set(value string) error {
	switch strings.ToLower(value) {
	case "":
		*r = RawJPGNothing
	case "keepraw":
		*r = RawJPGKeepRaw
	case "keepjpg":
		*r = RawJPGKeepJPG
	case "stackcoverraw":
		*r = RawJPGStackRaw
	case "stackcoverjpg":
		*r = RawJPGStackJPG
	default:
		return fmt.Errorf("invalid value %q for RawJPGFlag", value)
	}
	return nil
}

func (r RawJPGFlag) String() string {
	switch r {
	case RawJPGNothing:
		return ""
	case RawJPGKeepRaw:
		return "KeepRaw"
	case RawJPGKeepJPG:
		return "KeepJPG"
	case RawJPGStackRaw:
		return "StackCoverRaw"
	case RawJPGStackJPG:
		return "StackCoverJPG"
	default:
		return "Unknown"
	}
}

func (r RawJPGFlag) Type() string {
	return "RawJPGFlag"
}
