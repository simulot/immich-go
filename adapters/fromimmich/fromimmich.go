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

	// albums   []immich.AlbumSimplified
	// tags     []immich.TagSimplified
	errCount int // Count the number of errors, to stop after 5
}

func NewFromImmich(ctx context.Context, app *app.Application, jnl *fileevent.Recorder, flags *FromImmichFlags) (*FromImmich, error) {
	client := &flags.client

	err := client.Open(ctx, app)
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

		err := f.getAssets(ctx, gOut)
		if err != nil {
			f.flags.client.ClientLog.Error(fmt.Sprintf("Error while getting from-immich assets: %v", err))
		}
	}()
	return gOut
}

func (f *FromImmich) getAssets(ctx context.Context, grpChan chan *assets.Group) error {
	// todo implement from-album and from-tag

	// var albumsIDs []string
	// var tagsIds []string
	// var err error
	// client := f.flags.client.Immich

	// if len(f.flags.Albums) > 0 {
	// 	f.albums, err = client.GetAllAlbums(ctx)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	for _, fromAlbum := range f.flags.Albums {
	// 		for _, a := range f.albums {
	// 			if a.AlbumName == fromAlbum {
	// 				albumsIDs = append(albumsIDs, a.ID)
	// 			}
	// 		}
	// 	}
	// }

	// if len(f.flags.Tags) > 0 {
	// 	f.tags, err = client.GetAllTags(ctx)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	for _, fromTag := range f.tags {
	// 		for _, t := range f.flags.Tags {
	// 			if t == fromTag.Value {
	// 				tagsIds = append(tagsIds, fromTag.ID)
	// 			}
	// 		}
	// 	}
	// }

	f.flags.MinimalRating = min(max(0, f.flags.MinimalRating), 5)

	so := immich.SearchOptions()

	if !f.flags.OnlyArchived && !f.flags.OnlyTrashed && !f.flags.OnlyFavorite {
		so.All()
	} else {
		if f.flags.OnlyArchived {
			so.WithOnlyArchived()
		}
		if f.flags.OnlyTrashed {
			so.WithOnlyTrashed()
		}
		if f.flags.OnlyFavorite {
			so.WithOnlyFavorite()
		}
		if f.flags.DateRange.IsSet() {
			so.WithDateRange(f.flags.DateRange)
		}
	}

	if f.flags.DateRange.IsSet() {
		so.WithDateRange(f.flags.DateRange)
	}

	if f.flags.MinimalRating > 1 {
		so.WithMinimalRate(f.flags.MinimalRating)
	}
	return f.flags.client.Immich.GetFilteredAssetsFn(ctx, so, func(a *immich.Asset) error {
		// Fetch details
		a, err := f.flags.client.Immich.GetAssetInfo(ctx, a.ID)
		if err != nil {
			return f.logError(err)
		}

		asset := a.AsAsset()
		asset.FromApplication = &assets.Metadata{
			FileName:    a.OriginalFileName,
			Latitude:    a.ExifInfo.Latitude,
			Longitude:   a.ExifInfo.Longitude,
			Description: a.ExifInfo.Description,
			DateTaken:   a.ExifInfo.DateTimeOriginal.Time,
			Trashed:     a.IsTrashed,
			Archived:    a.IsArchived,
			Favorited:   a.IsFavorite,
			Rating:      byte(a.ExifInfo.Rating),
			Tags:        asset.Tags,
		}
		asset.UseMetadata(asset.FromApplication)
		asset.File = fshelper.FSName(f.ifs, a.ID)

		// Transfer the album
		simplifiedA, err := f.flags.client.Immich.GetAssetAlbums(ctx, a.ID)
		if err != nil {
			return f.logError(err)
		}
		albums := immich.AlbumsFromAlbumSimplified(simplifiedA)
		// clear the ID of the album that exists in from server, but not in to server
		for i := range albums {
			albums[i].ID = ""
		}

		asset.Albums = albums

		// Transfer tags
		for t := range asset.Tags {
			asset.Tags[t].ID = ""
		}

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
