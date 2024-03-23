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

	"github.com/simulot/immich-go/cmd"
	"github.com/simulot/immich-go/helpers/gen"
	"github.com/simulot/immich-go/helpers/myflag"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/logger"
	"github.com/simulot/immich-go/ui"
)

type DuplicateCmd struct {
	*cmd.SharedFlags
	AssumeYes       bool             // When true, doesn't ask to the user
	DateRange       immich.DateRange // Set capture date range
	IgnoreTZErrors  bool             // Enable TZ error tolerance
	IgnoreExtension bool             // Ignore file extensions when checking for duplicates

	assetsByID          map[string]*immich.Asset
	assetsByBaseAndDate map[duplicateKey][]*immich.Asset
}

type duplicateKey struct {
	Date time.Time
	Name string
	Type string
}

func NewDuplicateCmd(ctx context.Context, common *cmd.SharedFlags, args []string) (*DuplicateCmd, error) {
	cmd := flag.NewFlagSet("duplicate", flag.ExitOnError)
	validRange := immich.DateRange{}
	_ = validRange.Set("1850-01-04,2030-01-01")
	app := DuplicateCmd{
		SharedFlags:         common,
		DateRange:           validRange,
		assetsByID:          map[string]*immich.Asset{},
		assetsByBaseAndDate: map[duplicateKey][]*immich.Asset{},
	}

	app.SharedFlags.SetFlags(cmd)

	cmd.BoolFunc("ignore-tz-errors", "Ignore timezone difference to check duplicates (default: FALSE).", myflag.BoolFlagFn(&app.IgnoreTZErrors, false))
	cmd.BoolFunc("yes", "When true, assume Yes to all actions", myflag.BoolFlagFn(&app.AssumeYes, false))
	cmd.Var(&app.DateRange, "date", "Process only documents having a capture date in that range.")
	cmd.BoolFunc("ignore-extension", "When true, ignores extensions when checking for duplicates (default: FALSE)", myflag.BoolFlagFn(&app.IgnoreExtension, false))
	err := cmd.Parse(args)
	if err != nil {
		return nil, err
	}
	err = app.SharedFlags.Start(ctx)
	if err != nil {
		return nil, err
	}
	return &app, err
}

func DuplicateCommand(ctx context.Context, common *cmd.SharedFlags, args []string) error {
	app, err := NewDuplicateCmd(ctx, common, args)
	if err != nil {
		return err
	}

	dupCount := 0
	app.Jnl.Log.MessageContinue(logger.OK, "Get server's assets...")
	err = app.Immich.GetAllAssetsWithFilter(ctx, func(a *immich.Asset) {
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
			Type: a.Type,
		}

		if app.IgnoreExtension {
			k.Name = strings.TrimSuffix(k.Name, path.Ext(a.OriginalPath))
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
	app.Jnl.Log.MessageTerminate(logger.OK, "%d received", len(app.assetsByID))
	app.Jnl.Log.MessageTerminate(logger.OK, "%d duplicate(s) determined.", dupCount)

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
			app.Jnl.Log.OK("There are %d copies of the asset %s, taken on %s ", len(l), k.Name, l[0].ExifInfo.DateTimeOriginal.Format(time.RFC3339))
			albums := []immich.AlbumSimplified{}
			assetsToDelete := []string{}
			sort.Slice(l, func(i, j int) bool { return l[i].ExifInfo.FileSizeInByte < l[j].ExifInfo.FileSizeInByte })
			for p, a := range l {
				if p < len(l)-1 {
					app.Jnl.Log.OK("  delete %s %dx%d, %s, %s", a.OriginalFileName, a.ExifInfo.ExifImageWidth, a.ExifInfo.ExifImageHeight, ui.FormatBytes(a.ExifInfo.FileSizeInByte), a.OriginalPath)
					assetsToDelete = append(assetsToDelete, a.ID)
					r, err := app.Immich.GetAssetAlbums(ctx, a.ID)
					if err != nil {
						app.Jnl.Log.Error("Can't get asset's albums: %s", err.Error())
					} else {
						albums = append(albums, r...)
					}
				} else {
					app.Jnl.Log.OK("  keep   %s %dx%d, %s, %s", a.OriginalFileName, a.ExifInfo.ExifImageWidth, a.ExifInfo.ExifImageHeight, ui.FormatBytes(a.ExifInfo.FileSizeInByte), a.OriginalPath)
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
							app.Jnl.Log.Error("Can't delete asset: %s", err.Error())
						} else {
							app.Jnl.Log.OK("  Asset removed")
							for _, al := range albums {
								app.Jnl.Log.OK("  Update the album %s with the best copy", al.AlbumName)
								_, err = app.Immich.AddAssetToAlbum(ctx, al.ID, []string{a.ID})
								if err != nil {
									app.Jnl.Log.Error("Can't delete asset: %s", err.Error())
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
