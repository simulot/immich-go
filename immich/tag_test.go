package immich

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUntagAssets(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/api/tags/tag-1/assets", r.URL.Path)

		var body struct {
			IDs []string `json:"ids"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, []string{"asset-1", "asset-2"}, body.IDs)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"id":"asset-1","success":true},
			{"id":"asset-2","success":false,"error":"not_found"}
		]`))
	}))
	defer server.Close()

	client, err := NewImmichClient(server.URL, "api-key")
	require.NoError(t, err)

	resp, err := client.UntagAssets(context.Background(), "tag-1", []string{"asset-1", "asset-2"})
	require.NoError(t, err)
	require.Len(t, resp, 2)
	assert.True(t, resp[0].Success)
	assert.False(t, resp[1].Success)
	assert.Equal(t, "not_found", resp[1].Error)
}