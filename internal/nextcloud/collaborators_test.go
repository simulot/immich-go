package nextcloud

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAlbumCollaborators(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PROPFIND", r.Method)
		assert.Equal(t, "/remote.php/dav/photos/alice/albums/Road%20trip", r.URL.EscapedPath())
		assert.Equal(t, "0", r.Header.Get("Depth"))
		assert.Contains(t, r.Header.Get("Content-Type"), "application/xml")
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusMultiStatus)
		_, _ = fmt.Fprint(w, `<?xml version="1.0"?>
<d:multistatus xmlns:d="DAV:" xmlns:nc="http://nextcloud.org/ns">
  <d:response>
    <d:propstat>
      <d:prop>
        <nc:collaborators>
          <nc:collaborator>
            <nc:id>bob</nc:id>
            <nc:label>Bob</nc:label>
            <nc:type>0</nc:type>
          </nc:collaborator>
          <nc:collaborator>
            <nc:id>friends</nc:id>
            <nc:label>Friends</nc:label>
            <nc:type>1</nc:type>
          </nc:collaborator>
          <nc:collaborator>
            <nc:id></nc:id>
            <nc:label>Public link</nc:label>
            <nc:type>3</nc:type>
          </nc:collaborator>
        </nc:collaborators>
      </d:prop>
    </d:propstat>
  </d:response>
</d:multistatus>`)
	}))
	defer server.Close()

	client, err := NewClient(Config{
		BaseURL:  server.URL,
		Username: "alice",
		Password: "secret",
		Timeout:  time.Minute,
	})
	require.NoError(t, err)

	collaborators, err := GetAlbumCollaborators(context.Background(), client, "alice", "Road trip")
	require.NoError(t, err)
	require.Len(t, collaborators, 3)
	assert.Equal(t, AlbumCollaborator{ID: "bob", Label: "Bob", Type: AlbumCollaboratorTypeUser}, collaborators[0])
	assert.Equal(t, AlbumCollaborator{ID: "friends", Label: "Friends", Type: AlbumCollaboratorTypeGroup}, collaborators[1])
	assert.Equal(t, AlbumCollaborator{ID: "", Label: "Public link", Type: AlbumCollaboratorTypeLink}, collaborators[2])
}

func TestParseAlbumCollaboratorType(t *testing.T) {
	t.Parallel()

	assert.Equal(t, AlbumCollaboratorTypeUser, parseAlbumCollaboratorType("0"))
	assert.Equal(t, AlbumCollaboratorTypeGroup, parseAlbumCollaboratorType("group"))
	assert.Equal(t, AlbumCollaboratorTypeLink, parseAlbumCollaboratorType("3"))
	assert.Equal(t, AlbumCollaboratorTypeUnknown, parseAlbumCollaboratorType("unexpected"))
}
