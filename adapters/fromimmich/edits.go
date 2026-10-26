package fromimmich

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/assets"
)

// editLists is what a run knows of the edit lists it reads. The search calls getAssets' function
// from several goroutines.
type editLists struct {
	refused atomic.Bool  // the API key lacks asset.edit.get: GET /edits is not called again in the run
	unread  atomic.Int64 // edited assets sent on without their edit list, because of that
}

// readEditList puts the edit list of the edited asset id in md, as its sidecar records it: the
// actions (crop, rotate, mirror) in the order Immich applies them to the original, each with its
// action and its parameters as the server gives them, but without its id. The id names the action
// on this server only (saving the edits again gives new ids), and the sidecar is rewritten only when
// its content changes: with no id, and the keys of each action in a fixed order, the list is the
// same from run to run while the edits are. An asset with no edits gets none.
//
// A key without asset.edit.get is not an error, so that an archive job keeps working after an
// upgrade: the server refuses the call, md gets EditsUnread and no list, and the asset is archived
// without it (a .JSON file already written keeps the list it has). GET /edits is not called again
// in the run, and warnUnreadEditLists logs one warning at its end. Any other error is returned.
func (fic *FromImmichCmd) readEditList(ctx context.Context, id string, md *assets.Metadata) error {
	var raw json.RawMessage
	err := immich.ErrAssetEditsRefused
	if !fic.edits.refused.Load() {
		raw, err = fic.client.Immich.(immich.ImmichAssetEditsInterface).GetAssetEdits(ctx, id)
	}
	switch {
	case errors.Is(err, immich.ErrAssetEditsRefused):
		fic.edits.refused.Store(true)
		fic.edits.unread.Add(1)
		md.EditsUnread = true
		return nil
	case err != nil:
		return err
	}
	if md.Edits, err = withoutIDs(raw); err != nil {
		return fmt.Errorf("the edits of asset %s: %w", id, err)
	}
	return nil
}

// warnUnreadEditLists logs, once at the end of the run, how many edited assets were sent on without
// their edit list because the API key lacks asset.edit.get.
func (fic *FromImmichCmd) warnUnreadEditLists() {
	if n := fic.edits.unread.Load(); n > 0 {
		fic.app.Log().Warn("the API key lacks the asset.edit.get permission: edited assets archived without their edit list (a .JSON file already written keeps its own)", "count", n)
	}
}

// withoutIDs returns the edit actions without their ids, or nil when there is none.
func withoutIDs(edits json.RawMessage) (json.RawMessage, error) {
	var actions []map[string]json.RawMessage
	if err := json.Unmarshal(edits, &actions); err != nil {
		return nil, err
	}
	if len(actions) == 0 {
		return nil, nil
	}
	for _, action := range actions {
		delete(action, "id")
	}
	return json.Marshal(actions)
}
