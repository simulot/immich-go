package filters

import (
	"fmt"
	"strings"

	"github.com/simulot/immich-go/internal/assets"
)

type HeicJpgFlag int

const (
	HeicJpgNothing   HeicJpgFlag = iota
	HeicJpgKeepHeic              // Keep only HEIC files
	HeicJpgKeepJPG               // Keep only JPEG files
	HeicJpgStackHeic             // Stack HEIC and JPEG files, with the HEIC file as the cover
	HeicJpgStackJPG              // Stack HEIC and JPEG files, with the JPEG file as the cover
)

func (h HeicJpgFlag) GroupFilter() Filter {
	switch h {
	case HeicJpgNothing:
		return unGroupHeicJpeg
	case HeicJpgKeepHeic:
		return groupHeicJpgKeepHeic
	case HeicJpgKeepJPG:
		return groupHeicJpgKeepJPG
	case HeicJpgStackHeic:
		return groupHeicJpgStackHeic
	case HeicJpgStackJPG:
		return groupHeicJpgStackJPG
	default:
		return nil
	}
}

func unGroupHeicJpeg(g *assets.Group) *assets.Group {
	if g.Grouping != assets.GroupByHeicJpg {
		return g
	}
	g.Grouping = assets.GroupByNone
	return g
}

func groupHeicJpgKeepHeic(g *assets.Group) *assets.Group {
	if g.Grouping != assets.GroupByHeicJpg {
		return g
	}
	// Keep only heic files
	removedAssets := []*assets.Asset{}
	keep := 0
	for _, a := range g.Assets {
		if a.Ext == ".heic" {
			keep++
		} else {
			removedAssets = append(removedAssets, a)
		}
	}

	if keep > 0 {
		for _, a := range removedAssets {
			g.RemoveAsset(a, "Keep only HEIC files in HEIC/JPEG group")
		}
	}
	if len(g.Assets) < 2 {
		g.Grouping = assets.GroupByNone
	}
	return g
}

func groupHeicJpgKeepJPG(g *assets.Group) *assets.Group {
	if g.Grouping != assets.GroupByHeicJpg {
		return g
	}
	// Keep only heic files
	removedAssets := []*assets.Asset{}
	keep := 0
	for _, a := range g.Assets {
		if a.Ext == ".jpg" || a.Ext == ".jpeg" {
			keep++
		} else {
			removedAssets = append(removedAssets, a)
		}
	}
	if keep > 0 {
		for _, a := range removedAssets {
			g.RemoveAsset(a, "Keep only HEIC files in HEIC/JPEG group")
		}
	}
	if len(g.Assets) < 2 {
		g.Grouping = assets.GroupByNone
	}
	return g
}

func groupHeicJpgStackHeic(g *assets.Group) *assets.Group {
	if g.Grouping != assets.GroupByHeicJpg {
		return g
	}
	// Set the cover index to the first HEIC file
	for i, a := range g.Assets {
		if a.Ext == ".heic" {
			g.CoverIndex = i
			break
		}
	}
	return g
}

func groupHeicJpgStackJPG(g *assets.Group) *assets.Group {
	if g.Grouping != assets.GroupByHeicJpg {
		return g
	}
	// Set the cover index to the first JPEG file
	for i, a := range g.Assets {
		if a.Ext == ".jpg" || a.Ext == ".jpeg" {
			g.CoverIndex = i
			break
		}
	}
	return g
}

func (h *HeicJpgFlag) Set(value string) error {
	switch strings.ToLower(value) {
	case "":
		*h = HeicJpgNothing
	case "keepheic":
		*h = HeicJpgKeepHeic
	case "keepjpg":
		*h = HeicJpgKeepJPG
	case "stackcoverheic":
		*h = HeicJpgStackHeic
	case "stackcoverjpg":
		*h = HeicJpgStackJPG
	default:
		return fmt.Errorf("invalid value %q for HeicJpgFlag", value)
	}
	return nil
}

func (h HeicJpgFlag) String() string {
	switch h {
	case HeicJpgNothing:
		return ""
	case HeicJpgKeepHeic:
		return "KeepHeic"
	case HeicJpgKeepJPG:
		return "KeepJPG"
	case HeicJpgStackHeic:
		return "StackCoverHeic"
	case HeicJpgStackJPG:
		return "StackCoverJPG"
	default:
		return "Unknown"
	}
}

func (h HeicJpgFlag) Type() string {
	return "HeicJpgFlag"
}
