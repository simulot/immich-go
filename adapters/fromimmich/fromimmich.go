package fromimmich

import (
	"context"
	"fmt"
	"time"

	"github.com/simulot/immich-go/commands/application"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/fileevent"
)

type FromImmich struct {
	flags  *FromImmichFlags
	client *application.Client
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
	f := FromImmich{
		flags:  flags,
		client: &client,
	}
	return &f, nil
}

func (f *FromImmich) Browse(ctx context.Context) chan *assets.Group {
	gOut := make(chan *assets.Group)
	go func() {
		defer close(gOut)
		err := f.getWithoutTags(ctx, gOut)
		if err != nil {
			f.flags.client.ClientLog.Error(fmt.Sprintf("Error while getting Immich assets: %v", err))
		}
	}()
	return gOut
}

const timeFormat = "2006-01-02T15:04:05.000Z"

func (f *FromImmich) getWithoutTags(ctx context.Context, grpChan chan *assets.Group) error {
	query := immich.SearchMetadataQuery{
		Make:         f.flags.Make,
		Model:        f.flags.Model,
		WithArchived: f.flags.WithArchived,
	}

	if f.flags.WithTrashed {
		query.TrashedAfter = time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC).Format(timeFormat)
	}

	if f.flags.DateRange.IsSet() {
		query.TakenAfter = f.flags.DateRange.After.Format(timeFormat)
		query.TakenBefore = f.flags.DateRange.Before.Format(timeFormat)
	}

	return f.client.Immich.GetAllAssetsWithFilter(ctx, &query, func(a *immich.Asset) error {
		if f.flags.Favorite && !a.IsFavorite {
			return nil
		}

		if f.flags.MinimalRating > 0 && a.Rating < f.flags.MinimalRating {
			return nil
		}

		if len(f.flags.Albums) > 0 {
			keepMe := false
			for _, album := range f.flags.Albums {
				for _, aAlbum := range a.Albums {
					keepMe = keepMe || album == aAlbum.AlbumName
				}
			}
			if !keepMe {
				return nil
			}
		}
		g := assets.NewGroup(assets.GroupByNone, a.AsAsset())
		select {
		case grpChan <- g:
		case <-ctx.Done():
			return ctx.Err()
		}
		return nil
	})
}
