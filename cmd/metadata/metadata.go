package metadata

import (
	"context"
	"flag"
	"fmt"
	"math"
	"path"
	"strings"
	"time"

	"github.com/simulot/immich-go/cmd"
	"github.com/simulot/immich-go/helpers/docker"
	"github.com/simulot/immich-go/helpers/myflag"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/immich/metadata"
)

type MetadataCmd struct {
	*cmd.SharedFlags
	DryRun                 bool
	MissingDateDespiteName bool
	MissingDate            bool
	DockerHost             string
}

func NewMetadataCmd(ctx context.Context, common *cmd.SharedFlags, args []string) (*MetadataCmd, error) {
	var err error
	cmd := flag.NewFlagSet("metadata", flag.ExitOnError)
	app := MetadataCmd{
		SharedFlags: common,
	}

	app.SharedFlags.SetFlags(cmd)
	cmd.BoolFunc("dry-run", "display actions, but don't touch the server assets", myflag.BoolFlagFn(&app.DryRun, false))
	cmd.BoolFunc("missing-date", "select all assets where the date is missing", myflag.BoolFlagFn(&app.MissingDate, false))
	cmd.BoolFunc("missing-date-with-name", "select all assets where the date is missing but the name contains a the date", myflag.BoolFlagFn(&app.MissingDateDespiteName, false))
	cmd.StringVar(&app.DockerHost, "docker-host", "local", "Immich's docker host where to inject sidecar file as workaround for the issue #3888. 'local' for local connection, 'ssh://user:password@server' for remote host.")
	err = cmd.Parse(args)
	if err != nil {
		return nil, err
	}
	err = app.SharedFlags.Start(ctx)
	return &app, err
}

func MetadataCommand(ctx context.Context, common *cmd.SharedFlags, args []string) error {
	app, err := NewMetadataCmd(ctx, common, args)
	if err != nil {
		return err
	}

	var dockerConn *docker.DockerConnect

	if !app.DryRun {
		dockerConn, err = docker.NewDockerConnection(ctx, app.DockerHost, "immich_server")
		if err != nil {
			return err
		}
		fmt.Printf("Connected to the immich's docker container at %q\n", app.DockerHost)
	}

	fmt.Println("Get server's assets...")
	list, err := app.Immich.GetAllAssets(ctx)
	if err != nil {
		return err
	}
	fmt.Printf(" %d received\n", len(list))

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
			dt := metadata.TimeFromFullPath(path.Base(a.OriginalPath))
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
		fmt.Printf("%s, (%s %s): %s\n", b.a.OriginalPath, b.a.ExifInfo.Make, b.a.ExifInfo.Model, strings.Join(b.reason, ", "))
	}
	fmt.Printf("%d broken assets\n", len(brockenAssets))
	fmt.Printf("Among them, %d can be fixed with current settings\n", fixable)

	if fixable == 0 {
		return nil
	}

	if app.DryRun {
		fmt.Println("Dry-run mode. Exiting")
		fmt.Println("use -dry-run=false after metadata command")
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
		fmt.Printf("Uploading sidecar for %s... \n", a.OriginalPath)
		scContent, err := b.SideCar.Bytes()
		if err != nil {
			return err
		}
		err = uploader.Upload(a.OriginalPath+".xmp", scContent)
		if err != nil {
			return err
		}
		fmt.Println("done")
	}
	return nil
}
