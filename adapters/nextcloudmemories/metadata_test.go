package nextcloudmemories

import (
	"testing"
	"time"

	"github.com/simulot/immich-go/internal/nextcloud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetadataFromMemoriesMapsArchivedAndOwnedAlbumsOnly(t *testing.T) {
	t.Parallel()

	info := &nextcloud.MemoriesImageInfo{
		Basename:  "IMG_0001.JPG",
		DateTaken: 1700000000,
		Exif: map[string]any{
			"Description":  "Sunset",
			"Rating":       "4",
			"GPSLatitude":  "12.34",
			"GPSLongitude": "56.78",
		},
		Tags: map[string]string{
			"1": "Travel",
		},
	}
	info.Clusters.Albums = []nextcloud.MemoriesAlbum{
		{AlbumID: 7, Name: "Roadtrip", User: "alice"},
		{AlbumID: 9, Name: "Roadtrip", User: "bob", UserDisplay: "Bob"},
	}

	md := metadataFromMemories(nextcloud.MemoriesPhoto{
		Archived:   true,
		IsFavorite: true,
		DateTaken:  1700000000,
	}, info, metadataMappingOptions{OwnerUID: "alice", SyncAlbums: true, SyncTags: true})

	require.NotNil(t, md)
	assert.True(t, md.Archived)
	assert.True(t, md.Favorited)
	assert.Equal(t, "Sunset", md.Description)
	assert.Equal(t, byte(4), md.Rating)
	assert.Equal(t, 12.34, md.Latitude)
	assert.Equal(t, 56.78, md.Longitude)
	assert.Equal(t, time.Unix(1700000000, 0).In(time.Local), md.DateTaken)
	require.Len(t, md.Tags, 1)
	assert.Equal(t, "Travel", md.Tags[0].Value)
	require.Len(t, md.Albums, 1)
	assert.Equal(t, "Roadtrip", md.Albums[0].Title)
	assert.Contains(t, md.Albums[0].Description, "\"album_id\":7")
	assert.Contains(t, md.Albums[0].Description, "\"owner_uid\":\"alice\"")
}

func TestMetadataFromMemoriesOptionallyTagsAlbumMembership(t *testing.T) {
	t.Parallel()

	info := &nextcloud.MemoriesImageInfo{}
	info.Clusters.Albums = []nextcloud.MemoriesAlbum{{AlbumID: 42, Name: "Roadtrip", User: "alice"}}

	md := metadataFromMemories(nextcloud.MemoriesPhoto{}, info, metadataMappingOptions{
		OwnerUID:           "alice",
		TagAlbumMembership: true,
	})

	require.NotNil(t, md)
	require.Len(t, md.Tags, 1)
	assert.Equal(t, "immich-go/src/nextcloud-memories/album/42", md.Tags[0].Value)
	assert.Empty(t, md.Albums)
}

func TestMetadataFromMemoriesSkipsSourceTagsByDefault(t *testing.T) {
	t.Parallel()

	info := &nextcloud.MemoriesImageInfo{
		Tags: map[string]string{
			"1": "Travel",
		},
	}

	md := metadataFromMemories(nextcloud.MemoriesPhoto{}, info, metadataMappingOptions{})

	require.NotNil(t, md)
	assert.Empty(t, md.Tags)
}

func TestMetadataFromMemoriesOptionallyIncludesSourceTags(t *testing.T) {
	t.Parallel()

	info := &nextcloud.MemoriesImageInfo{
		Tags: map[string]string{
			"1": "Travel",
		},
	}

	md := metadataFromMemories(nextcloud.MemoriesPhoto{}, info, metadataMappingOptions{SyncTags: true})

	require.NotNil(t, md)
	require.Len(t, md.Tags, 1)
	assert.Equal(t, "Travel", md.Tags[0].Value)
}

func TestMetadataFromMemoriesCanCreateAlbumsAndMembershipTagsTogether(t *testing.T) {
	t.Parallel()

	info := &nextcloud.MemoriesImageInfo{}
	info.Clusters.Albums = []nextcloud.MemoriesAlbum{{AlbumID: 42, Name: "Roadtrip", User: "alice"}}

	md := metadataFromMemories(nextcloud.MemoriesPhoto{}, info, metadataMappingOptions{
		OwnerUID:           "alice",
		SyncAlbums:         true,
		TagAlbumMembership: true,
	})

	require.NotNil(t, md)
	require.Len(t, md.Tags, 1)
	assert.Equal(t, "immich-go/src/nextcloud-memories/album/42", md.Tags[0].Value)
	require.Len(t, md.Albums, 1)
	assert.Equal(t, "Roadtrip", md.Albums[0].Title)
}
