/*
Check the list of photos to list and discard duplicates.
*/
package duplicate

import (
	"context"
	"flag"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/simulot/immich-go/helpers/gen"
	"github.com/simulot/immich-go/helpers/myflag"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/logger"
	"github.com/simulot/immich-go/ui"
)

type DuplicateCmd struct {
	logger *logger.Log
	Immich *immich.ImmichClient // Immich client

	AssumeYes      bool             // When true, doesn't ask to the user
	DateRange      immich.DateRange // Set capture date range
	IgnoreTZErrors bool             // Enable TZ error tolerance

	assetsByID          map[string]*immich.Asset
	assetsByBaseAndDate map[duplicateKey][]*immich.Asset
}

type duplicateKey struct {
	Date time.Time
	Name string
}

func NewDuplicateCmd(ctx context.Context, ic *immich.ImmichClient, logger *logger.Log, args []string) (*DuplicateCmd, error) {
	cmd := flag.NewFlagSet("duplicate", flag.ExitOnError)
	validRange := immich.DateRange{}
	_ = validRange.Set("1850-01-04,2030-01-01")
	app := DuplicateCmd{
		logger:              logger,
		Immich:              ic,
		DateRange:           validRange,
		assetsByID:          map[string]*immich.Asset{},
		assetsByBaseAndDate: map[duplicateKey][]*immich.Asset{},
	}

	cmd.BoolFunc("ignore-tz-errors", "Ignore timezone difference to check duplicates (default: FALSE).", myflag.BoolFlagFn(&app.IgnoreTZErrors, false))
	cmd.BoolFunc("yes", "When true, assume Yes to all actions", myflag.BoolFlagFn(&app.AssumeYes, false))
	cmd.Var(&app.DateRange, "date", "Process only documents having a capture date in that range.")
	err := cmd.Parse(args)
	return &app, err
}

func DuplicateCommand(ctx context.Context, ic *immich.ImmichClient, log *logger.Log, args []string) error {
	app, err := NewDuplicateCmd(ctx, ic, log, args)
	if err != nil {
		return err
	}

	dupCount := 0
	log.MessageContinue(logger.OK, "Get server's assets...")
	err = app.Immich.GetAllAssetsWithFilter(ctx, nil, func(a *immich.Asset) {
		if a.IsTrashed {
			return
		}
		if !app.DateRange.InRange(a.ExifInfo.DateTimeOriginal.Time) {
			return
		}
		app.assetsByID[a.ID] = a
		d := a.ExifInfo.DateTimeOriginal.Time.Round(time.Minute)
		if app.IgnoreTZErrors {
			d = time.Date(d.Year(), d.Month(), d.Day(), 0, d.Minute(), d.Second(), 0, time.UTC)
		}
		k := duplicateKey{
			Date: d,
			Name: strings.ToUpper(a.OriginalFileName + path.Ext(a.OriginalPath)),
		}

		l := app.assetsByBaseAndDate[k]
		if len(l) > 0 {
			dupCount++
		}
		app.assetsByBaseAndDate[k] = append(l, a)
	})
	if err != nil {
		return err
	}
	log.MessageTerminate(logger.OK, "%d received", len(app.assetsByID))
	log.MessageTerminate(logger.OK, "%d duplicate(s) determined.", dupCount)

	keys := gen.MapFilterKeys(app.assetsByBaseAndDate, func(i []*immich.Asset) bool {
		return len(i) > 1
	})
	sort.Slice(keys, func(i, j int) bool {
		c := keys[i].Date.Compare(keys[j].Date)
		switch c {
		case -1:
			return true
		case +1:
			return false
		}
		c = strings.Compare(keys[i].Name, keys[j].Name)

		return c == -1
	})

	for _, k := range keys {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			l := app.assetsByBaseAndDate[k]
			app.logger.OK("There are %d copies of the asset %s, taken on %s ", len(l), k.Name, l[0].ExifInfo.DateTimeOriginal.Format(time.RFC3339))
			albums := []immich.AlbumSimplified{}
			assetsToDelete := []string{}
			sort.Slice(l, func(i, j int) bool { return l[i].ExifInfo.FileSizeInByte < l[j].ExifInfo.FileSizeInByte })
			for p, a := range l {
				if p < len(l)-1 {
					log.OK("  delete %s %dx%d, %s, %s", a.OriginalFileName, a.ExifInfo.ExifImageWidth, a.ExifInfo.ExifImageHeight, ui.FormatBytes(a.ExifInfo.FileSizeInByte), a.OriginalPath)
					assetsToDelete = append(assetsToDelete, a.ID)
					r, err := app.Immich.GetAssetAlbums(ctx, a.ID)
					if err != nil {
						log.Error("Can't get asset's albums: %s", err.Error())
					} else {
						albums = append(albums, r...)
					}
				} else {
					log.OK("  keep   %s %dx%d, %s, %s", a.OriginalFileName, a.ExifInfo.ExifImageWidth, a.ExifInfo.ExifImageHeight, ui.FormatBytes(a.ExifInfo.FileSizeInByte), a.OriginalPath)
					yes := app.AssumeYes
					if !app.AssumeYes {
						r, err := ui.ConfirmYesNo(ctx, "Proceed?", "n")
						if err != nil {
							return err
						}
						if r == "y" {
							yes = true
						}
					}
					if yes {
						err = app.Immich.DeleteAssets(ctx, assetsToDelete, false)
						if err != nil {
							log.Error("Can't delete asset: %s", err.Error())
						} else {
							log.OK("  Asset removed")
							for _, al := range albums {
								log.OK("  Update the album %s with the best copy", al.AlbumName)
								_, err = app.Immich.AddAssetToAlbum(ctx, al.ID, []string{a.ID})
								if err != nil {
									log.Error("Can't delete asset: %s", err.Error())
								}
							}
						}
					}
				}
			}
		}
	}
	return nil
}
