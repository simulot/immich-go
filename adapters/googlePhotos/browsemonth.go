package gp

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"time"

	"github.com/simulot/immich-go/adapters"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/groups"
)

// PreScan builds the full catalog (passOne + solvePuzzle) and returns a sorted
// list of YYYY-MM strings representing months that have assets.
func (toc *TakeoutCmd) PreScan(ctx context.Context) ([]string, error) {
	// Build the catalog: scan all archives and solve the metadata puzzle
	for _, w := range toc.fsyss {
		if err := toc.passOneFsWalk(ctx, w); err != nil {
			return nil, err
		}
	}
	if err := toc.solvePuzzle(ctx); err != nil {
		return nil, err
	}
	toc.catalogBuilt = true

	// Collect dates from all matched files
	monthSet := make(map[string]struct{})
	for _, cat := range toc.catalogs {
		for _, a := range cat.matchedFiles {
			d := gpAssetDate(a)
			if !d.IsZero() {
				month := fmt.Sprintf("%04d-%02d", d.Year(), d.Month())
				monthSet[month] = struct{}{}
			}
		}
	}

	months := make([]string, 0, len(monthSet))
	for m := range monthSet {
		months = append(months, m)
	}
	sort.Strings(months)
	return months, nil
}

// BrowseMonth yields only assets whose date falls within the target month.
// The catalog must have been built by PreScan first.
func (toc *TakeoutCmd) BrowseMonth(ctx context.Context, month string) chan *assets.Group {
	gOut := make(chan *assets.Group)

	if !toc.catalogBuilt {
		close(gOut)
		return gOut
	}

	if month != "no-date" {
		after, before, err := adapters.MonthToDateRange(month)
		if err != nil {
			close(gOut)
			return gOut
		}
		toc.targetMonth = month
		toc.targetAfter = after
		toc.targetBefore = before
	} else {
		toc.targetMonth = "no-date"
		toc.targetAfter = time.Time{}
		toc.targetBefore = time.Time{}
	}

	// Reset fileTracker status so dedup works within this month only
	toc.fileTracker.Range(func(k fileKeyTracker, v trackingInfo) bool {
		v.status = 0
		toc.fileTracker.Store(k, v)
		return true
	})

	go func() {
		defer func() {
			close(gOut)
			toc.targetMonth = ""
		}()
		_ = toc.passTwoForMonth(ctx, gOut)
	}()
	return gOut
}

// passTwoForMonth is like passTwo but filters assets by the target month.
// Assets outside the month are skipped without being closed or discarded.
func (toc *TakeoutCmd) passTwoForMonth(ctx context.Context, gOut chan *assets.Group) error {
	dirs := sort.StringSlice(make([]string, 0, len(toc.catalogs)))
	for d := range toc.catalogs {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)

	for _, dir := range dirs {
		if len(toc.catalogs[dir].matchedFiles) > 0 {
			err := toc.handleDirForMonth(ctx, dir, gOut)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// handleDirForMonth is like handleDir but adds month-based filtering.
// Assets outside the target month are silently skipped.
func (toc *TakeoutCmd) handleDirForMonth(ctx context.Context, dir string, gOut chan *assets.Group) error {
	catalog := toc.catalogs[dir]
	dirEntries := make([]*assets.Asset, 0, len(catalog.matchedFiles))
	isNoDate := toc.targetMonth == "no-date"

	for name := range catalog.matchedFiles {
		a := catalog.matchedFiles[name]
		key := fileKeyTracker{baseName: name, size: int64(a.FileSize)}
		track, _ := toc.fileTracker.Load(key)
		if track.status == fileevent.ProcessedUploadSuccess {
			continue // already emitted in this month batch
		}

		// Month filter: skip assets outside target month (no close, no discard)
		d := gpAssetDate(a)
		if isNoDate {
			if !d.IsZero() {
				continue
			}
		} else if d.IsZero() || d.Before(toc.targetAfter) || !d.Before(toc.targetBefore) {
			continue
		}

		// Apply other metadata filters (archive, partner, trashed, album)
		if code := toc.filterOnMetadata(ctx, a); code != fileevent.Code(0) {
			continue
		}
		dirEntries = append(dirEntries, a)
	}

	in := make(chan *assets.Asset)
	go func() {
		defer close(in)

		sort.Slice(dirEntries, func(i, j int) bool {
			radicalI := dirEntries[i].Radical
			radicalJ := dirEntries[j].Radical
			if radicalI != radicalJ {
				return radicalI < radicalJ
			}
			return dirEntries[i].CaptureDate.Before(dirEntries[j].CaptureDate)
		})

		for _, a := range dirEntries {
			if toc.CreateAlbums {
				if toc.ImportIntoAlbum != "" {
					a.Albums = []assets.Album{{Title: toc.ImportIntoAlbum}}
				} else {
					key := fileKeyTracker{baseName: filepath.Base(a.File.Name()), size: int64(a.FileSize)}
					track, _ := toc.fileTracker.Load(key)
					for _, p := range track.paths {
						if album, ok := toc.albums[p]; ok {
							title := album.Title
							if title == "" {
								if !toc.KeepUntitled {
									continue
								}
								title = filepath.Base(p)
							}
							a.Albums = append(a.Albums, assets.Album{
								Title:       title,
								Description: album.Description,
								Latitude:    album.Latitude,
								Longitude:   album.Longitude,
							})
						}
					}
				}
				if toc.PartnerSharedAlbum != "" && a.FromPartner {
					a.Albums = append(a.Albums, assets.Album{Title: toc.PartnerSharedAlbum})
				}
				if a.FromApplication != nil {
					a.FromApplication.Albums = a.Albums
				}
			}
			if a.Latitude == 0 && a.Longitude == 0 {
				for _, album := range a.Albums {
					if album.Latitude != 0 || album.Longitude != 0 {
						a.Latitude = album.Latitude
						a.Longitude = album.Longitude
						break
					}
				}
			}
			if toc.TakeoutTag {
				a.AddTag(toc.TakeoutName)
			}

			select {
			case in <- a:
			case <-ctx.Done():
				return
			}
		}
	}()

	gs := groups.NewGrouperPipeline(ctx, toc.groupers...).PipeGrouper(ctx, in)
	for g := range gs {
		select {
		case gOut <- g:
			for _, a := range g.Assets {
				key := fileKeyTracker{
					baseName: path.Base(a.File.Name()),
					size:     int64(a.FileSize),
				}
				track, _ := toc.fileTracker.Load(key)
				track.status = fileevent.ProcessedUploadSuccess
				toc.fileTracker.Store(key, track)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

// gpAssetDate returns the best available date for a Google Photos asset.
func gpAssetDate(a *assets.Asset) time.Time {
	if !a.CaptureDate.IsZero() {
		return a.CaptureDate
	}
	if !a.FileDate.IsZero() {
		return a.FileDate
	}
	return a.Taken
}
