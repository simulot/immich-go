package incremental_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/simulot/immich-go/app/root"
	"github.com/stretchr/testify/require"
)

// Exercise the real CLI and HTTP client across fresh command instances, not just
// the importer's fake client. The source ZIP can disappear before enrichment.
func TestCLISeparateZIPsAndHTTPPatchRetry(t *testing.T) {
	t.Setenv("IMMICH_GO_EXPERIMENTAL_TAKEOUT_API_KEY", "test-key")
	var mu sync.Mutex
	uploads, patches := 0, 0
	failPatch := true
	date, description := "2022-01-01T00:00:00Z", ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if r.Header.Get("x-api-key") != "test-key" {
			http.Error(w, "missing authentication", http.StatusUnauthorized)
			return
		}
		switch {
		case r.URL.Path == "/api/server/ping":
			_, _ = io.WriteString(w, `{"res":"pong"}`)
		case r.URL.Path == "/api/users/me":
			_, _ = io.WriteString(w, `{"id":"user-1","email":"test@example.test"}`)
		case r.URL.Path == "/api/server/media-types":
			_, _ = io.WriteString(w, `{"image":[".jpg"]}`)
		case r.URL.Path == "/api/server/about":
			_, _ = io.WriteString(w, `{"version":"v2.0.0"}`)
		case r.URL.Path == "/api/assets" && r.Method == http.MethodPost:
			reader, err := r.MultipartReader()
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			found := false
			for {
				part, err := reader.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				data, err := io.ReadAll(part)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				if part.FormName() == "assetData" && string(data) == "photo bytes" {
					found = true
				}
			}
			if !found {
				http.Error(w, "missing media bytes", http.StatusBadRequest)
				return
			}
			uploads++
			_, _ = io.WriteString(w, `{"id":"asset-1","status":"created"}`)
		case r.URL.Path == "/api/assets/asset-1" && r.Method == http.MethodPut:
			var fields map[string]json.RawMessage
			if err := json.NewDecoder(r.Body).Decode(&fields); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if len(fields) != 2 {
				http.Error(w, "unexpected patch fields", http.StatusBadRequest)
				return
			}
			patches++
			if failPatch {
				http.Error(w, "injected patch failure", http.StatusBadRequest)
				return
			}
			if err := json.Unmarshal(fields["dateTimeOriginal"], &date); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := json.Unmarshal(fields["description"], &description); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			fallthrough
		case r.URL.Path == "/api/assets/asset-1" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "asset-1", "exifInfo": map[string]string{"dateTimeOriginal": strings.TrimSuffix(date, "Z") + "+00:00", "description": description}})
		default:
			http.Error(w, "unexpected endpoint "+r.URL.Path, http.StatusNotFound)
		}
	}))
	defer server.Close()
	dir := t.TempDir()
	mediaPath := "Takeout/Google Photos/Photos from 2020/photo.jpg"
	writeZIP := func(name, entry, data string) string {
		t.Helper()
		filename := filepath.Join(dir, name)
		f, err := os.Create(filename)
		require.NoError(t, err)
		z := zip.NewWriter(f)
		_, err = z.Create("Takeout/")
		require.NoError(t, err)
		w, err := z.Create(entry)
		require.NoError(t, err)
		_, err = io.WriteString(w, data)
		require.NoError(t, err)
		require.NoError(t, z.Close())
		require.NoError(t, f.Close())
		return filename
	}
	first := writeZIP("first.zip", mediaPath, "photo bytes")
	second := writeZIP("second.zip", mediaPath+".json", `{"title":"photo.jpg","description":"from Google","photoTakenTime":{"timestamp":"1577836800"}}`)
	run := func(zipFiles ...string) error {
		t.Helper()
		cmd, _ := root.RootImmichGoCommand(context.Background())
		args := []string{"experimental-takeout", "--acknowledge-incomplete-import", "--state", filepath.Join(dir, "state.json"), "--export-id", "test-export", "--server", server.URL, "--log-file", filepath.Join(dir, "run.log")}
		cmd.SetArgs(append(args, zipFiles...))
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		return cmd.ExecuteContext(context.Background())
	}
	require.NoError(t, run(first))
	require.NoError(t, os.Remove(first))
	require.ErrorContains(t, run(second), "pending metadata")
	mu.Lock()
	failPatch = false
	mu.Unlock()
	require.NoError(t, run())
	require.NoError(t, run(second))
	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, 1, uploads)
	require.Equal(t, 2, patches)
	require.Equal(t, "2020-01-01T00:00:00Z", date)
	require.Equal(t, "from Google", description)
}
