package fromimmich

import (
	"context"
	"sync"

	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/assets"
)

// stackIndex is what the run knows, before it reads the assets, of which ones are in a stack.
// GET /assets/{id} reports an asset's stack only while a member other than the primary is on the
// timeline and not trashed. GET /stacks (the key needs stack.read) lists every stack with its
// primary and its members that are not trashed or hidden, archived ones included; it places the
// others. The same search with withStacked=false lists only the assets in no stack, so an asset
// it leaves out is in one (checked once more for that asset alone, as paging can skip a row while
// the server changes). An asset in a stack that neither reports gets no stack in its sidecar (a
// stack is never guessed), and a warning.
type stackIndex struct {
	listed    map[string]assets.Stack // asset ID -> its stack, as GET /stacks lists it
	size      map[string]int          // stack ID -> its size, as GET /stacks lists it
	mu        sync.Mutex
	unstacked map[string]bool // the assets the withStacked=false search lists: in no stack
}

// noteUnstacked records an asset the withStacked=false search lists.
func (idx *stackIndex) noteUnstacked(id string) {
	idx.mu.Lock()
	idx.unstacked[id] = true
	idx.mu.Unlock()
}

// listStacks reads GET /stacks: each listed asset's stack (the primary always, the other members
// unless trashed or hidden), and each stack's size as listed. A key without stack.read gets a
// warning and no list; the stacks GET /assets/{id} reports are archived all the same.
func (fic *FromImmichCmd) listStacks(ctx context.Context) (*stackIndex, error) {
	idx := &stackIndex{listed: map[string]assets.Stack{}, size: map[string]int{}, unstacked: map[string]bool{}}
	stacks, err := fic.client.Immich.GetAllStacks(ctx)
	if immich.IsForbidden(err) {
		fic.app.Log().Warn("the API key lacks stack.read: a stack whose members but the primary are all archived cannot be archived")
		return idx, nil
	}
	if err != nil {
		return nil, err
	}
	for _, s := range stacks {
		members := map[string]bool{s.PrimaryAssetID: true}
		for _, m := range s.Assets {
			members[m.ID] = true
		}
		idx.size[s.ID] = len(members)
		for id := range members {
			idx.listed[id] = assets.Stack{ID: s.ID, Primary: id == s.PrimaryAssetID, AssetCount: len(members)}
		}
	}
	return idx, nil
}

// stackOf returns a's stack on the source server, as GET /assets/{id} reports it or GET /stacks
// lists it, or nil. An asset the withStacked=false search left out and that neither places is
// looked up once more: still in a stack, it gets a warning and no stack. A failed lookup is
// returned as the error.
func (fic *FromImmichCmd) stackOf(ctx context.Context, idx *stackIndex, a *immich.Asset) (*assets.Stack, error) {
	var st *assets.Stack
	if a.Stack != nil && a.Stack.ID != "" {
		st = &assets.Stack{ID: a.Stack.ID, Primary: a.Stack.PrimaryAssetID == a.ID, AssetCount: a.Stack.AssetCount}
	} else if l, ok := idx.listed[a.ID]; ok {
		st = &l
	}
	if st != nil {
		st.AssetCount = max(st.AssetCount, idx.size[st.ID]) // GET /stacks counts archived members too
		return st, nil
	}
	if idx.unstacked[a.ID] {
		return nil, nil
	}
	inStack, err := fic.client.Immich.IsInStack(ctx, a.ID)
	if err != nil {
		return nil, err
	}
	switch {
	case inStack && a.IsTrashed:
		fic.app.Log().Warn("trashed stack member that Immich does not report (GET /stacks leaves trashed members out): no stack in its sidecar",
			"id", a.ID, "file", a.OriginalFileName)
	case inStack:
		fic.app.Log().Warn("asset in a stack that Immich does not report (it is hidden, or GET /stacks was refused): no stack in its sidecar",
			"id", a.ID, "file", a.OriginalFileName)
	}
	return nil, nil
}
