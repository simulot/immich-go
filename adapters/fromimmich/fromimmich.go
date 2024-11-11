package fromimmich

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/immichfs"
	"github.com/simulot/immich-go/internal/metadata"
)

type FromImmich struct {
	flags *FromImmichFlags
	// client *application.Client
	ifs *immichfs.ImmichFS

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
		flags: flags,
		ifs:   ifs,
	}
	return &f, nil
}

func (f *FromImmich) Browse(ctx context.Context) chan *assets.Group {
	gOut := make(chan *assets.Group)
	go func() {
		defer close(gOut)
		var err error
		switch {
		case len(f.flags.Albums) > 0:
			err = f.getAssetsFromAlbums(ctx, gOut)
		default:
			err = f.getAssets(ctx, gOut)
		}
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

	return f.flags.client.Immich.GetAllAssetsWithFilter(ctx, &query, func(a *immich.Asset) error {
		if f.flags.Favorite && !a.IsFavorite {
			return nil
		}
		if !f.flags.WithTrashed && a.IsTrashed {
			return nil
		}
		return f.filterAsset(ctx, a, grpChan)
	})
}

func (f *FromImmich) getAssetsFromAlbums(ctx context.Context, grpChan chan *assets.Group) error {
	f.mustFetchAlbums = false

	assets := map[string]*immich.Asset{} // List of assets to get by ID

	albums, err := f.flags.client.Immich.GetAllAlbums(ctx)
	if err != nil {
		return f.logError(err)
	}
	for _, album := range albums {
		for _, albumName := range f.flags.Albums {
			if album.AlbumName == albumName {
				al, err := f.flags.client.Immich.GetAlbumInfo(ctx, album.ID, false)
				if err != nil {
					return f.logError(err)
				}
				for _, a := range al.Assets {
					if _, ok := assets[a.ID]; !ok {
						a.Albums = append(a.Albums, immich.AlbumSimplified{
							AlbumName: album.AlbumName,
						})
						assets[a.ID] = a
					} else {
						assets[a.ID].Albums = append(assets[a.ID].Albums, immich.AlbumSimplified{
							AlbumName: album.AlbumName,
						})
					}
				}
			}
		}
	}

	for _, a := range assets {
		err = f.filterAsset(ctx, a, grpChan)
		if err != nil {
			return f.logError(err)
		}
	}
	return nil
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
		simplifiedAlbums, err = f.flags.client.Immich.GetAssetAlbums(ctx, a.ID)
		if err != nil {
			return f.logError(err)
		}
	}
	if len(f.flags.Albums) > 0 && len(simplifiedAlbums) > 0 {
		keepMe := false
		newAlbumList := []immich.AlbumSimplified{}
		for _, album := range f.flags.Albums {
			for _, aAlbum := range simplifiedAlbums {
				if album == aAlbum.AlbumName {
					keepMe = true
					newAlbumList = append(newAlbumList, aAlbum)
				}
			}
		}
		if !keepMe {
			return nil
		}
		simplifiedAlbums = newAlbumList
	}

	// Some information are missing in the metadata result,
	// so we need to get the asset details

	a, err = f.flags.client.Immich.GetAssetInfo(ctx, a.ID)
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
	g.Albums = asset.Albums
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
