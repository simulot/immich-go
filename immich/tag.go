package immich

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/simulot/immich-go/internal/assets"
)

type TagSimplified struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (ts TagSimplified) AsTag() assets.Tag {
	return assets.Tag{
		ID:    ts.ID,
		Name:  ts.Name,
		Value: ts.Value,
	}
}

type TagAssetsResponse struct {
	Error   string `json:"error,omitempty"` // [duplicate, no_permission, not_found, unknown]
	ID      string `json:"id"`
	Success bool   `json:"success"`
}

func (ic *ImmichClient) UpsertTags(ctx context.Context, tags []string) ([]TagSimplified, error) {
	if ic.dryRun {
		resp := make([]TagSimplified, len(tags))
		for i, t := range tags {
			resp[i] = TagSimplified{
				ID:    uuid.NewString(),
				Name:  t,
				Value: t,
			}
		}
		return resp, nil
	}
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
	if ic.dryRun {
		resp := make([]TagAssetsResponse, len(assetIDs))
		for i, a := range assetIDs {
			resp[i] = TagAssetsResponse{
				ID:      a,
				Success: true,
			}
		}
		return resp, nil
	}

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

func (ic *ImmichClient) BulkTagAssets(
	ctx context.Context,
	tagIDs []string,
	assetIDs []string,
) (struct {
	Count int `json:"count"`
}, error,
) {
	if ic.dryRun {
		return struct {
			Count int `json:"count"`
		}{Count: len(assetIDs)}, nil
	}

	var resp struct {
		Count int `json:"count"`
	}

	body := struct {
		TagIDs   []string `json:"tagIds"`
		AssetIDs []string `json:"assetIds"`
	}{
		TagIDs:   tagIDs,
		AssetIDs: assetIDs,
	}
	err := ic.newServerCall(ctx, EndPointBulkTagAssets).
		do(putRequest("/tags/assets", setJSONBody(body)), responseJSON(&resp))

	return resp, err
}

func (ic *ImmichClient) GetAllTags(ctx context.Context) ([]TagSimplified, error) {
	var resp []TagSimplified
	err := ic.newServerCall(ctx, EndPointGetAllTags).
		do(getRequest("/tags"), responseJSON(&resp))
	if err != nil {
		return nil, err
	}
	return resp, nil
}
