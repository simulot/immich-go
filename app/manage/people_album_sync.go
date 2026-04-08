package manage

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/immich"
	"github.com/spf13/cobra"
)

// PeopleAlbumSyncCmd holds the state for the people-album-sync subcommand.
type PeopleAlbumSyncCmd struct {
	ConfigFile string
	client     app.Client
}

// NewPeopleAlbumSyncCommand creates the cobra command for syncing people to albums.
func NewPeopleAlbumSyncCommand(ctx context.Context, a *app.Application) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "people-album-sync [flags]",
		Short: "Sync people to album mappings",
		Long:  "Reads a YAML config that maps albums to people and ensures each album contains all photos of its mapped people. Additive only, never removes or deletes.",
	}

	syncCmd := &PeopleAlbumSyncCmd{}
	cmd.Flags().StringVar(&syncCmd.ConfigFile, "config", "", "Path to the manage YAML config file (required)")
	_ = cmd.MarkFlagRequired("config")
	syncCmd.client.RegisterFlags(cmd.Flags(), "")
	cmd.TraverseChildren = true

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		err := syncCmd.client.Open(ctx, a)
		if err != nil {
			return err
		}

		manageCfg, err := ParseConfig(syncCmd.ConfigFile)
		if err != nil {
			return err
		}

		if manageCfg.PeopleAlbumSync == nil {
			return fmt.Errorf("config: missing 'people-album-sync' section")
		}

		if err := manageCfg.PeopleAlbumSync.validate(); err != nil {
			return err
		}

		return syncCmd.run(ctx, a.Log().Logger, manageCfg.PeopleAlbumSync)
	}

	return cmd
}

func (syncCmd *PeopleAlbumSyncCmd) run(ctx context.Context, log *slog.Logger, cfg *PeopleAlbumSyncConfig) error {
	immichClient := syncCmd.client.Immich

	for _, mapping := range cfg.Albums {
		label := mapping.albumLabel()
		log.Info("Processing album mapping", "album", label)

		peopleIDs, err := resolvePeopleIDs(ctx, immichClient, mapping.People)
		if err != nil {
			return fmt.Errorf("album %q: %w", label, err)
		}
		log.Info("Resolved people", "album", label, "people_ids", peopleIDs)

		albumID, err := resolveOrCreateAlbum(ctx, immichClient, mapping, log)
		if err != nil {
			return fmt.Errorf("album %q: %w", label, err)
		}
		log.Info("Resolved album", "album", label, "album_id", albumID)

		albumInfo, err := immichClient.GetAlbumInfo(ctx, albumID, true)
		if err != nil {
			return fmt.Errorf("album %q: getting album info: %w", label, err)
		}
		existingAssets := toSet(albumInfo.AssetIDs)

		so := immich.SearchOptions().WithPeople(peopleIDs...)
		var mu sync.Mutex
		toAddSet := make(map[string]bool)
		err = immichClient.GetFilteredAssetsFn(ctx, so, func(asset *immich.Asset) error {
			mu.Lock()
			defer mu.Unlock()
			if !existingAssets[asset.ID] {
				toAddSet[asset.ID] = true
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("album %q: searching assets: %w", label, err)
		}

		toAdd := make([]string, 0, len(toAddSet))
		for id := range toAddSet {
			toAdd = append(toAdd, id)
		}

		if len(toAdd) == 0 {
			log.Info("Album already up to date", "album", label)
			continue
		}

		log.Info("Adding assets to album", "album", label, "count", len(toAdd))
		batchSize := 500
		for i := 0; i < len(toAdd); i += batchSize {
			end := min(i+batchSize, len(toAdd))
			batch := toAdd[i:end]
			results, err := immichClient.AddAssetToAlbum(ctx, albumID, batch)
			if err != nil {
				return fmt.Errorf("album %q: adding assets: %w", label, err)
			}
			added := 0
			for _, r := range results {
				if r.Success {
					added++
				}
			}
			log.Info("Batch complete", "album", label, "added", added, "batch_size", len(batch))
		}
	}

	return nil
}

// resolvePeopleIDs converts a PeopleSelector (names + direct IDs) into a flat list of Immich person UUIDs.
func resolvePeopleIDs(ctx context.Context, immichClient immich.ImmichInterface, sel PeopleSelector) ([]string, error) {
	var ids []string

	ids = append(ids, sel.IDs...)

	if len(sel.Names) > 0 {
		icP, ok := immichClient.(immich.ImmichPeopleInterface)
		if !ok {
			return nil, fmt.Errorf("immich client does not support people API")
		}
		peopleMap, err := icP.GetPeopleByNames(ctx, sel.Names)
		if err != nil {
			return nil, fmt.Errorf("resolving people names: %w", err)
		}
		for _, name := range sel.Names {
			person, found := peopleMap[name]
			if !found || person == nil {
				return nil, fmt.Errorf("person %q not found in Immich", name)
			}
			ids = append(ids, person.ID)
		}
	}

	return ids, nil
}

// resolveOrCreateAlbum finds an existing album by name/ID or creates a new one.
func resolveOrCreateAlbum(ctx context.Context, immichClient immich.ImmichInterface, mapping AlbumMapping, log *slog.Logger) (string, error) {
	if mapping.AlbumID != "" {
		return mapping.AlbumID, nil
	}

	albums, err := immichClient.GetAllAlbums(ctx)
	if err != nil {
		return "", fmt.Errorf("listing albums: %w", err)
	}
	for _, album := range albums {
		if album.AlbumName == mapping.Album {
			return album.ID, nil
		}
	}

	log.Info("Album not found, creating", "album", mapping.Album)
	album, err := immichClient.CreateAlbum(ctx, mapping.Album, "", nil)
	if err != nil {
		return "", fmt.Errorf("creating album: %w", err)
	}
	return album.ID, nil
}

func toSet(ids []string) map[string]bool {
	m := make(map[string]bool, len(ids))
	for _, id := range ids {
		m[id] = true
	}
	return m
}
