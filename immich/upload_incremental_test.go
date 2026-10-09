package immich_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/stretchr/testify/require"
)

func TestUploadDuplicateWithoutReadingMultipart(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/users/me":
			_, _ = io.WriteString(w, `{"id":"user-1"}`)
		case "/api/server/media-types":
			_, _ = io.WriteString(w, `{"image":[".jpg"]}`)
		case "/api/assets":
			// A real checksum duplicate may be returned before the server reads
			// any multipart bytes. The client writer then receives ClosedPipe.
			w.Header().Set("Connection", "close")
			_, _ = io.WriteString(w, `{"id":"existing-asset","status":"duplicate"}`)
		default:
			http.Error(w, "unexpected endpoint", http.StatusNotFound)
		}
	}))
	defer server.Close()
	client, err := immich.NewImmichClient(server.URL, "test-key")
	require.NoError(t, err)
	_, err = client.ValidateConnection(context.Background())
	require.NoError(t, err)
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "photo.jpg"), []byte(strings.Repeat("x", 8<<20)), 0o600))
	a := &assets.Asset{File: fshelper.FSName(os.DirFS(dir), "photo.jpg"), OriginalFileName: "photo.jpg"}
	_, err = a.GetChecksum()
	require.NoError(t, err)
	receipt, err := client.AssetUpload(context.Background(), a)
	require.NoError(t, a.Close())
	require.NoError(t, err)
	require.Equal(t, immich.UploadDuplicate, receipt.Status)
	require.Equal(t, "existing-asset", receipt.ID)
}
