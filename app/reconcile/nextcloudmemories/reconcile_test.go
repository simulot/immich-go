package nextcloudmemories

import (
	"context"
	"errors"
	"testing"

	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/immich"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandPassesCleanupFlag(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	parent := &cobra.Command{Use: "reconcile"}
	a := app.New(ctx, parent)
	called := false
	cleanup := false
	cmd := newCommand(ctx, a, func(ctx context.Context, a *app.Application, client *app.Client, cleanupMigrationTags bool) error {
		called = true
		cleanup = cleanupMigrationTags
		return nil
	})
	require.NoError(t, cmd.Flags().Set("cleanup-migration-tags", "true"))

	err := cmd.RunE(cmd, nil)
	require.NoError(t, err)
	assert.True(t, called)
	assert.True(t, cleanup)
}

func TestReconcilerRunAddsMissingAssets(t *testing.T) {
	t.Parallel()

	tagID := "tag-101"
	description, err := applyManagedAlbumState("", managedAlbumState{
		Source:        managedAlbumStateSource,
		SchemaVersion: 1,
		AlbumID:       101,
		OwnerUID:      "alice",
		AlbumName:     "Roadtrip",
	})
	require.NoError(t, err)

	client := &reconcileTestAlbumClient{
		albums: []immich.AlbumSimplified{{ID: "album-1", AlbumName: "Roadtrip", Description: description}},
		albumInfo: map[string]immich.AlbumContent{
			"album-1": {
				ID:        "album-1",
				AlbumName: "Roadtrip",
				Assets:    []*immich.Asset{{ID: "existing-asset"}},
			},
		},
		tags: []immich.TagSimplified{{ID: tagID, Value: memoriesAlbumMembershipTag(101), Name: "101"}},
	}
	assets := reconcileTestAssetLister{
		assetsByTagID: map[string][]*immich.Asset{
			tagID: {
				{ID: "existing-asset", OwnerID: "user-1"},
				{ID: "new-asset", OwnerID: "user-1"},
				{ID: "foreign-asset", OwnerID: "user-2"},
				{ID: "external-library", OwnerID: "user-1", LibraryID: "lib-1"},
			},
		},
	}

	r := reconciler{albums: client, assets: assets, userID: "user-1"}
	result, err := r.run(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, result.ManagedAlbums)
	assert.Equal(t, 1, result.MatchedAlbums)
	assert.Equal(t, 1, result.AssetsAdded)
	assert.Equal(t, 1, result.AlreadyPresent)
	assert.Equal(t, []string{"new-asset"}, client.addCalls["album-1"])
}

func TestReconcilerRunReportsUnresolvedAndMalformedAlbums(t *testing.T) {
	t.Parallel()

	description, err := applyManagedAlbumState("", managedAlbumState{
		Source:        managedAlbumStateSource,
		SchemaVersion: 1,
		AlbumID:       202,
		OwnerUID:      "alice",
		AlbumName:     "Shared",
	})
	require.NoError(t, err)

	client := &reconcileTestAlbumClient{
		albums: []immich.AlbumSimplified{
			{ID: "album-1", AlbumName: "Shared", Description: description},
			{ID: "album-2", AlbumName: "Broken", Description: memoriesManagedAlbumStateHeader + "\n{"},
		},
		albumInfo: map[string]immich.AlbumContent{
			"album-1": {ID: "album-1", AlbumName: "Shared"},
		},
	}

	r := reconciler{albums: client, assets: reconcileTestAssetLister{}, userID: "user-1"}
	result, err := r.run(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []int{202}, result.UnresolvedAlbumIDs)
	assert.Equal(t, []string{"Broken"}, result.MalformedAlbums)
	assert.Empty(t, client.addCalls)
}

func TestRunNextcloudMemoriesReconcileRejectsCleanup(t *testing.T) {
	t.Parallel()

	err := runNextcloudMemoriesReconcile(context.Background(), &app.Application{}, &app.Client{}, true)
	require.Error(t, err)
	assert.ErrorIs(t, err, errCleanupMigrationTagsUnsupported)
}

type reconcileTestAlbumClient struct {
	albums           []immich.AlbumSimplified
	albumInfo        map[string]immich.AlbumContent
	tags             []immich.TagSimplified
	addCalls         map[string][]string
	addResponsesByID map[string][]immich.UpdateAlbumResult
	addErr           error
}

func (f *reconcileTestAlbumClient) GetAllAlbums(context.Context) ([]immich.AlbumSimplified, error) {
	return f.albums, nil
}

func (f *reconcileTestAlbumClient) GetAlbumInfo(_ context.Context, id string, _ bool) (immich.AlbumContent, error) {
	return f.albumInfo[id], nil
}

func (f *reconcileTestAlbumClient) GetAllTags(context.Context) ([]immich.TagSimplified, error) {
	return f.tags, nil
}

func (f *reconcileTestAlbumClient) AddAssetToAlbum(_ context.Context, albumID string, assets []string) ([]immich.UpdateAlbumResult, error) {
	if f.addErr != nil {
		return nil, f.addErr
	}
	if f.addCalls == nil {
		f.addCalls = map[string][]string{}
	}
	f.addCalls[albumID] = append([]string(nil), assets...)
	if responses, ok := f.addResponsesByID[albumID]; ok {
		return responses, nil
	}
	responses := make([]immich.UpdateAlbumResult, 0, len(assets))
	for _, assetID := range assets {
		responses = append(responses, immich.UpdateAlbumResult{ID: assetID, Success: true})
	}
	return responses, nil
}

type reconcileTestAssetLister struct {
	assetsByTagID map[string][]*immich.Asset
	err           error
}

func (f reconcileTestAssetLister) ListAssetsByTag(_ context.Context, tagID string, filter func(*immich.Asset) error) error {
	if f.err != nil {
		return f.err
	}
	for _, asset := range f.assetsByTagID[tagID] {
		if err := filter(asset); err != nil {
			return err
		}
	}
	return nil
}

func TestReconcilerPropagatesAssetListErrors(t *testing.T) {
	t.Parallel()

	description, err := applyManagedAlbumState("", managedAlbumState{
		Source:        managedAlbumStateSource,
		SchemaVersion: 1,
		AlbumID:       5,
		OwnerUID:      "alice",
	})
	require.NoError(t, err)

	r := reconciler{
		albums: &reconcileTestAlbumClient{
			albums: []immich.AlbumSimplified{{ID: "album-1", AlbumName: "A", Description: description}},
			albumInfo: map[string]immich.AlbumContent{"album-1": {ID: "album-1", AlbumName: "A"}},
			tags: []immich.TagSimplified{{ID: "tag-5", Value: memoriesAlbumMembershipTag(5)}},
		},
		assets: reconcileTestAssetLister{err: errors.New("boom")},
		userID: "user-1",
	}

	_, err = r.run(context.Background())
	require.Error(t, err)
	assert.ErrorContains(t, err, "boom")
}