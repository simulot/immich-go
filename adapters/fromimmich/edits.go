package fromimmich

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/simulot/immich-go/immich"
)

// editList returns the edit list of an edited asset, as its sidecar records it: the actions (crop,
// rotate, mirror) in the order Immich applies them to the original, each with its action and its
// parameters as the server gives them, but without its id. The id names the action on this server
// only (saving the edits again gives new ids), and the sidecar is rewritten only when its content
// changes: with no id, and the keys of each action in a fixed order, the list is the same from run
// to run while the edits are. It returns nil when the asset has no edits.
func (fic *FromImmichCmd) editList(ctx context.Context, id string) (json.RawMessage, error) {
	raw, err := fic.client.Immich.(immich.ImmichAssetEditsInterface).GetAssetEdits(ctx, id)
	if err != nil {
		return nil, err
	}
	var actions []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &actions); err != nil {
		return nil, fmt.Errorf("the edits of asset %s: %w", id, err)
	}
	if len(actions) == 0 {
		return nil, nil
	}
	for _, action := range actions {
		delete(action, "id")
	}
	return json.Marshal(actions)
}
