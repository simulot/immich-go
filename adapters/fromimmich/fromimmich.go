package fromimmich

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/simulot/immich-go/internal/immichfs"
)

type FromImmich struct {
	flags *FromImmichFlags
	// client *app.Client
	ifs *immichfs.ImmichFS
	ic  *filenames.InfoCollector

	albums          []immich.AlbumSimplified
	mustFetchAlbums bool // When there isn't any album as criteria, should check if the asset belongs to an album (true)
	errCount        int  // Count the number of errors, to stop after 5
}

func NewFromImmich(ctx context.Context, app *app.Application, jnl *fileevent.Recorder, flags *FromImmichFlags) (*FromImmich, error) {
	client := &flags.client
	err := client.Initialize(ctx, app)
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
		ic:    filenames.NewInfoCollector(time.Local, client.Immich.SupportedMedia()),
	}
	return &f, nil
}

func (f *FromImmich) Browse(ctx context.Context) chan *assets.Group {
	gOut := make(chan *assets.Group)
	go func() {
		defer close(gOut)
		var err error

		err = f.getAssets(ctx, gOut)
		if err != nil {
			f.flags.client.ClientLog.Error(fmt.Sprintf("Error while getting from-immich assets: %v", err))
		}
	}()
	return gOut
}

const timeFormat = "2006-01-02T15:04:05.000Z"

func (f *FromImmich) getAssets(ctx context.Context, grpChan chan *assets.Group) error {
	var albumsIDs []string
	var err error
	client := f.flags.client.Immich

	if len(f.flags.Albums) > 0 {
		f.albums, err = client.GetAllAlbums(ctx)
		if err != nil {
			return err
		}
		for _, fromAlbum := range f.flags.Albums {
			for _, a := range f.albums {
				if a.AlbumName == fromAlbum {
					albumsIDs = append(albumsIDs, a.ID)
				}
			}
		}
	}

	query := immich.SearchMetadataQuery{
		Make:       f.flags.Make,
		Model:      f.flags.Model,
		WithExif:   true,
		IsFavorite: f.flags.Favorite,
		AlbumIds:   albumsIDs,
		// WithArchived: f.flags.WithArchived,
	}

	f.mustFetchAlbums = true
	if f.flags.DateRange.IsSet() {
		query.TakenAfter = f.flags.DateRange.After.Format(timeFormat)
		query.TakenBefore = f.flags.DateRange.Before.Format(timeFormat)
	}

	return f.flags.client.Immich.GetAllAssetsWithFilter(ctx, &query, func(a *immich.Asset) error {
		// apply filters that don't fit in the immch search api
		if f.flags.MinimalRating > 0 && a.Rating < f.flags.MinimalRating {
			return nil
		}
		if f.flags.DateRange.IsSet() {
			if !f.flags.DateRange.InRange(a.ExifInfo.DateTimeOriginal.Time) {
				return nil
			}
		}

		asset := a.AsAsset()
		asset.SetNameInfo(f.ic.GetInfo(asset.OriginalFileName))
		asset.File = fshelper.FSName(f.ifs, a.ID)

		asset.FromApplication = &assets.Metadata{
			FileName:    a.OriginalFileName,
			FileDate:    a.FileModifiedAt.Time,
			Latitude:    a.ExifInfo.Latitude,
			Longitude:   a.ExifInfo.Longitude,
			Description: a.ExifInfo.Description,
			DateTaken:   a.ExifInfo.DateTimeOriginal.Time,
			Trashed:     a.IsTrashed,
			Archived:    a.IsArchived,
			Favorited:   a.IsFavorite,
			Rating:      byte(a.Rating),
			Tags:        asset.Tags,
		}

		simplifiedA, err := f.flags.client.Immich.GetAssetAlbums(ctx, a.ID)
		if err != nil {
			return f.logError(err)
		}

		albums := immich.AlbumsFromAlbumSimplified(simplifiedA)
		// clear the ID of the album that exists in from server, but not in to server
		for i := range albums {
			albums[i].ID = ""
		}
		asset.FromApplication.Albums = albums
		asset.Albums = albums

		g := assets.NewGroup(assets.GroupByNone, asset)
		select {
		case grpChan <- g:
		case <-ctx.Done():
			return ctx.Err()
		}
		return nil
	})
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
