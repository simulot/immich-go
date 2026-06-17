package nextcloudmemories

import (
	"strings"
	"testing"

	"github.com/simulot/immich-go/internal/nextcloud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyManagedAlbumStateAppendsAndReplacesManagedBlock(t *testing.T) {
	t.Parallel()

	description, err := applyManagedAlbumState("Summer trip to Norway", managedAlbumState{
		Source:        "nextcloud-memories",
		SchemaVersion: 1,
		AlbumID:       12345,
		OwnerUID:      "alice",
		AlbumName:     "Roadtrip",
	})
	require.NoError(t, err)
	assert.Contains(t, description, "Summer trip to Norway")
	assert.Contains(t, description, memoriesManagedAlbumStateHeader)
	assert.Contains(t, description, "\"album_id\":12345")

	replaced, err := applyManagedAlbumState(description, managedAlbumState{
		Source:        "nextcloud-memories",
		SchemaVersion: 1,
		AlbumID:       12345,
		OwnerUID:      "alice",
		AlbumName:     "Roadtrip renamed",
	})
	require.NoError(t, err)
	assert.Contains(t, replaced, "Summer trip to Norway")
	assert.Contains(t, replaced, "\"album_name\":\"Roadtrip renamed\"")
	assert.Equal(t, 1, stringsCount(replaced, memoriesManagedAlbumStateHeader))
}

func TestApplyManagedAlbumStateRejectsConflictingIdentity(t *testing.T) {
	t.Parallel()

	description, err := applyManagedAlbumState("", managedAlbumState{
		Source:        "nextcloud-memories",
		SchemaVersion: 1,
		AlbumID:       12345,
		OwnerUID:      "alice",
		AlbumName:     "Roadtrip",
	})
	require.NoError(t, err)

	_, err = applyManagedAlbumState(description, managedAlbumState{
		Source:        "nextcloud-memories",
		SchemaVersion: 1,
		AlbumID:       9,
		OwnerUID:      "alice",
		AlbumName:     "Roadtrip",
	})
	require.Error(t, err)
	assert.ErrorContains(t, err, "conflicting managed album state")
}

func TestMemoriesOwnedAlbumDescriptionOnlyStampsOwnedAlbums(t *testing.T) {
	t.Parallel()

	owned := memoriesOwnedAlbumDescription(nextcloud.MemoriesAlbum{
		AlbumID: 7,
		Name:    "Roadtrip",
		User:    "alice",
	}, "alice")
	require.NotEmpty(t, owned)
	assert.Contains(t, owned, "\"album_id\":7")

	shared := memoriesOwnedAlbumDescription(nextcloud.MemoriesAlbum{
		AlbumID: 9,
		Name:    "Roadtrip",
		User:    "bob",
	}, "alice")
	assert.Empty(t, shared)
}

func TestMemoriesOwnedAlbumTitleSkipsSharedAlbums(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Roadtrip", memoriesOwnedAlbumTitle(nextcloud.MemoriesAlbum{
		AlbumID: 7,
		Name:    "Roadtrip",
		User:    "alice",
	}, "alice"))

	assert.Empty(t, memoriesOwnedAlbumTitle(nextcloud.MemoriesAlbum{
		AlbumID:     9,
		Name:        "Roadtrip",
		User:        "bob",
		UserDisplay: "Bob",
	}, "alice"))
}

func TestMemoriesAlbumMembershipTag(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "", memoriesAlbumMembershipTag(0))
	assert.Equal(t, "immich-go/src/nextcloud-memories/album/42", memoriesAlbumMembershipTag(42))
}

func stringsCount(value string, needle string) int {
	count := 0
	start := 0
	for {
		idx := strings.Index(value[start:], needle)
		if idx == -1 {
			return count
		}
		count++
		start += idx + len(needle)
	}
}
