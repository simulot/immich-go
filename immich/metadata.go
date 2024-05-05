package immich

import (
	"context"
)

type searchMetadataBody interface {
	setPage(p int)
}

type searchMetadataResponse struct {
	Assets struct {
		Total    int      `json:"total"`
		Count    int      `json:"count"`
		Items    []*Asset `json:"items"`
		NextPage int      `json:"nextPage,string"`
	}
}

type searchMetadataGetAllBody struct {
	Page      int  `json:"page"`
	WithExif  bool `json:"withExif,omitempty"`
	IsVisible bool `json:"isVisible,omitempty"`
}

func (sb *searchMetadataGetAllBody) setPage(p int) {
	sb.Page = p
}

func (ic *ImmichClient) callSearchMetadata(ctx context.Context, req searchMetadataBody, filter func(*Asset) error) error {
	req.setPage(1)
	for {
		resp := searchMetadataResponse{}
		err := ic.newServerCall(ctx, "GetAllAssets").do(post("/search/metadata", "application/json", setJSONBody(&req), setAcceptJSON()), responseJSON(&resp))
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
			break
		}
		req.setPage(resp.Assets.NextPage)
	}
	return nil
}

func (ic *ImmichClient) GetAllAssets(ctx context.Context) ([]*Asset, error) {
	var assets []*Asset

	req := searchMetadataGetAllBody{Page: 1, WithExif: true, IsVisible: true}
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
	req := searchMetadataGetAllBody{Page: 1, WithExif: true, IsVisible: true}
	return ic.callSearchMetadata(ctx, &req, filter)
}
