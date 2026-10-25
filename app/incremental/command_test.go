package incremental_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	cliincremental "github.com/simulot/immich-go/app/incremental"
	"github.com/simulot/immich-go/app/root"
	importer "github.com/simulot/immich-go/internal/incremental"
	"github.com/stretchr/testify/require"
)

// Exercise the real CLI and HTTP client across fresh command instances, not just
// the importer's fake client. Unnamed media is deferred, and named duplicate
// occurrences enrich the same asset after the source ZIP is no longer local.
func TestCLINamingGateDuplicateSidecarAndHTTPPatchRetry(t *testing.T) {
	for _, staging := range []bool{false, true} {
		name := "re-present"
		if staging {
			name = "staging-single-download"
		}
		t.Run(name, func(t *testing.T) { runCLINamingScenario(t, staging) })
	}
}

func TestCLIStagingRequiresExplicitPositiveBudget(t *testing.T) {
	for _, flags := range [][]string{
		{"--staging-dir", "pending"},
		{"--max-staging-bytes", "10"},
		{"--staging-dir", "pending", "--max-staging-bytes", "0"},
		{"--staging-dir", "pending", "--max-staging-bytes", "-1"},
	} {
		t.Run(strings.Join(flags, " "), func(t *testing.T) {
			cmd, _ := root.RootImmichGoCommand(context.Background())
			cmd.SetArgs(append([]string{"experimental-takeout", "--state", filepath.Join(t.TempDir(), "state.json"), "--export-id", "test-export", "--acknowledge-incomplete-import"}, flags...))
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})
			require.ErrorContains(t, cmd.ExecuteContext(context.Background()), "--staging-dir and a positive --max-staging-bytes must be supplied together")
		})
	}
}

func runCLINamingScenario(t *testing.T, staging bool) {
	t.Helper()
	if staging && runtime.GOOS == "windows" {
		t.Skip("durable staging explicitly rejects Windows pending ACL and durable publication support")
	}
	t.Setenv("IMMICH_GO_EXPERIMENTAL_TAKEOUT_API_KEY", "test-key")
	var mu sync.Mutex
	uploads, patches := 0, 0
	failPatch := false
	var uploadedName string
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
					uploadedName = part.FileName()
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
			if len(fields) < 1 || len(fields) > 2 {
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
			if value, ok := fields["description"]; ok {
				if err := json.Unmarshal(value, &description); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
			}
			fallthrough
		case r.URL.Path == "/api/assets/asset-1" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "asset-1", "originalFileName": uploadedName, "exifInfo": map[string]string{"dateTimeOriginal": strings.TrimSuffix(date, "Z") + "+00:00", "description": description}})
		default:
			http.Error(w, "unexpected endpoint "+r.URL.Path, http.StatusNotFound)
		}
	}))
	defer server.Close()
	dir := t.TempDir()
	mediaPath := "Takeout/Google Photos/Photos from 2020/photo.jpg"
	writeZIP := func(name string, entries map[string]string) string {
		t.Helper()
		filename := filepath.Join(dir, name)
		f, err := os.Create(filename)
		require.NoError(t, err)
		z := zip.NewWriter(f)
		_, err = z.Create("Takeout/")
		require.NoError(t, err)
		for entry, data := range entries {
			w, err := z.Create(entry)
			require.NoError(t, err)
			_, err = io.WriteString(w, data)
			require.NoError(t, err)
		}
		require.NoError(t, z.Close())
		require.NoError(t, f.Close())
		return filename
	}
	const canonicalName = "full-original-photo-name.jpg"
	albumPath := "Takeout/Google Photos/Holidays/photo.jpg"
	first := writeZIP("first.zip", map[string]string{mediaPath: "photo bytes"})
	second := writeZIP("second.zip", map[string]string{mediaPath + ".json": `{"title":"full-original-photo-name.jpg","photoTakenTime":{"timestamp":"1577836800"}}`})
	album := writeZIP("album.zip", map[string]string{albumPath: "photo bytes", albumPath + ".json": `{"title":"full-original-photo-name.jpg","description":"from Google","photoTakenTime":{"timestamp":"1577836800"}}`})
	stagingDir := filepath.Join(dir, "staging")
	var stagingArgs []string
	if staging {
		stagingArgs = []string{"--staging-dir", stagingDir, "--max-staging-bytes", "64"}
	}
	retainedPayloads := func() int {
		t.Helper()
		count := 0
		err := filepath.WalkDir(stagingDir, func(name string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.Type().IsRegular() {
				data, readErr := os.ReadFile(name)
				if readErr != nil {
					return readErr
				}
				if string(data) == "photo bytes" {
					count++
				}
			}
			return nil
		})
		require.NoError(t, err)
		return count
	}
	run := func(zipFiles ...string) error {
		t.Helper()
		cmd, application := root.RootImmichGoCommand(context.Background())
		if staging {
			for _, child := range cmd.Commands() {
				if child.Name() == "experimental-takeout" {
					cmd.RemoveCommand(child)
					break
				}
			}
			cmd.AddCommand(cliincremental.NewCommand(application, importer.WithAvailableBytes(func(string) (uint64, error) {
				return 100_000_000_000, nil
			})))
		}
		args := make([]string, 0, 10+len(stagingArgs)+len(zipFiles))
		args = append(args, "experimental-takeout", "--acknowledge-incomplete-import", "--state", filepath.Join(dir, "state.json"), "--export-id", "test-export", "--server", server.URL, "--log-file", filepath.Join(dir, "run.log"))
		args = append(args, stagingArgs...)
		cmd.SetArgs(append(args, zipFiles...))
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		return cmd.ExecuteContext(context.Background())
	}
	require.NoError(t, run(first))
	mu.Lock()
	require.Zero(t, uploads, "unknown canonical filenames must not be committed to Immich")
	mu.Unlock()
	if staging {
		require.Equal(t, 1, retainedPayloads())
		require.NoError(t, os.Remove(first), "the first archive must not be needed after successful staging")
	}
	require.NoError(t, run(second))
	if staging {
		require.Zero(t, retainedPayloads(), "durable upload receipt releases managed staged bytes")
	} else {
		mu.Lock()
		require.Zero(t, uploads, "metadata discovery does not retain media bytes without staging")
		mu.Unlock()
		require.NoError(t, run(first))
		require.NoError(t, os.Remove(first))
	}
	mu.Lock()
	require.Equal(t, 1, uploads)
	require.Equal(t, canonicalName, uploadedName, "multipart upload must use the sidecar's full name")
	initialPatches := patches
	failPatch = true
	mu.Unlock()
	require.ErrorContains(t, run(album), "pending metadata")
	require.NoError(t, os.Remove(album))
	mu.Lock()
	failPatch = false
	mu.Unlock()
	require.NoError(t, run())
	require.NoError(t, run(second))
	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, 1, uploads, "a second source path must reuse the logical receipt")
	require.Equal(t, initialPatches+2, patches)
	require.Equal(t, "2020-01-01T00:00:00Z", date)
	require.Equal(t, "from Google", description)
}
