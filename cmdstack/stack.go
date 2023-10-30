package cmdstack

import (
	"context"
	"flag"
	"immich-go/helpers/gen"
	"immich-go/immich"
	"immich-go/immich/logger"
	"immich-go/ui"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)

type StackCmd struct {
	Immich *immich.ImmichClient // Immich client
	logger *logger.Logger

	assetsById       map[string]*immich.Asset
	assetsByBaseName map[string][]*immich.Asset
	stacksByID       map[string][]string
	newStacks        map[newStackKey]newStack
	AssumeYes        bool
	DateRange        immich.DateRange // Set capture date range
}

type stack struct {
	IDs []*immich.Asset
}

type newStackKey struct {
	date     time.Time // time rounded at 5 min
	baseName string    // stack group
}

type newStack struct {
	coverID   string
	stackedID []string
	names     []string
}

func initSack(xtx context.Context, ic *immich.ImmichClient, log *logger.Logger, args []string) (*StackCmd, error) {
	cmd := flag.NewFlagSet("stack", flag.ExitOnError)
	validRange := immich.DateRange{}

	validRange.Set("1850-01-04,2030-01-01")
	app := StackCmd{
		logger:           log,
		Immich:           ic,
		assetsById:       map[string]*immich.Asset{},
		stacksByID:       map[string][]string{},
		assetsByBaseName: map[string][]*immich.Asset{},
		newStacks:        map[newStackKey]newStack{},
		DateRange:        validRange,
	}

	cmd.BoolFunc("yes", "When true, assume Yes to all actions", func(s string) error {
		var err error
		app.AssumeYes, err = strconv.ParseBool(s)
		return err
	})
	cmd.Var(&app.DateRange, "date", "Process only documents having a capture date in that range.")
	err := cmd.Parse(args)
	return &app, err
}
func NewStackCommand(ctx context.Context, ic *immich.ImmichClient, log *logger.Logger, args []string) error {
	app, err := initSack(ctx, ic, log, args)
	if err != nil {
		return err
	}

	log.MessageContinue(logger.OK, "Get server's assets...")

	err = app.Immich.GetAllAssetsWithFilter(ctx, nil, func(a *immich.Asset) {
		if a.IsTrashed {
			return
		}
		if !app.DateRange.InRange(a.ExifInfo.DateTimeOriginal.Time) {
			return
		}
		app.assetsById[a.ID] = a
		base := a.OriginalFileName
		if a.StackParentId != "" {
			s := app.stacksByID[a.StackParentId]
			app.stacksByID[a.StackParentId] = append(s, a.ID)
		}

		l := app.assetsByBaseName[base]
		app.assetsByBaseName[base] = append(l, a)
	})
	if err != nil {
		return err
	}
	log.MessageTerminate(logger.OK, " %d received", len(app.assetsById))

	// Search for BURST or pairs JPG and raw
	for _, a := range app.assetsById {
		// Ignore images already in a stack
		_, inAStack := app.stacksByID[a.ID]
		inAStack = inAStack || a.StackParentId != ""
		if inAStack {
			continue
		}

		base := a.OriginalFileName
		if idx := strings.Index(base, "_BURST"); idx > 1 {
			key := newStackKey{
				baseName: base[:idx],
				date:     a.ExifInfo.DateTimeOriginal.Time.Round(1 * time.Minute),
			}
			s, ok := app.newStacks[key]
			if !ok {
				s.coverID = a.ID
			}
			s.stackedID = append(s.stackedID, a.ID)
			s.names = append(s.names, a.OriginalFileName+path.Ext(a.OriginalPath))
			if strings.Contains(base, "COVER") {
				s.coverID = a.ID
			}
			app.newStacks[key] = s
			continue
		}

		l := app.assetsByBaseName[base]
		if len(l) > 1 {
			t := a.ExifInfo.DateTimeOriginal.Time.Round(1 * time.Minute)
			for _, sameBase := range l {
				if sameBase.ID != a.ID {
					t2 := sameBase.ExifInfo.DateTimeOriginal.Time.Round(1 * time.Minute)
					if t.Equal(t2) {
						key := newStackKey{
							baseName: base,
							date:     a.ExifInfo.DateTimeOriginal.Time.Round(1 * time.Minute),
						}
						s, ok := app.newStacks[key]
						if !ok {
							s.coverID = a.ID
						}
						s.stackedID = append(s.stackedID, a.ID)
						s.names = append(s.names, a.OriginalFileName+path.Ext(a.OriginalPath))
						if strings.ToLower(path.Ext(a.OriginalFileName)) == ".jpg" {
							s.coverID = a.ID
						}
						app.newStacks[key] = s
					}
				}
			}
		}
	}

	keys := gen.MapFilterKeys(app.newStacks, func(i newStack) bool {
		return len(i.stackedID) > 1
	})

	if len(keys) == 0 {
		log.OK("No possibility of stack detected")
		return nil
	}

	log.OK("%d possible stack(s) detected", len(keys))
	sort.Slice(keys, func(i, j int) bool {
		c := keys[i].date.Compare(keys[j].date)
		switch c {
		case -1:
			return true
		case +1:
			return false
		}
		c = strings.Compare(keys[i].baseName, keys[j].baseName)
		switch c {
		case -1:
			return true
		}
		return false
	})

	for _, k := range keys {
		log.OK("Stack following images taken on %s", k.date)
		cover := app.newStacks[k].coverID
		IDs := gen.DeleteItem[string](app.newStacks[k].stackedID, cover)
		names := app.newStacks[k].names
		sort.Strings(names)
		for _, n := range names {
			log.OK("  %s", n)
		}
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
			err := app.Immich.UpdateAssets(ctx, IDs, false, false, false, cover)
			if err != nil {
				log.Warning("Can't stack images: %s", err)
			}
		}
	}

	return nil
}
