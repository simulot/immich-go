package nextcloudmemories

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/simulot/immich-go/adapters/shared"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/filetypes"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/simulot/immich-go/internal/namematcher"
)

var defaultBannedFiles = namematcher.MustList(shared.DefaultBannedFiles...)

func (nc *Command) Browse(ctx context.Context) chan *assets.Group {
	gOut := make(chan *assets.Group)

	go func() {
		defer close(gOut)

		err := nc.browse(ctx, gOut)
		if nc.app != nil {
			_ = nc.app.ProcessError(err)
		}
	}()

	return gOut
}

func (nc *Command) browse(ctx context.Context, gOut chan<- *assets.Group) error {
	if nc.sourceFS == nil {
		return errors.New("nextcloud Memories source is not initialized")
	}
	if len(nc.selectedRoots) == 0 {
		return errors.New("nextcloud Memories import has no selected timeline roots")
	}
	if err := nc.ensureMetadataIndex(ctx); err != nil {
		return err
	}

	supportedMedia := filetypes.DefaultSupportedMedia
	infoCollector := filenames.NewInfoCollector(nil, supportedMedia)
	processor := nc.fileProcessor()
	if nc.app != nil {
		supportedMedia = nc.app.GetSupportedMedia()
		infoCollector = filenames.NewInfoCollector(nc.app.GetTZ(), supportedMedia)
	}

	seen := map[string]struct{}{}

	for _, root := range nc.selectedRoots {
		walkRoot := timelineRootToFSPath(root)
		err := fs.WalkDir(nc.sourceFS, walkRoot, func(name string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			if entry.IsDir() {
				if matchesBanned(defaultBannedFiles, name, true) {
					return fs.SkipDir
				}
				return nil
			}

			info, err := entry.Info()
			if err != nil {
				if processor != nil {
					processor.RecordNonAsset(ctx, fshelper.FSName(nc.sourceFS, name), 0, fileevent.ErrorFileAccess, "error", err.Error())
				}
				return nil
			}

			return nc.emitAsset(ctx, gOut, seen, name, info, infoCollector, supportedMedia, processor)
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (nc *Command) emitAsset(ctx context.Context, gOut chan<- *assets.Group, seen map[string]struct{}, name string, info fs.FileInfo, infoCollector *filenames.InfoCollector, supportedMedia filetypes.SupportedMedia, processor interface {
	RecordAssetDiscovered(context.Context, fshelper.FSAndName, int64, fileevent.Code)
	RecordNonAsset(context.Context, fshelper.FSAndName, int64, fileevent.Code, ...any)
},
) error {
	if matchesBanned(defaultBannedFiles, name, false) {
		if processor != nil {
			processor.RecordNonAsset(ctx, fshelper.FSName(nc.sourceFS, name), 0, fileevent.DiscoveredBanned, "reason", "banned file")
		}
		return nil
	}

	if _, ok := seen[name]; ok {
		return nil
	}
	seen[name] = struct{}{}

	if supportedMedia.IsUseLess(name) {
		if processor != nil {
			processor.RecordNonAsset(ctx, fshelper.FSName(nc.sourceFS, name), 0, fileevent.DiscoveredUnknown, "reason", "useless file")
		}
		return nil
	}

	mediaType := supportedMedia.TypeFromExt(path.Ext(name))
	switch mediaType {
	case filetypes.TypeSidecar:
		if processor != nil {
			processor.RecordNonAsset(ctx, fshelper.FSName(nc.sourceFS, name), info.Size(), fileevent.DiscoveredSidecar)
		}
		return nil
	case filetypes.TypeImage, filetypes.TypeVideo:
	default:
		if processor != nil {
			processor.RecordNonAsset(ctx, fshelper.FSName(nc.sourceFS, name), info.Size(), fileevent.DiscoveredUnsupported, "reason", "unsupported file type")
		}
		return nil
	}

	asset, err := nc.assetFromInfo(ctx, name, info, infoCollector)
	if err != nil {
		return err
	}
	if processor != nil {
		discoveryCode := fileevent.DiscoveredImage
		if mediaType == filetypes.TypeVideo {
			discoveryCode = fileevent.DiscoveredVideo
		}
		processor.RecordAssetDiscovered(ctx, asset.File, info.Size(), discoveryCode)
	}

	select {
	case gOut <- assets.NewGroup(assets.GroupByNone, asset):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (nc *Command) assetFromInfo(ctx context.Context, name string, info fs.FileInfo, infoCollector *filenames.InfoCollector) (*assets.Asset, error) {
	asset := &assets.Asset{
		File:             fshelper.FSName(nc.sourceFS, name),
		FileSize:         int(info.Size()),
		FileDate:         info.ModTime(),
		OriginalFileName: path.Base(name),
	}
	asset.SetNameInfo(infoCollector.GetInfo(asset.OriginalFileName))

	if nc.metadataIndex == nil {
		return asset, nil
	}

	md, ok, err := nc.metadataIndex.Get(ctx, name)
	if err != nil {
		if nc.RequireIndexed {
			return nil, err
		}
		if nc.app != nil {
			nc.app.Log().Warn("Nextcloud Memories metadata lookup failed; importing without source metadata", "file", name, "err", err)
		}
		return asset, nil
	}
	if !ok {
		if nc.RequireIndexed {
			return nil, fmt.Errorf("memories metadata missing for %q; the source library appears partially indexed. rerun without --require-indexed to continue", name)
		}
		if nc.app != nil {
			nc.app.Log().Warn("Nextcloud Memories asset is not indexed; importing without source metadata", "file", name)
		}
		return asset, nil
	}

	asset.FromApplication = asset.UseMetadata(md)
	return asset, nil
}

func (nc *Command) fileProcessor() interface {
	RecordAssetDiscovered(context.Context, fshelper.FSAndName, int64, fileevent.Code)
	RecordNonAsset(context.Context, fshelper.FSAndName, int64, fileevent.Code, ...any)
} {
	if nc.app == nil {
		return nil
	}
	return nc.app.FileProcessor()
}

func timelineRootToFSPath(root string) string {
	root = strings.TrimSpace(root)
	if root == "" || root == "/" {
		return "."
	}
	return strings.TrimPrefix(root, "/")
}

func matchesBanned(list namematcher.List, name string, isDir bool) bool {
	trimmed := strings.TrimSuffix(name, "/")
	if isDir {
		if list.MatchDir(name) {
			return true
		}
		if trimmed != name && list.MatchDir(trimmed) {
			return true
		}
		return false
	}
	if list.MatchFile(name) {
		return true
	}
	if trimmed != name && list.MatchFile(trimmed) {
		return true
	}
	return false
}
