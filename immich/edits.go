package immich

import (
	"context"
	"encoding/json"
)

// ImmichAssetEditsInterface reads the edits of an asset. Not a part of the immich client interface,
// to simplify the client mocks, as ImmichGetSuggestion.
type ImmichAssetEditsInterface interface {
	GetAssetEdits(ctx context.Context, id string) (json.RawMessage, error)
}

var _ ImmichAssetEditsInterface = (*ImmichClient)(nil)

// GetAssetEdits returns the edits of an asset, as the server gives them: a JSON array of actions
// (crop, rotate, mirror), each {"id", "action", "parameters"}, in the order Immich applies them to
// the original. Immich never changes the original: it renders the edited version from it with
// these actions. Since Immich v2.5.0; the API key needs the asset.edit.get permission.
// https://api.immich.app/endpoints/assets/getAssetEdits
func (ic *ImmichClient) GetAssetEdits(ctx context.Context, id string) (json.RawMessage, error) {
	var r struct {
		Edits json.RawMessage `json:"edits"`
	}
	err := ic.newServerCall(ctx, "GetAssetEdits").do(getRequest("/assets/"+id+"/edits", setAcceptJSON()), responseJSON(&r))
	if err != nil {
		return nil, err
	}
	return r.Edits, nil
}
