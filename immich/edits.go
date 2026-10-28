package immich

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// ImmichAssetEditsInterface reads the edits of an asset. Not a part of the immich client interface,
// to simplify the client mocks, as ImmichGetSuggestion.
type ImmichAssetEditsInterface interface {
	GetAssetEdits(ctx context.Context, id string) (json.RawMessage, error)
}

var _ ImmichAssetEditsInterface = (*ImmichClient)(nil)

// ErrAssetEditsRefused is in the error of GetAssetEdits when the server refuses the call (HTTP 403):
// the API key lacks the asset.edit.get permission.
var ErrAssetEditsRefused = errors.New("the API key lacks the asset.edit.get permission")

// GetAssetEdits returns the edits of an asset, as the server gives them: a JSON array of actions
// (crop, rotate, mirror), each {"id", "action", "parameters"}, in the order Immich applies them to
// the original. Immich never changes the original: it renders the edited version from it with
// these actions. Since Immich v2.5.0. The API key needs the asset.edit.get permission: without it,
// the error is ErrAssetEditsRefused.
// https://api.immich.app/endpoints/assets/getAssetEdits
func (ic *ImmichClient) GetAssetEdits(ctx context.Context, id string) (json.RawMessage, error) {
	var r struct {
		Edits json.RawMessage `json:"edits"`
	}
	err := ic.newServerCall(ctx, "GetAssetEdits").do(getRequest("/assets/"+id+"/edits", setAcceptJSON()), responseJSON(&r))
	if ce := (callError{}); errors.As(err, &ce) && ce.status == http.StatusForbidden {
		return nil, fmt.Errorf("%w: %w", ErrAssetEditsRefused, err)
	}
	if err != nil {
		return nil, err
	}
	return r.Edits, nil
}
