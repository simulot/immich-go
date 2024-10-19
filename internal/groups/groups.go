package groups

import (
	"context"
	"errors"
	"time"
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

type Kind int

const (
	KindNone Kind = iota
	KindMotionPhoto
	KindBurst
	KindRawJpg
	KindEdited
)

type Asset interface {
	Base() string         // base name
	Radical() string      // base name without extension
	Ext() string          // extension
	DateTaken() time.Time // date taken
	Type() string         // type of the asset  video, image
}

type AssetGroup struct {
	Assets     []Asset
	Kind       Kind
	CoverIndex int // index of the cover assert in the Assets slice
}

// NewAssetGroup create a new asset group
func NewAssetGroup(kind Kind, a ...Asset) *AssetGroup {
	return &AssetGroup{
		Kind:   kind,
		Assets: a,
	}
}

// AddAsset add an asset to the group
func (g *AssetGroup) AddAsset(a Asset) {
	g.Assets = append(g.Assets, a)
}

// SetCover set the cover asset of the group
func (g *AssetGroup) SetCover(i int) *AssetGroup {
	g.CoverIndex = i
	return g
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

/*
A grouper is a filter that inspects the assets and creates groups of assets based on date taken
and file names.

A group builder requires that assets are delivered the entry channel sorted by date taken. Doing so, isolated assets
are released quickly, and can be processed by the next filter in the pipeline.

Detected groups are sent to the output channel for been passed to the main upload program
*/
type Grouper interface {
	Group(ctx context.Context, in <-chan Asset, outg <-chan *AssetGroup, outa chan<- Asset)
}

/*
A grouper pipeline is a chain of groupers that process assets in sequence.
The 1st grouper should be the one that detects the most specific groups, and the last one should detect the most generic ones.
This way, the most specific groups are detected first, and the most generic ones are detected last.
*/

type GrouperPipeline struct {
	groupers []Grouper
}

func NewGrouperPipeline(ctx context.Context, gs ...Grouper) *GrouperPipeline {
	g := &GrouperPipeline{
		groupers: gs,
	}
	return g
}

func pipeChannels[T any](ctx context.Context, in chan T) chan T {
	outa := make(chan T)
	go func() {
		defer close(outa)
		for {
			select {
			case <-ctx.Done():
				return

			case a, ok := <-in:
				if !ok {
					return
				}
				outa <- a

			}
		}
	}()
	return outa
}

func (p *GrouperPipeline) Process(ctx context.Context, in chan Asset) (<-chan *AssetGroup, chan Asset) {
	outg := make(<-chan *AssetGroup)
	var outa chan Asset

	for _, g := range p.groupers {
		outa := make(chan Asset)
		go g.Group(ctx, in, outg, outa)
		in = outa // next grouper input is the current grouper output
	}
	return outg, outa
}
