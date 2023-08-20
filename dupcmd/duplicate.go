/*
Check the list of photos to list and discard duplicates.
*/
package dupcmd

import (
	"context"
	"flag"
	"immich-go/immich"
	"immich-go/immich/logger"
	"sort"
	"strings"
	"time"
)

type DuplicateCmd struct {
	logger *logger.Logger
	Immich *immich.ImmichClient // Immich client

	DryRun    bool             // Display actions but don't change anything
	DateRange immich.DateRange // Set capture date range
}

type duplicateKey struct {
	Date time.Time
	Name string
}

func NewDuplicateCmd(ctx context.Context, ic *immich.ImmichClient, logger *logger.Logger, args []string) (*DuplicateCmd, error) {
	cmd := flag.NewFlagSet("upload", flag.ExitOnError)
	validRange := immich.DateRange{}
	validRange.Set("1850-01-04,3000-01-01")
	app := DuplicateCmd{
		logger:    logger,
		Immich:    ic,
		DateRange: validRange,
	}

	cmd.BoolVar(&app.DryRun, "dry-run", true, "display actions but don't touch source or destination")
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

	log.MessageContinue(logger.OK, "Analyzing...")
	duplicate := map[duplicateKey][]*immich.Asset{}

	count := 0
	dupCount := 0
	for _, a := range list {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			count++
			if app.DateRange.InRange(a.ExifInfo.DateTimeOriginal) {
				k := duplicateKey{
					Date: a.ExifInfo.DateTimeOriginal,
					Name: a.OriginalFileName,
				}
				l := duplicate[k]
				if len(l) > 0 {
					dupCount++
				}
				l = append(l, a)
				duplicate[k] = l
			}
			if count%253 == 0 {
				log.Progress("%d medias, %d duplicate(s)...", count, dupCount)
			}
		}
	}
	log.Progress("%d medias, %d duplicate(s)...", count, dupCount)
	log.MessageTerminate(logger.OK, " analyze completed.")

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
			app.logger.Info("%s:", k.Date.Format(time.RFC3339))
			l := duplicate[k]
			albums := []string{}
			delete := []string{}
			sort.Slice(l, func(i, j int) bool { return l[i].ExifInfo.FileSizeInByte < l[j].ExifInfo.FileSizeInByte })
			for p, a := range duplicate[k] {
				if p < len(l)-1 {
					log.Info("   %s(%s) %dx%d, %d bytes: delete", a.OriginalFileName, a.ID, a.ExifInfo.ExifImageWidth, a.ExifInfo.ExifImageHeight, a.ExifInfo.FileSizeInByte)
					delete = append(delete, a.ID)
					r, err := app.Immich.GetAssetAlbums(ctx, a.ID)
					if err != nil {
						log.Error("Can't get asset's albums: %s", err.Error())
					} else {
						for _, al := range r {
							albums = append(albums, al.ID)
						}
					}
				} else {
					log.Info("   %s(%s) %dx%d, %d bytes: keep", a.OriginalFileName, a.ID, a.ExifInfo.ExifImageWidth, a.ExifInfo.ExifImageHeight, a.ExifInfo.FileSizeInByte)
					if !app.DryRun {
						log.OK("Deleting following assets: %s", strings.Join(delete, ","))
						_, err = app.Immich.DeleteAssets(ctx, delete)
						if err != nil {
							log.Error("Can't delete asset: %s", err.Error())
						}
					} else {
						log.Info("Skip deleting following %s, dry run mode", strings.Join(delete, ","))
					}
					for _, al := range albums {
						if !app.DryRun {
							log.OK("Adding %s to album %s", a.ID, al)
							_, err = app.Immich.AddAssetToAlbum(ctx, al, []string{a.ID})
							if err != nil {
								log.Error("Can't delete asset: %s", err.Error())
							}
						} else {
							log.OK("Skip Adding %s to album %s, dry run mode", a.ID, al)
						}
					}
				}
			}
		}
	}
	return nil
}
