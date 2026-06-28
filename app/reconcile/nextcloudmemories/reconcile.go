package nextcloudmemories

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/immich"
)

var errCleanupMigrationTagsUnsupported = errors.New("cleanup of migration tags is not supported yet")

type albumClient interface {
	GetAllAlbums(ctx context.Context) ([]immich.AlbumSimplified, error)
	GetAlbumInfo(ctx context.Context, id string, withoutAssets bool) (immich.AlbumContent, error)
	GetAllTags(ctx context.Context) ([]immich.TagSimplified, error)
	AddAssetToAlbum(ctx context.Context, albumID string, assets []string) ([]immich.UpdateAlbumResult, error)
}

type taggedAssetLister interface {
	ListAssetsByTag(ctx context.Context, tagID string, filter func(*immich.Asset) error) error
}

type immichTaggedAssetLister struct {
	client immich.ImmichInterface
}

func (l immichTaggedAssetLister) ListAssetsByTag(ctx context.Context, tagID string, filter func(*immich.Asset) error) error {
	return l.client.GetFilteredAssetsFn(ctx, immich.SearchOptions().All().WithTags(tagID), filter)
}

type managedAlbum struct {
	album immich.AlbumSimplified
	state managedAlbumState
	info  immich.AlbumContent
	tagID string
	name  string
}

type reconciliationResult struct {
	ManagedAlbums       int
	MatchedAlbums       int
	AssetsAdded         int
	AlreadyPresent      int
	MalformedAlbums     []string
	UnresolvedAlbumIDs  []int
	PermissionFailures  []string
	AlbumsWithoutAssets []string
}

type reconciler struct {
	albums albumClient
	assets taggedAssetLister
	userID string
}

func runNextcloudMemoriesReconcile(ctx context.Context, a *app.Application, client *app.Client, cleanup bool) error {
	if cleanup {
		return fmt.Errorf("%w: Immich tag removal endpoints are not implemented in immich-go yet", errCleanupMigrationTagsUnsupported)
	}
	if err := client.Open(ctx, a); err != nil {
		return err
	}

	r := reconciler{
		albums: client.Immich,
		assets: immichTaggedAssetLister{client: client.Immich},
		userID: client.User.ID,
	}
	result, err := r.run(ctx)
	if err != nil {
		return err
	}
	printReconciliationSummary(a, result)
	return nil
}

func (r reconciler) run(ctx context.Context) (reconciliationResult, error) {
	result := reconciliationResult{}

	albums, err := r.albums.GetAllAlbums(ctx)
	if err != nil {
		return result, err
	}
	tags, err := r.albums.GetAllTags(ctx)
	if err != nil {
		return result, err
	}
	tagsByValue := make(map[string]immich.TagSimplified, len(tags))
	for _, tag := range tags {
		tagsByValue[tag.Value] = tag
	}

	managedAlbums, err := r.loadManagedAlbums(ctx, albums, tagsByValue, &result)
	if err != nil {
		return result, err
	}
	result.ManagedAlbums = len(managedAlbums)

	for _, album := range managedAlbums {
		if album.tagID == "" {
			result.UnresolvedAlbumIDs = append(result.UnresolvedAlbumIDs, album.state.AlbumID)
			continue
		}

		candidateIDs, alreadyPresent, err := r.findCandidateAssets(ctx, album)
		if err != nil {
			return result, err
		}
		result.AlreadyPresent += alreadyPresent
		result.MatchedAlbums++

		if len(candidateIDs) == 0 {
			result.AlbumsWithoutAssets = append(result.AlbumsWithoutAssets, album.name)
			continue
		}

		responses, err := r.albums.AddAssetToAlbum(ctx, album.album.ID, candidateIDs)
		if err != nil {
			return result, err
		}
		added, permissionFailure := summarizeAddResults(responses, candidateIDs)
		result.AssetsAdded += added
		if permissionFailure {
			result.PermissionFailures = append(result.PermissionFailures, album.name)
		}
	}

	sort.Ints(result.UnresolvedAlbumIDs)
	result.PermissionFailures = dedupeStrings(result.PermissionFailures)
	result.AlbumsWithoutAssets = dedupeStrings(result.AlbumsWithoutAssets)
	result.MalformedAlbums = dedupeStrings(result.MalformedAlbums)
	return result, nil
}

func (r reconciler) loadManagedAlbums(ctx context.Context, albums []immich.AlbumSimplified, tagsByValue map[string]immich.TagSimplified, result *reconciliationResult) ([]managedAlbum, error) {
	managed := make([]managedAlbum, 0)
	for _, album := range albums {
		_, state, err := parseManagedAlbumState(album.Description)
		if err != nil {
			result.MalformedAlbums = append(result.MalformedAlbums, album.AlbumName)
			continue
		}
		if state == nil || state.Source != managedAlbumStateSource || state.SchemaVersion != 1 || state.AlbumID <= 0 {
			continue
		}
		info, err := r.albums.GetAlbumInfo(ctx, album.ID, false)
		if err != nil {
			return nil, err
		}
		tagID := ""
		if tag := tagsByValue[memoriesAlbumMembershipTag(state.AlbumID)]; tag.ID != "" {
			tagID = tag.ID
		}
		managed = append(managed, managedAlbum{
			album: album,
			state: *state,
			info:  info,
			tagID: tagID,
			name:  managedAlbumDisplayName(album, *state),
		})
	}
	sort.Slice(managed, func(i, j int) bool {
		return managed[i].state.AlbumID < managed[j].state.AlbumID
	})
	return managed, nil
}

func (r reconciler) findCandidateAssets(ctx context.Context, album managedAlbum) ([]string, int, error) {
	existing := make(map[string]struct{}, len(album.info.Assets))
	for _, asset := range album.info.Assets {
		existing[asset.ID] = struct{}{}
	}

	seen := map[string]struct{}{}
	assetIDs := make([]string, 0)
	alreadyPresent := 0
	err := r.assets.ListAssetsByTag(ctx, album.tagID, func(asset *immich.Asset) error {
		if asset.OwnerID != r.userID || asset.LibraryID != "" {
			return nil
		}
		if _, ok := seen[asset.ID]; ok {
			return nil
		}
		seen[asset.ID] = struct{}{}
		if _, ok := existing[asset.ID]; ok {
			alreadyPresent++
			return nil
		}
		assetIDs = append(assetIDs, asset.ID)
		return nil
	})
	if err != nil {
		return nil, 0, err
	}
	return assetIDs, alreadyPresent, nil
}

func managedAlbumDisplayName(album immich.AlbumSimplified, state managedAlbumState) string {
	if strings.TrimSpace(album.AlbumName) != "" {
		return fmt.Sprintf("%s (source album %d)", strings.TrimSpace(album.AlbumName), state.AlbumID)
	}
	if strings.TrimSpace(state.AlbumName) != "" {
		return fmt.Sprintf("%s (source album %d)", strings.TrimSpace(state.AlbumName), state.AlbumID)
	}
	return fmt.Sprintf("album %d", state.AlbumID)
}

func summarizeAddResults(responses []immich.UpdateAlbumResult, requested []string) (int, bool) {
	if len(responses) == 0 {
		return len(requested), false
	}
	added := 0
	permissionFailure := false
	for _, response := range responses {
		if response.Success {
			added++
			continue
		}
		if response.Error == "duplicate" {
			continue
		}
		if response.Error == "no_permission" {
			permissionFailure = true
		}
	}
	return added, permissionFailure
}

func printReconciliationSummary(a *app.Application, result reconciliationResult) {
	log := a.Log()
	log.Message("Reconciled %d managed Nextcloud Memories albums", result.ManagedAlbums)
	log.Message("Matched %d managed albums with migration tags", result.MatchedAlbums)
	log.Message("Added %d assets to shared albums", result.AssetsAdded)
	if result.AlreadyPresent > 0 {
		log.Message("Skipped %d assets already present in destination albums", result.AlreadyPresent)
	}
	if len(result.MalformedAlbums) > 0 {
		log.Message("Ignored albums with malformed managed state: %s", strings.Join(result.MalformedAlbums, ", "))
	}
	if len(result.UnresolvedAlbumIDs) > 0 {
		ids := make([]string, 0, len(result.UnresolvedAlbumIDs))
		for _, id := range result.UnresolvedAlbumIDs {
			ids = append(ids, fmt.Sprintf("%d", id))
		}
		log.Message("Managed albums missing synthetic membership tags: %s", strings.Join(ids, ", "))
	}
	if len(result.PermissionFailures) > 0 {
		log.Message("Albums reporting permission failures: %s", strings.Join(result.PermissionFailures, ", "))
	}
	if len(result.AlbumsWithoutAssets) > 0 {
		log.Message("Managed albums with no matching user-owned assets to add: %s", strings.Join(result.AlbumsWithoutAssets, ", "))
	}
}

func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}