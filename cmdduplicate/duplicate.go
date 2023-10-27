/*
Check the list of photos to list and discard duplicates.
*/
package cmdduplicate

import (
	"context"
	"flag"
	"immich-go/immich"
	"immich-go/immich/logger"
	"immich-go/ui"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)

type DuplicateCmd struct {
	logger *logger.Logger
	Immich *immich.ImmichClient // Immich client

	AssumeYes bool             // Display actions but don't change anything
	DateRange immich.DateRange // Set capture date range
}

type duplicateKey struct {
	Date time.Time
	Name string
}

func NewDuplicateCmd(ctx context.Context, ic *immich.ImmichClient, logger *logger.Logger, args []string) (*DuplicateCmd, error) {
	cmd := flag.NewFlagSet("duplicate", flag.ExitOnError)
	validRange := immich.DateRange{}
	validRange.Set("1850-01-04,2030-01-01")
	app := DuplicateCmd{
		logger:    logger,
		Immich:    ic,
		DateRange: validRange,
	}

	cmd.BoolFunc("yes", "When true, assume Yes to all actions", func(s string) error {
		var err error
		app.AssumeYes, err = strconv.ParseBool(s)
		return err
	})
	cmd.Var(&app.DateRange, "date", "Process only document having a	capture date in that range.")
	err := cmd.Parse(args)
	return &app, err
}

func DuplicateCommand(ctx context.Context, ic *immich.ImmichClient, log *logger.Logger, args []string) error {
	app, err := NewDuplicateCmd(ctx, ic, log, args)
	if err != nil {
		return err
	}

	log.MessageContinue(logger.OK, "Get server's assets...")
	var list []*immich.Asset
	list, err = app.Immich.GetAllAssets(ctx, nil)
	if err != nil {
		return err
	}
	log.MessageTerminate(logger.OK, "%d received", len(list))

	log.MessageContinue(logger.Info, "Analyzing...")
	duplicate := map[duplicateKey][]*immich.Asset{}

	count := 0
	dupCount := 0
	for _, a := range list {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if a.IsTrashed {
				continue
			}
			count++
			if app.DateRange.InRange(a.ExifInfo.DateTimeOriginal.Time) {
				k := duplicateKey{
					Date: a.ExifInfo.DateTimeOriginal.Time,
					Name: strings.ToUpper(a.OriginalFileName + path.Ext(a.OriginalPath)),
				}
				l := duplicate[k]
				if len(l) > 0 {
					dupCount++
				}
				l = append(l, a)
				duplicate[k] = l
			}
			if true || count%253 == 0 {
				log.Progress(logger.Info, "%d medias, %d duplicate(s)...", count, dupCount)
			}
		}
	}
	log.MessageTerminate(logger.OK, "%d medias, %d duplicate(s). Analyze completed.", count, dupCount)

	keys := []duplicateKey{}
	for k, l := range duplicate {
		if len(l) < 2 {
			continue
		}
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		switch keys[i].Date.Compare(keys[j].Date) {
		case -1:
			return true
		case +1:
			return false
		}
		return strings.Compare(keys[i].Name, keys[j].Name) == -1
	})

	for _, k := range keys {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			app.logger.OK("There are %d copies of the asset %s, taken on %s ", len(duplicate[k]), k.Name, k.Date.Format(time.RFC3339))
			l := duplicate[k]
			albums := []immich.AlbumSimplified{}
			delete := []string{}
			sort.Slice(l, func(i, j int) bool { return l[i].ExifInfo.FileSizeInByte < l[j].ExifInfo.FileSizeInByte })
			for p, a := range duplicate[k] {
				if p < len(l)-1 {
					log.OK("  delete %s %dx%d, %s, %s", a.OriginalFileName, a.ExifInfo.ExifImageWidth, a.ExifInfo.ExifImageHeight, ui.FormatBytes(a.ExifInfo.FileSizeInByte), a.OriginalPath)
					delete = append(delete, a.ID)
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
						_, err = app.Immich.DeleteAssets(ctx, delete)
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
