package immich

import (
	"context"
	"time"

	"github.com/simulot/immich-go/internal/assets"
	cliflags "github.com/simulot/immich-go/internal/cliFlags"
	"github.com/simulot/immich-go/internal/gen"
	"golang.org/x/sync/errgroup"
)

// The immich's search functionality doesn't provide the possibility to fetch assets with different properties
// with one call:
//  visibility:  archive or timeline or hidden or locked
//  trashed: date range
//  rate >= minimal rate

type searchOptions struct {
	withExif         bool
	takenRange       cliflags.DateRange // created date range
	withAll          bool               // to get all assets timeline,archive,hidden,locked and archived
	withOnlyTrashed  bool               // to get only trashed items
	withNotInAlbum   bool               // assets not in any album, set the isNotInAlbum
	withOnlyFavorite bool               // get only favorite assets

	// following filters are resolved as ID
	withAlbums []string // album names
	withTags   []string // tag names
	withPeople []string // people names

	// future options
	// deviceIds []string           // device id used for the upload
	// cities    []string           // city names
	// countries []string           // country names
	// states    []string           // state names

	// following filters requires several calls
	rates        []int
	visibilities []assets.Visibility
}

func SearchOptions() *searchOptions {
	return &searchOptions{}
}

// set WithExif
func (so *searchOptions) WithExif() *searchOptions {
	so.withExif = true
	return so
}

var defaultVisibility = []assets.Visibility{assets.VisibilityArchive, assets.VisibilityTimeline, assets.VisibilityHidden}

// set the queried visibilities in archive, timeline, hidden, locked values
func (so *searchOptions) WithVisibility(visibilities ...assets.Visibility) *searchOptions {
	gen.AddOnce(so.visibilities, visibilities...)
	return so
}

// get everything
func (so *searchOptions) All() *searchOptions {
	so.withAll = true
	so.visibilities = defaultVisibility
	so.withExif = true
	return so
}

// set the rates to be queried
func (so *searchOptions) WithMinimalRate(r int) *searchOptions {
	so.rates = nil
	if r >= 1 && r <= 5 {
		for i := r; i <= 5; i++ {
			so.rates = append(so.rates, i)
		}
	}
	return so
}

// to get the assets not belonging to any album, clear WithAlbums
func (so *searchOptions) WithNotInAlbum() *searchOptions {
	so.withNotInAlbum = true
	so.withAlbums = nil
	return so
}

// to get the assets belonging to the listed albums by name, reset WithNotInAlbum
func (so *searchOptions) WithAlbums(albums ...string) *searchOptions {
	gen.AddOnce(so.withAlbums, albums...)
	so.withNotInAlbum = false
	return so
}

// to get assets with listed tags (by name)
func (so *searchOptions) WithTags(tags ...string) *searchOptions {
	gen.AddOnce(so.withTags, tags...)
	return so
}

// to get assets with listed people only (by name)
func (so *searchOptions) WithPeople(people ...string) *searchOptions {
	gen.AddOnce(so.withPeople, people...)
	return so
}

// to get assets captured within the date range
func (so *searchOptions) WithDateRange(dr cliflags.DateRange) *searchOptions {
	so.takenRange = dr
	return so
}

// to get only favorite assets
func (so *searchOptions) WithOnlyFavorite() *searchOptions {
	so.withOnlyFavorite = true
	so.withOnlyTrashed = false
	so.visibilities = defaultVisibility
	return so
}

// to get only trashed items
func (so *searchOptions) WithOnlyTrashed() *searchOptions {
	so.withOnlyFavorite = false
	so.withOnlyTrashed = true
	so.visibilities = defaultVisibility
	return so
}

// to get only archived assets
func (so *searchOptions) WithOnlyArchived() *searchOptions {
	so.withOnlyFavorite = false
	so.withOnlyTrashed = false
	so.visibilities = []assets.Visibility{assets.VisibilityArchive}
	return so
}

// func (so *searchOptions) WithDeviceIds(deviceIds ...string) *searchOptions {
// 	gen.AddOnce(so.deviceIds, deviceIds...)
// 	return so
// }

// func (so *searchOptions) WithCounties(countries ...string) *searchOptions {
// 	gen.AddOnce(so.countries, countries...)
// 	return so
// }

// func (so *searchOptions) WithStates(states ...string) *searchOptions {
// 	gen.AddOnce(so.states, states...)
// 	return so
// }

// func (so *searchOptions) WithCities(cities ...string) *searchOptions {
// 	gen.AddOnce(so.cities, cities...)
// 	return so
// }

func (ic *ImmichClient) buildSearchQueries(so *searchOptions) []SearchMetadataQuery {
	base := SearchMetadataQuery{
		WithExif:     so.withExif,
		IsNotInAlbum: so.withNotInAlbum,
		IsFavorite:   so.withOnlyFavorite,
	}

	if !so.takenRange.Before.IsZero() {
		base.TakenBefore = so.takenRange.Before.AddDate(0, 0, 1).Add(-time.Millisecond).Format(TimeFormat)
	}
	if !so.takenRange.After.IsZero() {
		base.TakenAfter = so.takenRange.After.Format(TimeFormat)
	}

	// TODO albums, Tags and persons

	if len(so.visibilities) == 0 {
		so.visibilities = defaultVisibility
	}
	qs := []SearchMetadataQuery{}

	for _, v := range so.visibilities {
		if len(so.rates) == 0 {
			q := base
			q.Visibility = v
			if so.withOnlyTrashed {
				q.TrashedAfter = time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC).Format(TimeFormat)
			}
			qs = append(qs, q)
			continue
		}
		for _, r := range so.rates {
			q := base
			q.Visibility = v
			q.Rating = r
			qs = append(qs, q)
		}
	}

	if so.withAll {
		// add same queries but with TrashedAfter to the query set
		qs2 := []SearchMetadataQuery{}
		for _, q := range qs {
			q.TrashedAfter = time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC).Format(TimeFormat)
			qs2 = append(qs2, q)
		}
		qs = append(qs, qs2...)
	}

	return qs
}

func (ic *ImmichClient) GetAllAssets(ctx context.Context, filter func(*Asset) error) error {
	return ic.GetFilteredAssetsFn(ctx, SearchOptions().All(), filter)
}

func (ic *ImmichClient) GetFilteredAssetsFn(ctx context.Context, so *searchOptions, filter func(*Asset) error) error {
	qs := ic.buildSearchQueries(so)
	wg, ctx := errgroup.WithContext(ctx)
	wg.SetLimit(4) // most of the queries will return nothing
	for _, q := range qs {
		wg.Go(func() error {
			return ic.callSearchMetadata(ctx, &q, filter)
		})
	}
	return wg.Wait()
}

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
	WithExif         bool              `json:"withExif,omitempty"`
	IsFavorite       bool              `json:"isFavorite,omitempty"`
	IsNotInAlbum     bool              `json:"isNotInAlbum,omitempty"`
	WithDeleted      bool              `json:"withDeleted,omitempty"`
	AlbumIds         []string          `json:"albumIds,omitempty"`
	TagIds           []string          `json:"tagIds,omitempty"`
	TakenBefore      string            `json:"takenBefore,omitzero"`
	TakenAfter       string            `json:"takenAfter,omitzero"`
	TrashedAfter     string            `json:"trashedAfter,omitzero"`
	TrashedBefore    string            `json:"trashedBefore,omitzero"`
	Model            string            `json:"model,omitempty"`
	Make             string            `json:"make,omitempty"`
	Checksum         string            `json:"checksum,omitempty"`
	OriginalFileName string            `json:"originalFileName,omitempty"`
	Rating           int               `json:"rating,omitzero"`
	Visibility       assets.Visibility `json:"visibility,omitempty"`
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

func (ic *ImmichClient) GetAllAssetsWithFilter(ctx context.Context, query *SearchMetadataQuery, filter func(*Asset) error) error {
	if query == nil {
		query = &SearchMetadataQuery{Page: 1, WithExif: true, WithDeleted: true}
	}
	query.Page = 1
	return ic.callSearchMetadata(ctx, query, filter)
}

// GetAssetByHash returns the asset with the given hash
// The hash is the base64 encoded sha1 of the file
func (ic *ImmichClient) GetAssetsByHash(ctx context.Context, hash string) ([]*Asset, error) {
	query := SearchMetadataQuery{Page: 1, WithExif: true, WithDeleted: true, Checksum: hash}
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
	query := SearchMetadataQuery{Page: 1, WithExif: true, WithDeleted: true, OriginalFileName: name}
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
