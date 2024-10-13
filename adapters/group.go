package adapters

import (
	"errors"
)

// A group of assets link assets that are linked together. This
// allows a specific treatment of the group.
//
// Groups can be:
//   - A photo and a movie as for motion picture or live photo
//   - A couple of RAW and JPG image
//   - A burst of photos
//   - A photo and its edited version
//
// A group has an asset that represents the group:
//   - for Raw/JPG --> the JPG
//	 - for Bursts: the photo identified as the cover
//   - not relevant for live photo
//
// All group's assets can be added to 0 or more albums

type GroupKind int

const (
	GroupKindNone GroupKind = iota
	GroupKindMotionPhoto
	GroupKindBurst
	GroupKindRawJpg
	GroupKindEdited
)

type AssetGroup struct {
	Kind       GroupKind
	CoverIndex int // index of the cover assert in the Assets slice
	Assets     []*LocalAssetFile
	Albums     []LocalAlbum
}

// NewAssetGroup create a new asset group
func NewAssetGroup(kind GroupKind, a ...*LocalAssetFile) *AssetGroup {
	return &AssetGroup{
		Kind:   kind,
		Assets: a,
	}
}

// AddAlbum assign the group to an album
func (g *AssetGroup) AddAlbum(album LocalAlbum) {
	for _, al := range g.Albums {
		if al.Title == album.Title {
			return
		}
	}
	g.Albums = append(g.Albums, album)
}

// AddAsset add an asset to the group
func (g *AssetGroup) AddAsset(a *LocalAssetFile) {
	g.Assets = append(g.Assets, a)
}

func (g *AssetGroup) Validate() error {
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
