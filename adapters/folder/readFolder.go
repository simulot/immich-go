package folder

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/exif"
	"github.com/simulot/immich-go/internal/exif/sidecars/xmpsidecar"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/filetypes"
	"github.com/simulot/immich-go/internal/filters"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/simulot/immich-go/internal/groups"
	"github.com/simulot/immich-go/internal/groups/burst"
	"github.com/simulot/immich-go/internal/groups/epsonfastfoto"
	"github.com/simulot/immich-go/internal/groups/series"
	"github.com/simulot/immich-go/internal/worker"
)

type LocalAssetBrowser struct {
	fsyss    []fs.FS
	log      *fileevent.Recorder
	flags    *ImportFolderOptions
	pool     *worker.Pool
	wg       sync.WaitGroup
	groupers []groups.Grouper
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
		flags.InfoCollector = filenames.NewInfoCollector(flags.ExifToolFlags.Timezone.TZ, flags.SupportedMedia)
	}

	if flags.ExifToolFlags.UseExifTool {
		err := exif.NewExifTool(&flags.ExifToolFlags)
		if err != nil {
			return nil, err
		}

	}

	if flags.ManageEpsonFastFoto {
		g := epsonfastfoto.Group{}
		la.groupers = append(la.groupers, g.Group)
	}
	if flags.ManageBurst != filters.BurstNothing {
		la.groupers = append(la.groupers, burst.Group)
	}
	la.groupers = append(la.groupers, series.Group)

	return &la, nil
}

func (la *LocalAssetBrowser) Browse(ctx context.Context) chan *assets.Group {
	gOut := make(chan *assets.Group)
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

func (la *LocalAssetBrowser) concurrentParseDir(ctx context.Context, fsys fs.FS, dir string, gOut chan *assets.Group) {
	la.wg.Add(1)
	ctx, cancel := context.WithCancelCause(ctx)
	go la.pool.Submit(func() {
		defer la.wg.Done()
		err := la.parseDir(ctx, fsys, dir, gOut)
		if err != nil {
			la.log.Log().Error(err.Error())
			cancel(err)
		}
	})
}

func (la *LocalAssetBrowser) parseDir(ctx context.Context, fsys fs.FS, dir string, gOut chan *assets.Group) error {
	fsName := ""
	if fsys, ok := fsys.(interface{ Name() string }); ok {
		fsName = fsys.Name()
	}

	var as []*assets.Asset
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
		name := path.Join(dir, base)
		if entry.IsDir() {
			continue
		}

		if la.flags.BannedFiles.Match(name) {
			la.log.Record(ctx, fileevent.DiscoveredDiscarded, fshelper.FSName(fsys, entry.Name()), "reason", "banned file")
			continue
		}

		ext := filepath.Ext(base)
		mediaType := la.flags.SupportedMedia.TypeFromExt(ext)

		if mediaType == filetypes.TypeUnknown {
			la.log.Record(ctx, fileevent.DiscoveredUnsupported, fshelper.FSName(fsys, name), "reason", "unsupported file type")
			continue
		}

		switch mediaType {
		case filetypes.TypeImage:
			la.log.Record(ctx, fileevent.DiscoveredImage, fshelper.FSName(fsys, name))
		case filetypes.TypeVideo:
			la.log.Record(ctx, fileevent.DiscoveredVideo, fshelper.FSName(fsys, name))
		case filetypes.TypeSidecar:
			if la.flags.IgnoreSideCarFiles {
				la.log.Record(ctx, fileevent.DiscoveredDiscarded, fshelper.FSName(fsys, name), "reason", "sidecar file ignored")
				continue
			}
			la.log.Record(ctx, fileevent.DiscoveredSidecar, fshelper.FSName(fsys, name))
			continue
		}

		if !la.flags.InclusionFlags.IncludedExtensions.Include(ext) {
			la.log.Record(ctx, fileevent.DiscoveredDiscarded, fshelper.FSName(fsys, name), "reason", "extension not included")
			continue
		}

		if la.flags.InclusionFlags.ExcludedExtensions.Exclude(ext) {
			la.log.Record(ctx, fileevent.DiscoveredDiscarded, fshelper.FSName(fsys, name), "reason", "extension excluded")
			continue
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// we have a file to process
			a, err := la.assetFromFile(ctx, fsys, name)
			if err != nil {
				la.log.Record(ctx, fileevent.Error, fshelper.FSName(fsys, name), "error", err.Error())
				return err
			}
			if a != nil {
				as = append(as, a)
			}
		}
	}

	// process the left over dirs
	for _, entry := range entries {
		base := entry.Name()
		name := path.Join(dir, base)
		if entry.IsDir() {
			if la.flags.BannedFiles.Match(name) {
				la.log.Record(ctx, fileevent.DiscoveredDiscarded, fshelper.FSName(fsys, name), "reason", "banned folder")
				continue // Skip this folder, no error
			}
			if la.flags.Recursive && entry.Name() != "." {
				la.concurrentParseDir(ctx, fsys, name, gOut)
			}
			continue
		}
	}

	in := make(chan *assets.Asset)
	go func() {
		defer close(in)

		sort.Slice(as, func(i, j int) bool {
			// Sort by radical first
			radicalI := as[i].Radical
			radicalJ := as[j].Radical
			if radicalI != radicalJ {
				return radicalI < radicalJ
			}
			// If radicals are the same, sort by date
			return as[i].CaptureDate.Before(as[j].CaptureDate)
		})

		for _, a := range as {
			if la.flags.InclusionFlags.DateRange.IsSet() {
				md := &assets.Metadata{}
				err := exif.GetMetaData(a, md, la.flags.ExifToolFlags)
				if err != nil {
					a.Close()
					la.log.Record(ctx, fileevent.DiscoveredDiscarded, a, "reason", "can't get the capture date")
					continue
				}
				a.FromSourceFile = a.UseMetadata(md)
			}

			// check the presence of a XMP file
			if b, xmpName := detectXMP(fsys, a.File.Name()); b {
				buf, err := fs.ReadFile(fsys, xmpName)
				if err != nil {
					la.log.Record(ctx, fileevent.Error, a, "error", err.Error())
				} else {
					md := &assets.Metadata{}
					err = xmpsidecar.ReadXMP(bytes.NewReader(buf), md)
					if err != nil {
						la.log.Record(ctx, fileevent.Error, a, "error", err.Error())
					} else {
						a.FromSideCar = a.UseMetadata(md)
						a.FromSideCar.File = fshelper.FSName(fsys, xmpName)
					}
				}
			}

			if !la.flags.InclusionFlags.DateRange.InRange(a.CaptureDate) {
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

	gs := groups.NewGrouperPipeline(ctx, la.groupers...).PipeGrouper(ctx, in)
	for g := range gs {
		// Add album information
		if la.flags.ImportIntoAlbum != "" {
			g.Albums = []assets.Album{{Title: la.flags.ImportIntoAlbum}}
		} else if la.flags.UsePathAsAlbumName != FolderModeNone && la.flags.UsePathAsAlbumName != "" {
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
			g.Albums = []assets.Album{{Title: Album}}
		} else {
			for _, a := range g.Assets {
				for _, al := range a.Albums {
					g.AddAlbum(al)
				}
			}
		}
		for _, a := range g.Assets {
			a.Albums = g.Albums
		}
		select {
		case gOut <- g:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func detectXMP(fsys fs.FS, name string) (bool, string) {
	xmp := name + ".xmp"
	_, err := fshelper.Stat(fsys, xmp)
	if err == nil {
		return true, xmp
	}
	xmp = name + ".XMP"
	_, err = fshelper.Stat(fsys, xmp)
	if err == nil {
		return true, xmp
	}

	name = strings.TrimSuffix(name, filepath.Ext(name))
	xmp = name + ".xmp"
	_, err = fshelper.Stat(fsys, xmp)
	if err == nil {
		return true, xmp
	}
	xmp = name + ".XMP"
	_, err = fshelper.Stat(fsys, xmp)
	if err == nil {
		return true, xmp
	}

	return false, ""
}

func (la *LocalAssetBrowser) assetFromFile(_ context.Context, fsys fs.FS, name string) (*assets.Asset, error) {
	a := &assets.Asset{
		File:             fshelper.FSName(fsys, name),
		OriginalFileName: filepath.Base(name),
	}
	i, err := fs.Stat(fsys, name)
	if err != nil {
		a.Close()
		return nil, err
	}
	a.FileSize = int(i.Size())
	a.FileDate = i.ModTime()
	a.SetNameInfo(la.flags.InfoCollector.GetInfo(a.OriginalFileName))
	return a, nil
}
