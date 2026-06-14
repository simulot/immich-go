package immich_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/simulot/immich-go/immich"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddUsersToAlbum(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "/api/albums/album-1/users", r.URL.Path)
		var body struct {
			AlbumUsers []immich.AlbumUserAdd `json:"albumUsers"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Len(t, body.AlbumUsers, 1)
		assert.Equal(t, "user-2", body.AlbumUsers[0].UserID)
		assert.Equal(t, immich.AlbumUserRoleEditor, body.AlbumUsers[0].Role)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"album-1","albumName":"Roadtrip","albumUsers":[{"role":"owner","user":{"id":"owner-1","email":"owner@example.com"}},{"role":"editor","user":{"id":"user-2","email":"editor@example.com"}}]}`))
	}))
	defer server.Close()

	client, err := immich.NewImmichClient(server.URL, "test-key")
	require.NoError(t, err)

	album, err := client.AddUsersToAlbum(context.Background(), "album-1", []immich.AlbumUserAdd{{UserID: "user-2", Role: immich.AlbumUserRoleEditor}})
	require.NoError(t, err)
	require.Len(t, album.AlbumUsers, 2)
	assert.Equal(t, "user-2", album.AlbumUsers[1].User.ID)
	assert.Equal(t, immich.AlbumUserRoleEditor, album.AlbumUsers[1].Role)
}

func TestUpdateAlbumUser(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "/api/albums/album-1/user/user-2", r.URL.Path)
		var body struct {
			Role immich.AlbumUserRole `json:"role"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, immich.AlbumUserRoleViewer, body.Role)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := immich.NewImmichClient(server.URL, "test-key")
	require.NoError(t, err)

	require.NoError(t, client.UpdateAlbumUser(context.Background(), "album-1", "user-2", immich.AlbumUserRoleViewer))
}
