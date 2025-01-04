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
	WithExif         bool   `json:"withExif,omitempty"`
	IsVisible        bool   `json:"isVisible,omitempty"` // For motion stuff you need to pass isVisible=true to hide the motion ones (dijrasm91 — https://discord.com/channels/979116623879368755/1178366369423700080/1201206313699508295)
	WithDeleted      bool   `json:"withDeleted,omitempty"`
	WithArchived     bool   `json:"withArchived,omitempty"`
	TakenBefore      string `json:"takenBefore,omitempty"`
	TakenAfter       string `json:"takenAfter,omitempty"`
	Model            string `json:"model,omitempty"`
	Make             string `json:"make,omitempty"`
	Checksum         string `json:"checksum,omitempty"`
	OriginalFileName string `json:"originalFileName,omitempty"`
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

// GetAssetByHash returns the asset with the given hash
// The hash is the base64 encoded sha1 of the file
func (ic *ImmichClient) GetAssetsByHash(ctx context.Context, hash string) ([]*Asset, error) {
	query := SearchMetadataQuery{Page: 1, WithExif: true, IsVisible: true, WithDeleted: true, Checksum: hash}
	query.Page = 1
	list := []*Asset{}
	filter := func(asset *Asset) error {
		if asset.Checksum == hash {
			list = append(list, asset)
		}
		return nil
	}
	err := ic.callSearchMetadata(ctx, &query, filter)
	if err != nil {
		return nil, err
	}
	return list, nil
}

// GetAssetByHash returns the asset with the given hash
// The hash is the base64 encoded sha1 of the file
func (ic *ImmichClient) GetAssetsByImageName(ctx context.Context, name string) ([]*Asset, error) {
	query := SearchMetadataQuery{Page: 1, WithExif: true, IsVisible: true, WithDeleted: true, OriginalFileName: name}
	query.Page = 1
	list := []*Asset{}
	filter := func(asset *Asset) error {
		if asset.OriginalFileName == name {
			list = append(list, asset)
		}
		return nil
	}
	err := ic.callSearchMetadata(ctx, &query, filter)
	if err != nil {
		return nil, err
	}
	return list, nil
}
