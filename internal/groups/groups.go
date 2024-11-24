package groups

import (
	"context"
	"sync"

	"github.com/simulot/immich-go/internal/assets"
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

// Grouper is an interface for a type that can group assets.
type Grouper func(ctx context.Context, in <-chan *assets.Asset, out chan<- *assets.Asset, gOut chan<- *assets.Group)

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

// PipeGrouper groups assets in a pipeline of groupers.
// Group opens and closes intermediate channels as required.
func (p *GrouperPipeline) PipeGrouper(ctx context.Context, in chan *assets.Asset) chan *assets.Group {
	// Create channels
	gOut := make(chan *assets.Group) // output channel for groups
	out := make(chan *assets.Asset)  // output channel for the last grouper

	inChans := make([]chan *assets.Asset, len(p.groupers))
	outChans := make([]chan *assets.Asset, len(p.groupers))

	// initialize channels for each grouper
	for i := range p.groupers {
		if i == 0 {
			inChans[i] = in
		} else {
			inChans[i] = outChans[i-1]
		}
		if i < len(p.groupers)-1 {
			outChans[i] = make(chan *assets.Asset) // intermediate channels between groupers
		} else {
			outChans[i] = out
		}
	}

	// call groupers with the appropriate channels
	wg := sync.WaitGroup{}
	for i := range p.groupers {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			p.groupers[i](ctx, inChans[i], outChans[i], gOut)
			if i < len(p.groupers)-1 {
				close(outChans[i]) // close intermediate channels
			}
		}(i)
	}

	// wait for all groupers to finish and close the output channel
	go func() {
		wg.Wait()
		close(out)
	}()

	// groups standalone assets
	go func() {
		defer close(gOut)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				a, ok := <-out
				if !ok {
					return
				}
				if a != nil {
					gOut <- assets.NewGroup(assets.GroupByNone, a)
				}
			}
		}
	}()

	return gOut
}
