package album

import (
	"context"
	"flag"
	"fmt"
	"regexp"
	"sort"
	"strconv"

	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/logger"
	"github.com/simulot/immich-go/ui"
)

func AlbumCommand(ctx context.Context, ic *immich.ImmichClient, log *logger.Log, args []string) error {
	if len(args) > 0 {
		cmd := args[0]
		args = args[1:]

		if cmd == "delete" {
			return deleteAlbum(ctx, ic, log, args)
		}
	}
	return fmt.Errorf("tool album need a command: delete")
}

type DeleteAlbumCmd struct {
	log       *logger.Log
	Immich    *immich.ImmichClient // Immich client
	pattern   *regexp.Regexp       // album pattern
	AssumeYes bool
}

func deleteAlbum(ctx context.Context, ic *immich.ImmichClient, log *logger.Log, args []string) error {
	app := &DeleteAlbumCmd{
		log:    log,
		Immich: ic,
	}
	cmd := flag.NewFlagSet("album delete", flag.ExitOnError)
	cmd.BoolFunc("yes", "When true, assume Yes to all actions", func(s string) error {
		var err error
		app.AssumeYes, err = strconv.ParseBool(s)
		return err
	})
	err := cmd.Parse(args)
	if err != nil {
		return err
	}

	args = cmd.Args()
	if len(args) > 0 {
		re, err := regexp.Compile(args[0])
		if err != nil {
			return fmt.Errorf("album pattern %q can't be parsed: %w", cmd.Arg(0), err)
		}
		app.pattern = re
	} else {
		app.pattern = regexp.MustCompile(`.*`)
	}

	albums, err := ic.GetAllAlbums(ctx)
	if err != nil {
		return fmt.Errorf("can't get the albums list: %w", err)
	}
	sort.Slice(albums, func(i, j int) bool {
		return albums[i].AlbumName < albums[j].AlbumName
	})

	for _, al := range albums {
		if app.pattern.MatchString(al.AlbumName) {
			yes := app.AssumeYes
			if !yes {
				app.log.OK("Delete album '%s'?", al.AlbumName)
				r, err := ui.ConfirmYesNo(ctx, "Proceed?", "n")
				if err != nil {
					return err
				}
				if r == "y" {
					yes = true
				}
			}
			if yes {
				app.log.MessageContinue(logger.OK, "Deleting album '%s'", al.AlbumName)
				err = app.Immich.DeleteAlbum(ctx, al.ID)
				if err != nil {
					return err
				} else {
					app.log.MessageTerminate(logger.OK, "done")
				}
			}
		}
	}
	return nil
}
