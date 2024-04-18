package duplicate

// Check the list of photos to list and discard duplicates.

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/simulot/immich-go/cmd"
	"github.com/simulot/immich-go/helpers/gen"
	"github.com/simulot/immich-go/helpers/myflag"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/logger"
	"golang.org/x/sync/errgroup"
)

type DuplicateCmd struct {
	*cmd.SharedFlags
	AssumeYes       bool             // When true, doesn't ask to the user
	DateRange       immich.DateRange // Set capture date range
	IgnoreTZErrors  bool             // Enable TZ error tolerance
	IgnoreExtension bool             // Ignore file extensions when checking for duplicates

	assetsByID          map[string]*immich.Asset
	assetsByBaseAndDate map[duplicateKey][]*immich.Asset
	keys                []duplicateKey
	page                *tea.Program
	ctx                 context.Context
}

type duplicateKey struct {
	Date time.Time
	Name string
	Type string
}

func DuplicateCommand(ctx context.Context, common *cmd.SharedFlags, args []string) error {
	app, err := newDuplicateCmd(ctx, common, args)
	if err != nil {
		return err
	}

	// Initialize the TUI
	app.page = tea.NewProgram(NewDuplicateModel(app, app.keys), tea.WithAltScreen())

	// Launch the getAssets and duplicate detection in the background
	errGrp := errgroup.Group{}
	errGrp.Go(func() error {
		err := app.getAssets()
		if err != nil {
			app.send(msgError{Err: err})
		}
		return err
	})

	m, err := app.page.Run()
	if err != nil {
		return err
	}
	if m, ok := m.(DuplicateModel); ok {
		return m.err
	}

	/*

		for _, k := range app.keys {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				l := app.assetsByBaseAndDate[k]
				app.Log.Print("There are %d copies of the asset %s, taken on %s ", len(l), k.Name, l[0].ExifInfo.DateTimeOriginal.Format(time.RFC3339))
				albums := []immich.AlbumSimplified{}
				assetsToDelete := []string{}
				sort.Slice(l, func(i, j int) bool { return l[i].ExifInfo.FileSizeInByte < l[j].ExifInfo.FileSizeInByte })
				for p, a := range l {
					if p < len(l)-1 {
						app.Log.Print("  delete %s %dx%d, %s, %s", a.OriginalFileName, a.ExifInfo.ExifImageWidth, a.ExifInfo.ExifImageHeight, ui.FormatBytes(a.ExifInfo.FileSizeInByte), a.OriginalPath)
						assetsToDelete = append(assetsToDelete, a.ID)
						r, err := app.Immich.GetAssetAlbums(ctx, a.ID)
						if err != nil {
							app.Log.Error("Can't get asset's albums: %s", err.Error())
						} else {
							albums = append(albums, r...)
						}
					} else {
						app.Log.Print("  keep   %s %dx%d, %s, %s", a.OriginalFileName, a.ExifInfo.ExifImageWidth, a.ExifInfo.ExifImageHeight, ui.FormatBytes(a.ExifInfo.FileSizeInByte), a.OriginalPath)
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
								app.Log.Error("Can't delete asset: %s", err.Error())
							} else {
								app.Log.Print("  Asset removed")
								for _, al := range albums {
									app.Log.Print("  Update the album %s with the best copy", al.AlbumName)
									_, err = app.Immich.AddAssetToAlbum(ctx, al.ID, []string{a.ID})
									if err != nil {
										app.Log.Error("Can't delete asset: %s", err.Error())
									}
								}
							}
						}
					}
				}
			}

		}
	*/
	return nil
}

func (app *DuplicateCmd) send(msg tea.Msg) {
	if app.NoUI {
		switch msg := msg.(type) {
		case logger.MsgLog:
		case logger.MsgStageSpinner:
			fmt.Println(msg.Label)
		}
	} else {
		app.page.Send(msg)
	}
}

func newDuplicateCmd(ctx context.Context, common *cmd.SharedFlags, args []string) (*DuplicateCmd, error) {
	cmd := flag.NewFlagSet("duplicate", flag.ExitOnError)
	validRange := immich.DateRange{}
	_ = validRange.Set("1850-01-04,2030-01-01")
	app := DuplicateCmd{
		SharedFlags:         common,
		DateRange:           validRange,
		assetsByID:          map[string]*immich.Asset{},
		assetsByBaseAndDate: map[duplicateKey][]*immich.Asset{},
		ctx:                 ctx,
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

func (app *DuplicateCmd) getAssets() error {
	statistics, err := app.Immich.GetServerStatistics(app.ctx)
	totalOnImmich := float64(statistics.Photos + statistics.Videos)
	received := 0
	dupCount := 0
	if err != nil {
		return err
	}

	done := errors.New("done")
	app.send(logger.MsgLog{Message: "Get %d asset(s) from the server"})

	err = app.Immich.GetAllAssetsWithFilter(app.ctx, func(ctx context.Context, a *immich.Asset) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			received++
			app.send(msgReceivePct(int(100 * float64(received) / totalOnImmich)))
			if a.IsTrashed {
				return nil
			}
			if !app.DateRange.InRange(a.ExifInfo.DateTimeOriginal.Time) {
				return nil
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
				app.send(msgDuplicate(dupCount))
				if dupCount > 20 {
					return done
				}
			}
			app.assetsByBaseAndDate[k] = append(l, a)
		}
		return nil
	})
	if err != nil && err != done {
		return err
	}
	// Get the duplicated sorted by date and name
	app.keys = gen.MapFilterKeys(app.assetsByBaseAndDate, func(i []*immich.Asset) bool {
		return len(i) > 1
	})
	sort.Slice(app.keys, func(i, j int) bool {
		c := app.keys[i].Date.Compare(app.keys[j].Date)
		switch c {
		case -1:
			return true
		case +1:
			return false
		}
		c = strings.Compare(app.keys[i].Name, app.keys[j].Name)
		return c == -1
	})
	app.send(app.keys)
	return nil
}
