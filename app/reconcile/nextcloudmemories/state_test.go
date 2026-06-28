package nextcloudmemories

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyManagedAlbumStateAppendsAndReplacesManagedBlock(t *testing.T) {
	t.Parallel()

	description, err := applyManagedAlbumState("Summer trip to Norway", managedAlbumState{
		Source:        managedAlbumStateSource,
		SchemaVersion: 1,
		AlbumID:       12345,
		OwnerUID:      "alice",
		AlbumName:     "Roadtrip",
	})
	require.NoError(t, err)
	assert.Contains(t, description, "Summer trip to Norway")
	assert.Contains(t, description, memoriesManagedAlbumStateHeader)
	assert.Contains(t, description, `"album_id":12345`)

	replaced, err := applyManagedAlbumState(description, managedAlbumState{
		Source:        managedAlbumStateSource,
		SchemaVersion: 1,
		AlbumID:       12345,
		OwnerUID:      "alice",
		AlbumName:     "Roadtrip renamed",
	})
	require.NoError(t, err)
	assert.Contains(t, replaced, "Summer trip to Norway")
	assert.Contains(t, replaced, `"album_name":"Roadtrip renamed"`)
	assert.Equal(t, 1, strings.Count(replaced, memoriesManagedAlbumStateHeader))
}

func TestApplyManagedAlbumStatePreservesDescriptionAfterManagedBlock(t *testing.T) {
	t.Parallel()

	description := strings.Join([]string{
		"Summer trip to Norway",
		"",
		memoriesManagedAlbumStateHeader,
		`{"source":"nextcloud-memories","schema_version":1,"album_id":12345,"owner_uid":"alice","album_name":"Roadtrip"}`,
		memoriesManagedAlbumStateFooter,
		"",
		"Shared with family",
	}, "\n")

	updated, err := applyManagedAlbumState(description, managedAlbumState{
		Source:        managedAlbumStateSource,
		SchemaVersion: 1,
		AlbumID:       12345,
		OwnerUID:      "alice",
		AlbumName:     "Roadtrip renamed",
	})
	require.NoError(t, err)
	assert.Contains(t, updated, "Summer trip to Norway")
	assert.Contains(t, updated, "Shared with family")
	assert.Contains(t, updated, `"album_name":"Roadtrip renamed"`)
	assert.Equal(t, 1, strings.Count(updated, memoriesManagedAlbumStateHeader))
	assert.Less(t, strings.Index(updated, "Shared with family"), strings.Index(updated, memoriesManagedAlbumStateHeader))
}

func TestApplyManagedAlbumStateRejectsConflictingIdentity(t *testing.T) {
	t.Parallel()

	description, err := applyManagedAlbumState("", managedAlbumState{
		Source:        managedAlbumStateSource,
		SchemaVersion: 1,
		AlbumID:       12345,
		OwnerUID:      "alice",
		AlbumName:     "Roadtrip",
	})
	require.NoError(t, err)

	_, err = applyManagedAlbumState(description, managedAlbumState{
		Source:        managedAlbumStateSource,
		SchemaVersion: 1,
		AlbumID:       9,
		OwnerUID:      "alice",
		AlbumName:     "Roadtrip",
	})
	require.Error(t, err)
	assert.ErrorContains(t, err, "conflicting managed album state")
}

func TestMemoriesAlbumMembershipTag(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "", memoriesAlbumMembershipTag(0))
	assert.Equal(t, "immich-go/src/nextcloud-memories/album/42", memoriesAlbumMembershipTag(42))
}

func TestParseManagedAlbumStateMalformed(t *testing.T) {
	t.Parallel()

	_, _, err := parseManagedAlbumState(memoriesManagedAlbumStateHeader + "\nnot-json\n" + memoriesManagedAlbumStateFooter)
	require.Error(t, err)
	assert.ErrorIs(t, err, errMalformedManagedAlbumState)
}