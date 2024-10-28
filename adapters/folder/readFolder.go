package folder

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/simulot/immich-go/adapters"
	"github.com/simulot/immich-go/internal/albums"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/groups"
	"github.com/simulot/immich-go/internal/groups/burst"
	"github.com/simulot/immich-go/internal/groups/series"
	"github.com/simulot/immich-go/internal/metadata"
	"github.com/simulot/immich-go/internal/worker"
)

type LocalAssetBrowser struct {
	fsyss    []fs.FS
	log      *fileevent.Recorder
	flags    *ImportFolderOptions
	exiftool *metadata.ExifTool
	pool     *worker.Pool
	wg       sync.WaitGroup
}

func NewLocalFiles(ctx context.Context, l *fileevent.Recorder, flags *ImportFolderOptions, fsyss ...fs.FS) (*LocalAssetBrowser, error) {
	if flags.ImportIntoAlbum != "" && flags.UsePathAsAlbumName != FolderModeNone {
		return nil, errors.New("cannot use both --into-album and --folder-as-album")
	}

	la := LocalAssetBrowser{
		fsyss: fsyss,
		flags: flags,
		log:   l,
		pool:  worker.NewPool(3), // TODO: Make this configurable
	}
	if flags.InfoCollector == nil {
		flags.InfoCollector = filenames.NewInfoCollector(flags.DateHandlingFlags.FilenameTimeZone.Location(), flags.SupportedMedia)
	}

	if flags.ExifToolFlags.UseExifTool {
		et, err := metadata.NewExifTool(&flags.ExifToolFlags)
		if err != nil {
			return nil, err
		}
		la.exiftool = et
	}

	return &la, nil
}

func (la *LocalAssetBrowser) Browse(ctx context.Context) chan *groups.AssetGroup {
	gOut := make(chan *groups.AssetGroup)
	go func() {
		defer close(gOut)
		for _, fsys := range la.fsyss {
			la.concurrentParseDir(ctx, fsys, ".", gOut)
		}
		la.wg.Wait()
		la.pool.Stop()
	}()
	return gOut
}

func (la *LocalAssetBrowser) concurrentParseDir(ctx context.Context, fsys fs.FS, dir string, gOut chan *groups.AssetGroup) {
	la.wg.Add(1)
	ctx, cancel := context.WithCancelCause(ctx)
	go la.pool.Submit(func() {
		defer la.wg.Done()
		err := la.parseDir(ctx, fsys, dir, gOut)
		if err != nil {
			cancel(err)
		}
	})
}

func (la *LocalAssetBrowser) parseDir(ctx context.Context, fsys fs.FS, dir string, gOut chan *groups.AssetGroup) error {
	fsName := ""
	if fsys, ok := fsys.(interface{ Name() string }); ok {
		fsName = fsys.Name()
	}

	var assets []*adapters.Asset
	var entries []fs.DirEntry
	var err error

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		entries, err = fs.ReadDir(fsys, dir)
		if err != nil {
			return err
		}
	}

	for _, entry := range entries {
		base := entry.Name()
		name := filepath.Join(dir, base)
		if entry.IsDir() {
			continue
		}

		if la.flags.BannedFiles.Match(name) {
			la.log.Record(ctx, fileevent.DiscoveredDiscarded, fileevent.AsFileAndName(fsys, entry.Name()), "reason", "banned file")
			continue
		}

		ext := filepath.Ext(base)
		mediaType := la.flags.SupportedMedia.TypeFromExt(ext)

		if mediaType == metadata.TypeUnknown {
			la.log.Record(ctx, fileevent.DiscoveredUnsupported, fileevent.AsFileAndName(fsys, name), "reason", "unsupported file type")
			continue
		}

		switch mediaType {
		case metadata.TypeImage:
			la.log.Record(ctx, fileevent.DiscoveredImage, fileevent.AsFileAndName(fsys, name))
		case metadata.TypeVideo:
			la.log.Record(ctx, fileevent.DiscoveredVideo, fileevent.AsFileAndName(fsys, name))
		case metadata.TypeSidecar:
			if la.flags.IgnoreSideCarFiles {
				la.log.Record(ctx, fileevent.DiscoveredDiscarded, fileevent.AsFileAndName(fsys, name), "reason", "sidecar file ignored")
				continue
			}
			la.log.Record(ctx, fileevent.DiscoveredSidecar, fileevent.AsFileAndName(fsys, name))
		}

		if !la.flags.InclusionFlags.IncludedExtensions.Include(ext) {
			la.log.Record(ctx, fileevent.DiscoveredDiscarded, fileevent.AsFileAndName(fsys, name), "reason", "extension not included")
			continue
		}

		if la.flags.InclusionFlags.ExcludedExtensions.Exclude(ext) {
			la.log.Record(ctx, fileevent.DiscoveredDiscarded, fileevent.AsFileAndName(fsys, name), "reason", "extension excluded")
			continue
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// we have a file to process
			a, err := la.assetFromFile(ctx, fsys, name)
			if err != nil {
				la.log.Record(ctx, fileevent.Error, fileevent.AsFileAndName(fsys, name), "error", err.Error())
				return err
			}
			if a != nil {
				assets = append(assets, a)
			}
		}
	}

	// process the left over dirs
	for _, entry := range entries {
		base := entry.Name()
		name := filepath.Join(dir, base)
		if entry.IsDir() {
			if la.flags.BannedFiles.Match(name) {
				la.log.Record(ctx, fileevent.DiscoveredDiscarded, fileevent.AsFileAndName(fsys, name), "reason", "banned folder")
				continue // Skip this folder, no error
			}
			if la.flags.Recursive && entry.Name() != "." {
				la.concurrentParseDir(ctx, fsys, name, gOut)
			}
			continue
		}
	}

	in := make(chan groups.Asset)
	go func() {
		defer close(in)

		sort.Slice(assets, func(i, j int) bool {
			// Sort by radical first
			radicalI := assets[i].NameInfo().Radical
			radicalJ := assets[j].NameInfo().Radical
			if radicalI != radicalJ {
				return radicalI < radicalJ
			}
			// If radicals are the same, sort by date
			return assets[i].CaptureDate.Before(assets[j].CaptureDate)
		})

		for _, a := range assets {
			if la.flags.InclusionFlags.DateRange.IsSet() && !la.flags.InclusionFlags.DateRange.InRange(a.CaptureDate) {
				a.Close()
				la.log.Record(ctx, fileevent.DiscoveredDiscarded, a, "reason", "asset outside date range")
				continue
			}
			select {
			case in <- a:
			case <-ctx.Done():
				return
			}
		}
	}()

	gs := groups.NewGrouperPipeline(ctx, series.Group, burst.Group).PipeGrouper(ctx, in)
	for g := range gs {
		// Add album information
		if la.flags.ImportIntoAlbum != "" {
			g.Albums = []albums.Album{{Title: la.flags.ImportIntoAlbum}}
		} else if la.flags.UsePathAsAlbumName != FolderModeNone {
			Album := ""
			switch la.flags.UsePathAsAlbumName {
			case FolderModeFolder:
				if dir == "." {
					Album = fsName
				} else {
					Album = filepath.Base(dir)
				}
			case FolderModePath:
				parts := []string{}
				if fsName != "" {
					parts = append(parts, fsName)
				}
				if dir != "." {
					parts = append(parts, strings.Split(dir, string(filepath.Separator))...)
				}
				Album = strings.Join(parts, la.flags.AlbumNamePathSeparator)
			}
			g.Albums = []albums.Album{{Title: Album}}
		}
		select {
		case gOut <- g:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (la *LocalAssetBrowser) assetFromFile(_ context.Context, fsys fs.FS, name string) (*adapters.Asset, error) {
	a := &adapters.Asset{
		FileName: name,
		Title:    filepath.Base(name),
		FSys:     fsys,
	}
	a.SetNameInfo(la.flags.InfoCollector.GetInfo(a.Title))

	md, err := a.ReadMetadata(la.flags.DateHandlingFlags.Method, adapters.ReadMetadataOptions{
		ExifTool:         la.exiftool,
		ExiftoolTimezone: la.flags.ExifToolFlags.Timezone.Location(),
		FilenameTimeZone: la.flags.DateHandlingFlags.FilenameTimeZone.Location(),
	})
	if err != nil {
		a.Close()
		return nil, err
	}

	i, err := fs.Stat(fsys, name)
	if err != nil {
		a.Close()
		return nil, err
	}
	a.FileSize = int(i.Size())
	a.FileDate = i.ModTime()

	if md != nil {
		a.CaptureDate = md.DateTaken
	}

	return a, nil
}
