package immich

import (
	"context"
	"fmt"
)

type TagSimplified struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

type TagAssetsResponse struct {
	Error   string `json:"error,omitempty"` // [duplicate, no_permission, not_found, unknown]
	ID      string `json:"id"`
	Success bool   `json:"success"`
}

func (ic *ImmichClient) UpsertTags(ctx context.Context, tags []string) ([]TagSimplified, error) {
	var resp []TagSimplified
	body := struct {
		Tags []string `json:"tags"`
	}{Tags: tags}
	err := ic.newServerCall(ctx, EndPointUpsertTags).
		do(putRequest("/tags", setJSONBody(body), setAcceptJSON()), responseJSON(&resp))
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (ic *ImmichClient) TagAssets(
	ctx context.Context,
	tagID string,
	assetIDs []string,
) ([]TagAssetsResponse, error) {
	var resp []TagAssetsResponse

	body := struct {
		IDs []string `json:"ids"`
	}{IDs: assetIDs}
	err := ic.newServerCall(ctx, EndPointTagAssets).
		do(putRequest(fmt.Sprintf("/tags/%s/assets", tagID), setJSONBody(body), setAcceptJSON()), responseJSON(&resp))

	if err != nil {
		return nil, err
	}
	return resp, nil
}
