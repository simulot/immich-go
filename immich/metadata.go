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

type searchMetadataGetAllBody struct {
	Page        int  `json:"page"`
	WithExif    bool `json:"withExif,omitempty"`
	IsVisible   bool `json:"isVisible,omitempty"`
	WithDeleted bool `json:"withDeleted,omitempty"`
	Size        int  `json:"size,omitempty"`
}

func (ic *ImmichClient) callSearchMetadata(ctx context.Context, req *searchMetadataGetAllBody, filter func(*Asset) error) error {
	req.Page = 1
	req.Size = 1000
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			resp := searchMetadataResponse{}
			err := ic.newServerCall(ctx, "GetAllAssets").do(postRequest("/search/metadata", "application/json", setJSONBody(&req), setAcceptJSON()), responseJSON(&resp))
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
			req.Page = resp.Assets.NextPage
		}
	}
}

func (ic *ImmichClient) GetAllAssets(ctx context.Context) ([]*Asset, error) {
	var assets []*Asset

	req := searchMetadataGetAllBody{Page: 1, WithExif: true, IsVisible: true, WithDeleted: true}
	err := ic.callSearchMetadata(ctx, &req, func(asset *Asset) error {
		assets = append(assets, asset)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return assets, nil
}

func (ic *ImmichClient) GetAllAssetsWithFilter(ctx context.Context, filter func(*Asset) error) error {
	req := searchMetadataGetAllBody{Page: 1, WithExif: true, IsVisible: true, WithDeleted: true}
	return ic.callSearchMetadata(ctx, &req, filter)
}
