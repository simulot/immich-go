package fromimmich

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/simulot/immich-go/commands/application"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/immichfs"
	"github.com/simulot/immich-go/internal/metadata"
)

type FromImmich struct {
	flags  *FromImmichFlags
	client *application.Client
	ifs    *immichfs.ImmichFS

	mustFetchAlbums bool // True if we need to fetch the asset's albums in 2nd step
	errCount        int  // Count the number of errors, to stop after 5
}

func NewFromImmich(ctx context.Context, jnl *fileevent.Recorder, flags *FromImmichFlags) (*FromImmich, error) {
	client := flags.client
	err := client.Initialize(ctx, jnl.Log())
	if err != nil {
		return nil, err
	}
	err = client.Open(ctx)
	if err != nil {
		return nil, err
	}

	ifs := immichfs.NewImmichFS(ctx, flags.client.Server, client.Immich)
	f := FromImmich{
		flags:  flags,
		client: &client,
		ifs:    ifs,
	}
	return &f, nil
}

func (f *FromImmich) Browse(ctx context.Context) chan *assets.Group {
	gOut := make(chan *assets.Group)
	go func() {
		defer close(gOut)
		err := f.getAssets(ctx, gOut)
		if err != nil {
			f.flags.client.ClientLog.Error(fmt.Sprintf("Error while getting Immich assets: %v", err))
		}
	}()
	return gOut
}

const timeFormat = "2006-01-02T15:04:05.000Z"

func (f *FromImmich) getAssets(ctx context.Context, grpChan chan *assets.Group) error {
	query := immich.SearchMetadataQuery{
		Make:  f.flags.Make,
		Model: f.flags.Model,
		// WithExif:     true,
		WithArchived: f.flags.WithArchived,
	}

	f.mustFetchAlbums = true
	if f.flags.DateRange.IsSet() {
		query.TakenAfter = f.flags.DateRange.After.Format(timeFormat)
		query.TakenBefore = f.flags.DateRange.Before.Format(timeFormat)
	}

	return f.client.Immich.GetAllAssetsWithFilter(ctx, &query, func(a *immich.Asset) error {
		if f.flags.Favorite && !a.IsFavorite {
			return nil
		}
		if !f.flags.WithTrashed && a.IsTrashed {
			return nil
		}
		return f.filterAsset(ctx, a, grpChan)
	})
}

func (f *FromImmich) filterAsset(ctx context.Context, a *immich.Asset, grpChan chan *assets.Group) error {
	var err error
	if f.flags.Favorite && !a.IsFavorite {
		return nil
	}

	if !f.flags.WithTrashed && a.IsTrashed {
		return nil
	}

	simplifiedAlbums := a.Albums

	if f.mustFetchAlbums && len(simplifiedAlbums) == 0 {
		simplifiedAlbums, err = f.client.Immich.GetAssetAlbums(ctx, a.ID)
		if err != nil {
			return f.logError(err)
		}
	}
	if len(f.flags.Albums) > 0 && len(simplifiedAlbums) > 0 {
		keepMe := false
		for _, album := range f.flags.Albums {
			for _, aAlbum := range simplifiedAlbums {
				keepMe = keepMe || album == aAlbum.AlbumName
			}
		}
		if !keepMe {
			return nil
		}
	}

	// Some information are missing in the metadata result,
	// so we need to get the asset details

	a, err = f.client.Immich.GetAssetInfo(ctx, a.ID)
	if err != nil {
		return f.logError(err)
	}
	a.Albums = simplifiedAlbums
	asset := a.AsAsset()
	ext := path.Ext(asset.FileName)
	asset.SetNameInfo(filenames.NameInfo{
		Base:    a.OriginalFileName,
		Ext:     ext,
		Radical: strings.TrimSuffix(asset.FileName, ext),
		Type:    metadata.DefaultSupportedMedia.TypeFromExt(ext),
		Taken:   asset.CaptureDate,
	})
	asset.FSys = f.ifs
	asset.FileName = a.ID

	if f.flags.MinimalRating > 0 && a.Rating < f.flags.MinimalRating {
		return nil
	}

	if f.flags.DateRange.IsSet() {
		if asset.CaptureDate.Before(f.flags.DateRange.After) || asset.CaptureDate.After(f.flags.DateRange.Before) {
			return nil
		}
	}

	g := assets.NewGroup(assets.GroupByNone, asset)
	select {
	case grpChan <- g:
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}

func (f *FromImmich) logError(err error) error {
	f.flags.client.ClientLog.Error(fmt.Sprintf("Error while getting Immich assets: %v", err))
	f.errCount++
	if f.errCount > 5 {
		err := errors.New("too many errors, aborting")
		f.flags.client.ClientLog.Error(err.Error())
		return err
	}
	return nil
}
