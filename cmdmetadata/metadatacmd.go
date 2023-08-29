package cmdmetadata

import (
	"context"
	"flag"
	"fmt"
	"immich-go/immich"
	"immich-go/immich/logger"
	"immich-go/immich/metadata"
	"path"
)

type MetadataCmd struct {
	Immich                 *immich.ImmichClient // Immich client
	Log                    *logger.Logger
	DryRun                 bool
	MissingDateDespiteName bool
	MissingDate            bool
}

func NewMetadataCmd(ctx context.Context, ic *immich.ImmichClient, logger *logger.Logger, args []string) (*MetadataCmd, error) {
	var err error
	cmd := flag.NewFlagSet("metadata", flag.ExitOnError)
	app := MetadataCmd{
		Immich: ic,
		Log:    logger,
	}

	cmd.BoolVar(&app.DryRun, "dry-run", true, "display actions, but don't touch the server assets")
	cmd.BoolVar(&app.MissingDate, "missing-date", false, "select all assets where the date is missing")
	cmd.BoolVar(&app.MissingDateDespiteName, "missing-date-with-name", false, "select all assets where the date is missing ut the name contains a the date")
	err = cmd.Parse(args)
	return &app, err
}

func MetadataCommand(ctx context.Context, ic *immich.ImmichClient, log *logger.Logger, args []string) error {
	app, err := NewMetadataCmd(ctx, ic, log, args)
	if err != nil {
		return err
	}

	app.Log.MessageContinue(logger.OK, "Get server's assets...")
	list, err := app.Immich.GetAllAssets(ctx, nil)
	if err != nil {
		return err
	}
	app.Log.MessageTerminate(logger.OK, " %d received", len(list))

	type broken struct {
		a      *immich.Asset
		reason []string
	}
	brockenAssets := []broken{}
	for _, a := range list {
		ba := broken{a: a}

		if (app.MissingDate) && a.ExifInfo.DateTimeOriginal == nil {
			ba.reason = append(ba.reason, "capture date not set")
		}
		if (app.MissingDate) && a.ExifInfo.DateTimeOriginal != nil && a.ExifInfo.DateTimeOriginal.Year() < 1900 {
			ba.reason = append(ba.reason, "capture date invalid")
		}

		if app.MissingDateDespiteName && (a.ExifInfo.DateTimeOriginal == nil || (a.ExifInfo.DateTimeOriginal != nil && a.ExifInfo.DateTimeOriginal.Year() < 1900)) {
			if !metadata.TakeTimeFromName(path.Base(a.OriginalPath)).IsZero() {
				ba.reason = append(ba.reason, "capture date invalid, but the name contains a date")
			}

		}
		/*
			if a.ExifInfo.Latitude == nil || a.ExifInfo.Longitude == nil {
				ba.reason = append(ba.reason, "GPS coordinates not set")
			} else if math.Abs(*a.ExifInfo.Latitude) < 0.00001 && math.Abs(*a.ExifInfo.Longitude) < 0.00001 {
				ba.reason = append(ba.reason, "GPS coordinates is near of 0;0")
			}
		*/
		if len(ba.reason) > 0 {
			brockenAssets = append(brockenAssets, ba)
		}
	}

	for _, b := range brockenAssets {
		fmt.Printf("docker -H ssh://root@192.168.10.17 cp 'immich_server:/usr/src/app/%s' '%s'\n", b.a.OriginalPath, path.Base(b.a.OriginalPath))
		/*
			app.Log.Message(logger.OK, "%s", b.a.OriginalPath)
				if s := strings.Join([]string{b.a.ExifInfo.Make, b.a.ExifInfo.Model}, " "); s != "" {
					app.Log.MessageContinue(logger.OK, ", %s", s)
				}
				app.Log.MessageTerminate(logger.OK, ", %s", strings.Join(b.reason, ", "))
		*/
	}
	app.Log.OK("%d broken assets", len(brockenAssets))
	if app.DryRun {
		log.OK("Dry-run mode. Exiting")
		return nil
	}
	return nil
}
