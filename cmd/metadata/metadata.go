package metadata

import (
	"context"
	"flag"
	"math"
	"path"
	"strings"
	"time"

	"github.com/simulot/immich-go/helpers/docker"
	"github.com/simulot/immich-go/helpers/myflag"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/immich/metadata"
	"github.com/simulot/immich-go/logger"
)

type MetadataCmd struct {
	Immich                 *immich.ImmichClient // Immich client
	Log                    *logger.Log
	DryRun                 bool
	MissingDateDespiteName bool
	MissingDate            bool
	DockerHost             string
}

func NewMetadataCmd(ctx context.Context, ic *immich.ImmichClient, logger *logger.Log, args []string) (*MetadataCmd, error) {
	var err error
	cmd := flag.NewFlagSet("metadata", flag.ExitOnError)
	app := MetadataCmd{
		Immich: ic,
		Log:    logger,
	}

	cmd.BoolFunc("dry-run", "display actions, but don't touch the server assets", myflag.BoolFlagFn(&app.DryRun, false))
	cmd.BoolFunc("missing-date", "select all assets where the date is missing", myflag.BoolFlagFn(&app.MissingDate, false))
	cmd.BoolFunc("missing-date-with-name", "select all assets where the date is missing but the name contains a the date", myflag.BoolFlagFn(&app.MissingDateDespiteName, false))
	cmd.StringVar(&app.DockerHost, "docker-host", "local", "Immich's docker host where to inject sidecar file as workaround for the issue #3888. 'local' for local connection, 'ssh://user:password@server' for remote host.")
	err = cmd.Parse(args)
	return &app, err
}

func MetadataCommand(ctx context.Context, ic *immich.ImmichClient, log *logger.Log, args []string) error {
	app, err := NewMetadataCmd(ctx, ic, log, args)
	if err != nil {
		return err
	}

	var dockerConn *docker.DockerConnect

	if !app.DryRun {
		dockerConn, err = docker.NewDockerConnection(ctx, app.DockerHost, "immich_server")
		if err != nil {
			return err
		}
		app.Log.OK("Connected to the immich's docker container at %q", app.DockerHost)
	}

	app.Log.MessageContinue(logger.OK, "Get server's assets...")
	list, err := app.Immich.GetAllAssets(ctx, nil)
	if err != nil {
		return err
	}
	app.Log.MessageTerminate(logger.OK, " %d received", len(list))

	type broken struct {
		a *immich.Asset
		metadata.SideCar
		fixable bool
		reason  []string
	}

	now := time.Now().Add(time.Hour * 24)
	brockenAssets := []broken{}
	for _, a := range list {
		ba := broken{a: a}

		if (app.MissingDate) && a.ExifInfo.DateTimeOriginal.IsZero() {
			ba.reason = append(ba.reason, "capture date not set")
		}
		if (app.MissingDate) && (a.ExifInfo.DateTimeOriginal.Year() < 1900 || a.ExifInfo.DateTimeOriginal.Compare(now) > 0) {
			ba.reason = append(ba.reason, "capture date invalid")
		}

		if app.MissingDateDespiteName {
			dt := metadata.TakeTimeFromName(path.Base(a.OriginalPath))
			if !dt.IsZero() {
				if a.ExifInfo.DateTimeOriginal.IsZero() || (math.Abs(float64(dt.Sub(a.ExifInfo.DateTimeOriginal.Time))) > float64(24.0*time.Hour)) {
					ba.reason = append(ba.reason, "capture date invalid, but the name contains a date")
					ba.fixable = true
					ba.SideCar.DateTaken = dt
				}
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

	fixable := 0
	for _, b := range brockenAssets {
		if b.fixable {
			fixable++
		}
		app.Log.OK("%s, (%s %s): %s", b.a.OriginalPath, b.a.ExifInfo.Make, b.a.ExifInfo.Model, strings.Join(b.reason, ", "))
	}
	app.Log.OK("%d broken assets", len(brockenAssets))
	app.Log.OK("Among them, %d can be fixed with current settings", fixable)

	if fixable == 0 {
		return nil
	}

	if app.DryRun {
		log.OK("Dry-run mode. Exiting")
		log.OK("use -dry-run=false after metadata command")
		return nil
	}

	uploader, err := dockerConn.BatchUpload(ctx, "/usr/src/app")
	if err != nil {
		return err
	}

	defer uploader.Close()

	for _, b := range brockenAssets {
		if !b.fixable {
			continue
		}
		a := b.a
		app.Log.MessageContinue(logger.OK, "Uploading sidecar for %s... ", a.OriginalPath)
		scContent, err := b.SideCar.Bytes()
		if err != nil {
			return err
		}
		err = uploader.Upload(a.OriginalPath+".xmp", scContent)
		if err != nil {
			return err
		}
		app.Log.MessageTerminate(logger.OK, "done")
	}
	return nil
}
