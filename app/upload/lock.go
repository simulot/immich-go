package upload

import (
	"context"
	"fmt"

	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/fileevent"
)

type lockOutcome int

const (
	lockNotNeeded lockOutcome = iota // nothing to do: input not locked, or server asset already locked
	lockApplied                      // server asset moved to the locked folder
	lockSkipped                      // server asset left untouched, see the returned reason
)

// lockClient is the part of the Immich client needed to lock an asset.
type lockClient interface {
	GetAssetAlbums(ctx context.Context, assetID string) ([]immich.AlbumSimplified, error)
	SetAssetsVisibility(ctx context.Context, ids []string, visibility assets.Visibility) error
}

// lockServerAsset moves the server asset to the Immich locked folder when the input asset is locked.
// Immich removes locked assets from albums, so an asset belonging to albums is left untouched.
// Albums are read from the server, the cached ones may be incomplete.
func lockServerAsset(ctx context.Context, c lockClient, input, server *assets.Asset) (lockOutcome, string) {
	if input.Visibility != assets.VisibilityLocked || server == nil || server.ID == "" || server.Visibility == assets.VisibilityLocked {
		return lockNotNeeded, ""
	}

	albums, err := c.GetAssetAlbums(ctx, server.ID)
	if err != nil {
		return lockSkipped, fmt.Sprintf("can't check the albums of the server asset: %s", err)
	}
	if len(albums) > 0 {
		names := make([]string, len(albums))
		for i, a := range albums {
			names[i] = a.AlbumName
		}
		return lockSkipped, fmt.Sprintf("the asset belongs to albums %q, locking it would remove it from them", names)
	}

	err = c.SetAssetsVisibility(ctx, []string{server.ID}, assets.VisibilityLocked)
	if err != nil {
		return lockSkipped, fmt.Sprintf("can't lock the server asset: %s", err)
	}
	server.Visibility = assets.VisibilityLocked
	return lockApplied, ""
}

// ensureLocked locks the server asset matching a locked input asset. It never fails the upload.
func (uc *UpCmd) ensureLocked(ctx context.Context, a, serverAsset *assets.Asset) {
	outcome, reason := lockServerAsset(ctx, uc.client.Immich, a, serverAsset)
	switch outcome {
	case lockApplied:
		uc.app.Log().Info("server asset moved to the locked folder", "file", a.File, "ID", serverAsset.ID)
	case lockSkipped:
		uc.app.FileProcessor().Logger().Record(ctx, fileevent.ProcessedLockSkipped, a.File, "reason", reason)
	case lockNotNeeded:
	}
}
