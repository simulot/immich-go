package immich

import (
	"context"
)

type searchMetadataResponse struct {
	Assets struct {
		Total    int      `json:"total"`
		Count    int      `json:"count"`
		Items    []*Asset `json:"items"`
		NextPage int      `json:"nextPage,string"`
	}
}

type SearchMetadataQuery struct {
	// pagination
	Page int `json:"page"`
	Size int `json:"size,omitempty"`

	// filters
	WithExif      bool   `json:"withExif,omitempty"`
	IsVisible     bool   `json:"isVisible,omitempty"`
	WithDeleted   bool   `json:"withDeleted,omitempty"`
	TakenBefore   string `json:"takenBefore,omitempty"`
	TakenAfter    string `json:"takenAfter,omitempty"`
	WithArchived  bool   `json:"withArchived,omitempty"`
	TrashedAfter  string `json:"trashedAfter,omitempty"`
	TrashedBefore string `json:"trashedBefore,omitempty"`
	Model         string `json:"model,omitempty"`
	Make          string `json:"make,omitempty"`
}

func (ic *ImmichClient) callSearchMetadata(ctx context.Context, query *SearchMetadataQuery, filter func(*Asset) error) error {
	query.Page = 1
	query.Size = 1000
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			resp := searchMetadataResponse{}
			err := ic.newServerCall(ctx, EndPointGetAllAssets).do(postRequest("/search/metadata", "application/json", setJSONBody(&query), setAcceptJSON()), responseJSON(&resp))
			if err != nil {
				return err
			}

			for _, a := range resp.Assets.Items {
				err = filter(a)
				if err != nil {
					return err
				}
			}

			if resp.Assets.NextPage == 0 {
				return nil
			}
			query.Page = resp.Assets.NextPage
		}
	}
}

func (ic *ImmichClient) GetAllAssets(ctx context.Context) ([]*Asset, error) {
	var assets []*Asset

	req := SearchMetadataQuery{Page: 1, WithExif: true, IsVisible: true, WithDeleted: true}
	err := ic.callSearchMetadata(ctx, &req, func(asset *Asset) error {
		assets = append(assets, asset)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return assets, nil
}

func (ic *ImmichClient) GetAllAssetsWithFilter(ctx context.Context, query *SearchMetadataQuery, filter func(*Asset) error) error {
	if query == nil {
		query = &SearchMetadataQuery{Page: 1, WithExif: true, IsVisible: true, WithDeleted: true}
	}
	query.Page = 1
	return ic.callSearchMetadata(ctx, query, filter)
}
