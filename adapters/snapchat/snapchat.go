package snapchat

import (
	"context"
	"io/fs"
	"path"
	"path/filepath"
	"strings"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/filetypes"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/simulot/immich-go/internal/groups"
)

func (sc *Command) Browse(ctx context.Context) chan *assets.Group {
	out := make(chan *assets.Group)
	go func() {
		defer close(out)

		catalog, err := sc.passOne(ctx)
		if err = sc.app.ProcessError(err); err != nil {
			return
		}

		err = sc.passTwo(ctx, catalog, out)
		if err = sc.app.ProcessError(err); err != nil {
			return
		}
	}()
	return out
}

func (sc *Command) passOne(ctx context.Context) (*catalog, error) {
	cat := newCatalog()
	for _, w := range sc.fsyss {
		err := fs.WalkDir(w, ".", func(name string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			if sc.BannedFiles.Match(name) {
				sc.processor.RecordNonAsset(ctx, fshelper.FSName(w, name), 0, fileevent.DiscoveredBanned, "reason", "banned file")
				return nil
			}

			if strings.EqualFold(name, memoriesHistoryJSON) || strings.HasSuffix(strings.ToLower(name), "/"+memoriesHistoryJSON) {
				m, err := readMemoriesHistory(w, name)
				if err != nil {
					sc.processor.RecordNonAsset(ctx, fshelper.FSName(w, name), 0, fileevent.ErrorFileAccess, "error", err.Error())
					return nil
				}
				cat.addMetadata(m)
				sc.processor.RecordNonAsset(ctx, fshelper.FSName(w, name), 0, fileevent.DiscoveredMetadata, "type", "snapchat memories history")
				return nil
			}

			if !isSnapMemoriesPath(name) {
				if sc.supportedMedia.IsUseLess(name) {
					sc.processor.RecordNonAsset(ctx, fshelper.FSName(w, name), 0, fileevent.DiscoveredUnknown, "reason", "useless file")
					return nil
				}
				if sc.supportedMedia.TypeFromExt(strings.ToLower(path.Ext(name))) == filetypes.TypeUnknown {
					sc.processor.RecordNonAsset(ctx, fshelper.FSName(w, name), 0, fileevent.DiscoveredUnsupported, "reason", "unsupported file type")
					return nil
				}
				return nil
			}

			info, err := fs.Stat(w, name)
			if err != nil {
				sc.processor.RecordNonAsset(ctx, fshelper.FSName(w, name), 0, fileevent.ErrorFileAccess, "error", err.Error())
				return nil
			}

			base := path.Base(name)
			if sid, ok := parseOverlaySID(base); ok {
				cat.setOverlay(normalizeSID(sid), &fileRef{fsys: w, name: name, base: base, size: info.Size(), modTime: info.ModTime()})
				sc.processor.RecordNonAsset(ctx, fshelper.FSName(w, name), info.Size(), fileevent.DiscoveredSidecar, "type", "snapchat overlay")
				return nil
			}

			sid, ok := parseMainSID(base)
			if !ok {
				sc.processor.RecordNonAsset(ctx, fshelper.FSName(w, name), info.Size(), fileevent.DiscoveredUnsupported, "reason", "unsupported snapchat memories filename")
				return nil
			}

			ext := strings.ToLower(filepath.Ext(base))
			if !sc.InclusionFlags.IncludedExtensions.Include(ext) {
				sc.processor.RecordAssetDiscardedImmediately(ctx, fshelper.FSName(w, name), info.Size(), fileevent.DiscardedFiltered, "extension not included")
				return nil
			}
			if sc.InclusionFlags.ExcludedExtensions.Exclude(ext) {
				sc.processor.RecordAssetDiscardedImmediately(ctx, fshelper.FSName(w, name), info.Size(), fileevent.DiscardedFiltered, "extension excluded")
				return nil
			}

			cat.setMain(normalizeSID(sid), &fileRef{fsys: w, name: name, base: base, size: info.Size(), modTime: info.ModTime()})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	if cat.metadataCount == 0 {
		sc.app.Log().Warn("can't find json/memories_history.json in input")
	}

	return cat, nil
}

func (sc *Command) passTwo(ctx context.Context, cat *catalog, out chan *assets.Group) error {
	entries := cat.sortedEntries()
	in := make(chan *assets.Asset)
	go func() {
		defer close(in)
		for _, e := range entries {
			if e.main == nil {
				if e.overlay != nil {
					sc.app.Log().Warn("overlay without matching main media", "file", e.overlay.name)
				}
				continue
			}

			asset := sc.makeAssetFromMediaFile(sc.makeMediaFile(e))
			code := fileevent.DiscoveredImage
			if sc.supportedMedia.TypeFromExt(strings.ToLower(filepath.Ext(e.main.base))) == filetypes.TypeVideo {
				code = fileevent.DiscoveredVideo
			}
			sc.processor.RecordAssetDiscovered(ctx, asset.File, int64(asset.FileSize), code)
			if sc.InclusionFlags.DateRange.IsSet() && !sc.InclusionFlags.DateRange.InRange(asset.CaptureDate) {
				asset.Close()
				sc.processor.RecordAssetDiscardedImmediately(ctx, asset.File, int64(asset.FileSize), fileevent.DiscardedFiltered, "asset outside date range")
				continue
			}

			select {
			case in <- asset:
			case <-ctx.Done():
				return
			}
		}
	}()

	gs := groups.NewGrouperPipeline(ctx, sc.groupers...).PipeGrouper(ctx, in)
	for g := range gs {
		select {
		case out <- g:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

func (sc *Command) makeMediaFile(e *catalogEntry) mediaFile {
	m := mediaFile{}
	m.sid = e.sid
	m.fsys = e.main.fsys
	m.name = e.main.name
	m.base = e.main.base
	m.size = e.main.size
	m.modTime = e.main.modTime
	m.main = e.main
	m.overlay = e.overlay
	m.metadata = e.metadata

	if m.metadata == nil {
		m.metadata = &assets.Metadata{}
	}
	if m.metadata.FileName == "" {
		m.metadata.FileName = e.main.base
	}
	if m.metadata.DateTaken.IsZero() {
		m.metadata.DateTaken = inferDateFromSnapName(e.main.base)
	}
	if m.metadata.DateTaken.IsZero() {
		m.metadata.DateTaken = e.main.modTime
	}
	return m
}
