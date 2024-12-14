package assets

import (
	"errors"
)

type GroupBy int

const (
	GroupByNone    GroupBy = iota
	GroupByBurst           // Group by burst
	GroupByRawJpg          // Group by raw/jpg
	GroupByHeicJpg         // Group by heic/jpg
	GroupByOther           // Group by other (same radical, not previous cases)
)

type removed struct {
	Asset  *Asset
	Reason string
}

type Group struct {
	Assets     []*Asset
	Removed    []removed
	Grouping   GroupBy
	CoverIndex int // index of the cover assert in the Assets slice
}

// NewGroup create a new asset group
func NewGroup(grouping GroupBy, a ...*Asset) *Group {
	return &Group{
		Grouping: grouping,
		Assets:   a,
	}
}

// AddAsset add an asset to the group
func (g *Group) AddAsset(a *Asset) {
	g.Assets = append(g.Assets, a)
}

// RemoveAsset remove an asset from the group
func (g *Group) RemoveAsset(a *Asset, reason string) {
	for i, asset := range g.Assets {
		if asset == a {
			g.Removed = append(g.Removed, removed{Asset: asset, Reason: reason})
			g.Assets = append(g.Assets[:i], g.Assets[i+1:]...)
			return
		}
	}
}

// SetCover set the cover asset of the group
func (g *Group) SetCover(i int) *Group {
	g.CoverIndex = i
	return g
}

func (g *Group) Validate() error {
	if g == nil {
		return errors.New("nil group")
	}
	if len(g.Assets) == 0 {
		return errors.New("empty group")
	}
	// test all asset not nil
	for _, a := range g.Assets {
		if a == nil {
			return errors.New("nil asset in group")
		}
	}
	if 0 > g.CoverIndex || g.CoverIndex > len(g.Assets) {
		return errors.New("cover index out of range")
	}
	return nil
}
