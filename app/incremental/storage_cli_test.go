package incremental_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	command "github.com/simulot/immich-go/app/incremental"
	"github.com/simulot/immich-go/app/root"
	importer "github.com/simulot/immich-go/internal/incremental"
	"github.com/stretchr/testify/require"
)

const (
	storageCLIPlenty      = uint64(100_000_000_000)
	storageCLIReserve     = uint64(10_000_000_000)
	storageCLIMediaPath   = "Takeout/Google Photos/Photos from 2020/photo.jpg"
	storageCLICanonical   = "the-full-original-photo-name.jpg"
	storageCLIPayload     = "storage CLI photo bytes"
	storageCLIArchiveA    = "export media 09.zip"
	storageCLIArchiveB    = "export metadata 02.zip"
	storageCLISidecarJSON = `{"title":"the-full-original-photo-name.jpg","photoTakenTime":{"timestamp":"1577836800"}}`
)

func TestCLIStorageDecimalThresholdsReachCommandErrorWriter(t *testing.T) {
	skipCLIStagingOnWindows(t)
	for _, tc := range []struct {
		name  string
		free  uint64
		label string
	}{
		{"ample-space", storageCLIPlenty, ""},
		{"one-byte-above-warning", 50_000_000_001, ""},
		{"warning-boundary", 50_000_000_000, "WARNING"},
		{"one-byte-below-warning", 49_999_999_999, "WARNING"},
		{"one-byte-above-danger", 20_000_000_001, "WARNING"},
		{"danger-boundary", 20_000_000_000, "DANGER"},
		{"one-byte-below-danger", 19_999_999_999, "DANGER"},
		{"exactly-ten-GB-left-after-payload", storageCLIReserve + uint64(len(storageCLIPayload)), "DANGER"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newStorageCLIFixture(t)
			archive := f.writeZIP(storageCLIArchiveA, map[string]string{storageCLIMediaPath: storageCLIPayload})
			result := f.run(tc.free, append(f.stagingFlags(128), archive)...)
			require.NoError(t, result.err, result.stderr)
			if tc.label == "" {
				require.NotContains(t, result.stderr, "WARNING:")
				require.NotContains(t, result.stderr, "DANGER:")
			} else {
				require.Contains(t, result.stderr, tc.label+":", "storage notices must honor cmd.SetErr")
				require.Contains(t, result.stderr, "GB free")
				if tc.label == "WARNING" {
					require.NotContains(t, result.stderr, "DANGER:")
				}
			}
			require.NotContains(t, result.stderr, "FALLBACK:")
			require.NotContains(t, result.stdout, "WARNING:")
			require.NotContains(t, result.stdout, "DANGER:")
			require.Equal(t, 1, f.payloadCopies(storageCLIPayload, false), "the reserve is decimal GB and equality is admissible")
			require.Empty(t, f.uploads(), "a low-space notice must not bypass the canonical filename gate")
		})
	}
}

func TestCLIStorageReserveFallbackPersistsCatalogAndReplaysOnlyUncachedArchive(t *testing.T) {
	skipCLIStagingOnWindows(t)
	f := newStorageCLIFixture(t)
	first := f.writeZIP(storageCLIArchiveA, map[string]string{storageCLIMediaPath: storageCLIPayload})
	firstZIP, err := os.ReadFile(first)
	require.NoError(t, err)
	second := f.writeZIP(storageCLIArchiveB, map[string]string{storageCLIMediaPath + ".json": storageCLISidecarJSON})
	manifest := f.writeArchiveList(storageCLIArchiveA, storageCLIArchiveB)
	// There is more than 10 GB free now, but retaining this payload would leave
	// one byte less than the reserve. Observe the public FALLBACK notification
	// while the command is running, before any cleanup could conceal a write.
	payloadsAtFallback := -1
	result := f.runWithProbe(func(string) (uint64, error) {
		return storageCLIReserve + uint64(len(storageCLIPayload)) - 1, nil
	}, func(chunk string) {
		if strings.Contains(chunk, "FALLBACK:") {
			payloadsAtFallback = f.payloadCopies(storageCLIPayload, true)
		}
	}, append(f.stagingFlags(128), "--archive-list", manifest, first)...)
	require.NoError(t, result.err, result.stderr)
	require.Contains(t, result.stderr, "DANGER:")
	require.Contains(t, result.stderr, "FALLBACK:")
	require.Contains(t, result.stderr, "10 GB")
	require.Zero(t, payloadsAtFallback, "projected reserve failure must precede writing even a partial payload")
	require.Zero(t, f.payloadCopies(storageCLIPayload, true), "fallback must leave neither full nor partial retained media")
	require.Empty(t, f.uploads(), "unknown names must never be uploaded to make storage available")
	requireNextArchive(t, result.stdout, storageCLIArchiveB, false)
	require.FileExists(t, first, "the command never deletes a user's ZIP")

	require.NoError(t, os.Remove(first))
	require.NoError(t, os.Remove(manifest))
	// Fresh commands omit both staging and archive-list flags. Even with ample
	// free space, the selected fallback and the next unscanned name must survive.
	result = f.run(storageCLIPlenty)
	require.NoError(t, result.err, result.stderr)
	require.Contains(t, result.stderr, "FALLBACK:")
	requireNextArchive(t, result.stdout, storageCLIArchiveB, false)
	require.Empty(t, f.uploads())

	result = f.run(11_000_000_000, second)
	require.NoError(t, result.err, result.stderr)
	require.Contains(t, result.stderr, "DANGER:", "danger must remain visible after restarting in fallback mode")
	require.Empty(t, f.uploads(), "a metadata-only ZIP cannot upload uncached media from a missing source")
	requireNextArchive(t, result.stdout, storageCLIArchiveA, true)
	require.NoError(t, os.Remove(second))
	// This replay request proves the first command durably cataloged its media
	// despite refusing retention. Restore the exact original ZIP for its digest.
	require.NoError(t, os.WriteFile(first, firstZIP, 0o600))
	result = f.run(11_000_000_000, first)
	require.NoError(t, result.err, result.stderr)
	require.Equal(t, []storageCLIUpload{{name: storageCLICanonical, data: storageCLIPayload}}, f.uploads())
	requireNoNextSource(t, result.stdout, storageCLIArchiveA, storageCLIArchiveB)
	require.Zero(t, f.payloadCopies(storageCLIPayload, false), "transient second-pass workspace must be released")

	require.NoError(t, os.Remove(first))
	result = f.run(11_000_000_000)
	require.NoError(t, result.err, result.stderr)
	require.Len(t, f.uploads(), 1, "a durable receipt must prevent another multipart upload after restart")
	requireNoNextSource(t, result.stdout, storageCLIArchiveA, storageCLIArchiveB)
}

func TestCLIStorageOnePassUsesCacheAfterSourceZIPAndArchiveListAreRemoved(t *testing.T) {
	skipCLIStagingOnWindows(t)
	f := newStorageCLIFixture(t)
	first := f.writeZIP(storageCLIArchiveA, map[string]string{storageCLIMediaPath: storageCLIPayload})
	second := f.writeZIP(storageCLIArchiveB, map[string]string{storageCLIMediaPath + ".json": storageCLISidecarJSON})
	manifest := f.writeArchiveList(storageCLIArchiveA, storageCLIArchiveB)
	result := f.run(storageCLIPlenty, append(f.stagingFlags(128), "--archive-list", manifest, first)...)
	require.NoError(t, result.err, result.stderr)
	require.Empty(t, f.uploads())
	require.Equal(t, 1, f.payloadCopies(storageCLIPayload, false))
	requireNextArchive(t, result.stdout, storageCLIArchiveB, false)
	require.NoError(t, os.Remove(first))
	require.NoError(t, os.Remove(manifest))

	result = f.run(storageCLIPlenty)
	require.NoError(t, result.err, result.stderr)
	requireNextArchive(t, result.stdout, storageCLIArchiveB, false)
	require.Equal(t, 1, f.payloadCopies(storageCLIPayload, false))

	result = f.run(storageCLIPlenty, second)
	require.NoError(t, result.err, result.stderr)
	require.Equal(t, []storageCLIUpload{{name: storageCLICanonical, data: storageCLIPayload}}, f.uploads())
	require.Zero(t, f.payloadCopies(storageCLIPayload, false), "only a durable upload receipt releases the cached copy")
	requireNoNextSource(t, result.stdout, storageCLIArchiveA, storageCLIArchiveB)
	require.NoError(t, os.Remove(second))
	result = f.run(storageCLIPlenty)
	require.NoError(t, result.err, result.stderr)
	require.Len(t, f.uploads(), 1)
	requireNoNextSource(t, result.stdout, storageCLIArchiveA, storageCLIArchiveB)
}

func TestCLIStorageCapFallbackPreservesCacheAndStopsNewRetentionAfterRestart(t *testing.T) {
	skipCLIStagingOnWindows(t)
	f := newStorageCLIFixture(t)
	const otherPath = "Takeout/Google Photos/Photos from 2020/other.jpg"
	const otherPayload = "uncached second photo bytes"
	const secondName = "uncached media 05.zip"
	const metadataName = "all metadata 01.zip"
	first := f.writeZIP(storageCLIArchiveA, map[string]string{storageCLIMediaPath: storageCLIPayload})
	second := f.writeZIP(secondName, map[string]string{otherPath: otherPayload})
	metadata := f.writeZIP(metadataName, map[string]string{
		storageCLIMediaPath + ".json": storageCLISidecarJSON,
		otherPath + ".json":           `{"title":"the-second-original-name.jpg","photoTakenTime":{"timestamp":"1577836800"}}`,
	})
	manifest := f.writeArchiveList(storageCLIArchiveA, secondName, metadataName)
	result := f.run(storageCLIPlenty, append(f.stagingFlags(len(storageCLIPayload)), "--archive-list", manifest, first)...)
	require.NoError(t, result.err, result.stderr)
	require.Equal(t, 1, f.payloadCopies(storageCLIPayload, false))
	require.NoError(t, os.Remove(first))

	result = f.run(storageCLIPlenty, second)
	require.NoError(t, result.err, "a staging cap is a graceful fallback, not a hard import failure: %s", result.stderr)
	require.Contains(t, result.stderr, "FALLBACK:")
	require.Equal(t, 1, f.payloadCopies(storageCLIPayload, false), "capacity fallback must never evict existing cached media")
	require.Zero(t, f.payloadCopies(otherPayload, false))
	require.Empty(t, f.uploads())
	requireNextArchive(t, result.stdout, metadataName, false)

	// Raising the cap does not silently undo a durable workflow choice. Re-read
	// the uncached ZIP with ample space and verify new retention remains stopped.
	result = f.run(storageCLIPlenty, append(f.stagingFlags(1024), second)...)
	require.NoError(t, result.err, result.stderr)
	require.Contains(t, result.stderr, "FALLBACK:")
	require.Equal(t, 1, f.payloadCopies(storageCLIPayload, false))
	require.Zero(t, f.payloadCopies(otherPayload, false))
	requireNextArchive(t, result.stdout, metadataName, false)
	require.NoError(t, os.Remove(second))

	result = f.run(storageCLIPlenty, metadata)
	require.NoError(t, result.err, result.stderr)
	require.Equal(t, []storageCLIUpload{{name: storageCLICanonical, data: storageCLIPayload}}, f.uploads(), "previously cached media must still upload during fallback")
	require.Zero(t, f.payloadCopies(storageCLIPayload, false))
	requireNextArchive(t, result.stdout, secondName, true)
}

func TestCLIStorageUnknownFreeSpaceIsActionableAndRetryable(t *testing.T) {
	skipCLIStagingOnWindows(t)
	f := newStorageCLIFixture(t)
	archive := f.writeZIP(storageCLIArchiveA, map[string]string{storageCLIMediaPath: storageCLIPayload})
	result := f.runWithProbe(func(string) (uint64, error) {
		return 0, errors.New("fixture availability probe failed")
	}, nil, append(f.stagingFlags(128), archive)...)
	require.ErrorContains(t, result.err, "available storage")
	require.ErrorContains(t, result.err, "fixture availability probe failed")
	require.Contains(t, result.stderr, "available storage", "the actual CLI must display the probe error")
	require.Contains(t, result.stderr, "fixture availability probe failed")
	require.NotContains(t, result.stderr, "FALLBACK:", "an unknown measurement is not measured low space")
	require.Empty(t, f.uploads())
	require.Zero(t, f.payloadCopies(storageCLIPayload, false))
	require.FileExists(t, archive)

	result = f.run(storageCLIPlenty, archive)
	require.NoError(t, result.err, "probe failure must release locks and permit a corrected retry: %s", result.stderr)
	require.Equal(t, 1, f.payloadCopies(storageCLIPayload, false))
	require.Empty(t, f.uploads())
}

func TestCLIStorageWithoutArchiveListDoesNotInventNextDownloadName(t *testing.T) {
	f := newStorageCLIFixture(t)
	archive := f.writeZIP("operator-chosen-Q7.zip", map[string]string{storageCLIMediaPath: storageCLIPayload})
	result := f.run(storageCLIPlenty, archive)
	require.NoError(t, result.err, result.stderr)
	require.Empty(t, f.uploads())
	line := nextStorageCLILine(t, result.stdout)
	require.Contains(t, line, "--archive-list", "explain how to identify unseen archives")
	require.NotRegexp(t, `"[^"\n]+\.zip"`, line, "neither replay an unresolved source early nor guess an unseen filename")
	require.NotRegexp(t, `https?://`, line, "archive guidance uses inventory filenames, not expiring download URLs")
}

func TestCLIStorageDamagedCacheNamesSourceAndRecoversWithoutDuplicateUpload(t *testing.T) {
	skipCLIStagingOnWindows(t)
	for _, damage := range []string{"missing", "size", "checksum"} {
		t.Run(damage, func(t *testing.T) {
			f := newStorageCLIFixture(t)
			first := f.writeZIP(storageCLIArchiveA, map[string]string{storageCLIMediaPath: storageCLIPayload})
			original, err := os.ReadFile(first)
			require.NoError(t, err)
			second := f.writeZIP(storageCLIArchiveB, map[string]string{storageCLIMediaPath + ".json": storageCLISidecarJSON})
			manifest := f.writeArchiveList(storageCLIArchiveA, storageCLIArchiveB)
			result := f.run(storageCLIPlenty, append(f.stagingFlags(128), "--archive-list", manifest, first)...)
			require.NoError(t, result.err, result.stderr)
			require.NoError(t, os.Remove(first))
			cached, err := filepath.Glob(filepath.Join(f.staging, "*.payload"))
			require.NoError(t, err)
			require.Len(t, cached, 1)
			switch damage {
			case "missing":
				require.NoError(t, os.Remove(cached[0]))
			case "size":
				require.NoError(t, os.WriteFile(cached[0], []byte("bad"), 0o600))
			case "checksum":
				require.NoError(t, os.WriteFile(cached[0], bytes.Repeat([]byte{'?'}, len(storageCLIPayload)), 0o600))
			}
			result = f.run(storageCLIPlenty, second)
			require.ErrorContains(t, result.err, "staged payload")
			requireNextArchive(t, result.stdout, storageCLIArchiveA, true)
			require.NotContains(t, nextStorageCLILine(t, result.stdout), "no ZIP download is needed")
			require.Empty(t, f.uploads())
			// The exact ZIP can be restored from a retained local copy or a fresh
			// download. The first repair reports the damage it encountered before
			// uploading; its durable receipt still prevents a duplicate retry.
			require.NoError(t, os.WriteFile(first, original, 0o600))
			result = f.run(storageCLIPlenty, first)
			require.ErrorContains(t, result.err, "staged payload")
			require.Equal(t, []storageCLIUpload{{name: storageCLICanonical, data: storageCLIPayload}}, f.uploads())
			requireNoNextSource(t, result.stdout, storageCLIArchiveA, storageCLIArchiveB)
			result = f.run(storageCLIPlenty)
			require.NoError(t, result.err, result.stderr)
			require.Len(t, f.uploads(), 1)
		})
	}
}

type storageCLIResult struct {
	stdout string
	stderr string
	err    error
}

type storageCLIUpload struct {
	name string
	data string
}

type storageCLIRemoteAsset struct {
	name        string
	date        string
	description string
}

type storageCLIFixture struct {
	t          *testing.T
	dir        string
	staging    string
	config     string
	server     *httptest.Server
	runs       int
	mu         sync.Mutex
	received   []storageCLIUpload
	remote     map[string]*storageCLIRemoteAsset
	httpErrors []string
}

func newStorageCLIFixture(t *testing.T) *storageCLIFixture {
	t.Helper()
	const fixtureAPIKey = "storage-cli-private-fixture-only"
	t.Setenv("IMMICH_GO_EXPERIMENTAL_TAKEOUT_API_KEY", fixtureAPIKey)
	f := &storageCLIFixture{t: t, dir: t.TempDir(), remote: make(map[string]*storageCLIRemoteAsset)}
	f.staging = filepath.Join(f.dir, "staging")
	f.config = filepath.Join(f.dir, "empty-config.yaml")
	require.NoError(t, os.WriteFile(f.config, []byte("{}\n"), 0o600))
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		fail := func(message string, status int) {
			f.httpErrors = append(f.httpErrors, message)
			http.Error(w, message, status)
		}
		if r.Header.Get("x-api-key") != fixtureAPIKey {
			fail("fixture authentication required", http.StatusUnauthorized)
			return
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/server/ping":
			_, _ = io.WriteString(w, `{"res":"pong"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/users/me":
			_, _ = io.WriteString(w, `{"id":"storage-user","email":"storage@example.test"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/server/media-types":
			_, _ = io.WriteString(w, `{"image":[".jpg"]}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/server/about":
			_, _ = io.WriteString(w, `{"version":"v2.0.0"}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/assets":
			reader, err := r.MultipartReader()
			if err != nil {
				fail("expected real multipart upload: "+err.Error(), http.StatusBadRequest)
				return
			}
			var upload storageCLIUpload
			mediaParts := 0
			for {
				part, err := reader.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					fail("read multipart part: "+err.Error(), http.StatusBadRequest)
					return
				}
				data, err := io.ReadAll(part)
				if err != nil {
					fail("read multipart bytes: "+err.Error(), http.StatusBadRequest)
					return
				}
				if part.FormName() == "assetData" {
					mediaParts++
					upload = storageCLIUpload{name: part.FileName(), data: string(data)}
				}
			}
			if mediaParts != 1 || upload.name == "" || upload.data == "" {
				fail("expected exactly one named, nonempty media part", http.StatusBadRequest)
				return
			}
			f.received = append(f.received, upload)
			id := fmt.Sprintf("storage-asset-%d", len(f.received))
			f.remote[id] = &storageCLIRemoteAsset{name: upload.name, date: "2022-01-01T00:00:00Z"}
			_ = json.NewEncoder(w).Encode(map[string]string{"id": id, "status": "created"})
		case strings.HasPrefix(r.URL.Path, "/api/assets/") && (r.Method == http.MethodGet || r.Method == http.MethodPut):
			id := strings.TrimPrefix(r.URL.Path, "/api/assets/")
			asset, ok := f.remote[id]
			if !ok {
				fail("asset receipt not found", http.StatusNotFound)
				return
			}
			if r.Method == http.MethodPut {
				var fields map[string]json.RawMessage
				if err := json.NewDecoder(r.Body).Decode(&fields); err != nil {
					fail("decode metadata update: "+err.Error(), http.StatusBadRequest)
					return
				}
				for field, value := range fields {
					var target *string
					switch field {
					case "dateTimeOriginal":
						target = &asset.date
					case "description":
						target = &asset.description
					default:
						fail("unexpected metadata field "+field, http.StatusBadRequest)
						return
					}
					if err := json.Unmarshal(value, target); err != nil {
						fail("invalid metadata value: "+err.Error(), http.StatusBadRequest)
						return
					}
				}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": id, "originalFileName": asset.name,
				"exifInfo": map[string]string{"dateTimeOriginal": strings.TrimSuffix(asset.date, "Z") + "+00:00", "description": asset.description},
			})
		default:
			fail("unexpected fixture endpoint "+r.Method+" "+r.URL.Path, http.StatusNotFound)
		}
	}))
	t.Cleanup(func() {
		f.server.Close()
		f.mu.Lock()
		defer f.mu.Unlock()
		require.Empty(t, f.httpErrors, "the CLI must use the authenticated fixture's supported API routes")
	})
	return f
}

func (f *storageCLIFixture) run(free uint64, args ...string) storageCLIResult {
	f.t.Helper()
	return f.runWithProbe(func(string) (uint64, error) { return free, nil }, nil, args...)
}

func (f *storageCLIFixture) runWithProbe(probe func(string) (uint64, error), observeStderr func(string), args ...string) storageCLIResult {
	f.t.Helper()
	ctx := context.Background()
	cmd, application := root.RootImmichGoCommand(ctx)
	replaced := false
	for _, child := range cmd.Commands() {
		if child.Name() == "experimental-takeout" {
			cmd.RemoveCommand(child)
			replaced = true
			break
		}
	}
	require.True(f.t, replaced, "construct and replace the real root command's experimental child")
	cmd.AddCommand(command.NewCommand(application, importer.WithAvailableBytes(probe)))
	var stdout bytes.Buffer
	stderr := storageCLIErrorWriter{observe: observeStderr}
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	f.runs++
	base := make([]string, 0, 12+len(args))
	base = append(base,
		"experimental-takeout", "--acknowledge-incomplete-import",
		"--config", f.config, "--state", filepath.Join(f.dir, "state.json"), "--export-id", "storage-fixture-export",
		"--server", f.server.URL, "--log-file", filepath.Join(f.dir, fmt.Sprintf("run-%d.log", f.runs)),
	)
	cmd.SetArgs(append(base, args...))
	err := cmd.ExecuteContext(ctx)
	require.NoError(f.t, application.Log().Close(ctx, cmd, application))
	return storageCLIResult{stdout: stdout.String(), stderr: stderr.String(), err: err}
}

type storageCLIErrorWriter struct {
	bytes.Buffer
	observe func(string)
}

func (w *storageCLIErrorWriter) Write(p []byte) (int, error) {
	n, err := w.Buffer.Write(p)
	if w.observe != nil {
		w.observe(string(p))
	}
	return n, err
}

func (f *storageCLIFixture) stagingFlags(capacity int) []string {
	return []string{"--staging-dir", f.staging, "--max-staging-bytes", strconv.Itoa(capacity)}
}

func (f *storageCLIFixture) uploads() []storageCLIUpload {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.received)
}

func (f *storageCLIFixture) writeZIP(name string, entries map[string]string) string {
	f.t.Helper()
	filename := filepath.Join(f.dir, name)
	out, err := os.Create(filename)
	require.NoError(f.t, err)
	z := zip.NewWriter(out)
	_, err = z.Create("Takeout/")
	require.NoError(f.t, err)
	names := make([]string, 0, len(entries))
	for entry := range entries {
		names = append(names, entry)
	}
	slices.Sort(names)
	for _, entry := range names {
		part, err := z.Create(entry)
		require.NoError(f.t, err)
		_, err = io.WriteString(part, entries[entry])
		require.NoError(f.t, err)
	}
	require.NoError(f.t, z.Close())
	require.NoError(f.t, out.Close())
	return filename
}

func (f *storageCLIFixture) writeArchiveList(names ...string) string {
	f.t.Helper()
	filename := filepath.Join(f.dir, "archive-list.txt")
	// Deliberately preserve inventory order, spaces in names, blank lines and
	// CRLF. The CLI must trim lines without sorting or inventing ZIP numbers.
	contents := "\n  " + strings.Join(names, "  \r\n\r\n  ") + "  \r\n"
	require.NoError(f.t, os.WriteFile(filename, []byte(contents), 0o600))
	return filename
}

func (f *storageCLIFixture) payloadCopies(payload string, includePartial bool) int {
	f.t.Helper()
	copies := 0
	err := filepath.WalkDir(f.staging, func(filename string, entry fs.DirEntry, walkErr error) error {
		if errors.Is(walkErr, fs.ErrNotExist) && filename == f.staging {
			return nil
		}
		if walkErr != nil {
			return walkErr
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		data, err := os.ReadFile(filename)
		if err != nil {
			return err
		}
		if string(data) == payload || (includePartial && len(data) > 0 && bytes.HasPrefix([]byte(payload), data)) {
			copies++
		}
		return nil
	})
	require.NoError(f.t, err)
	return copies
}

func skipCLIStagingOnWindows(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("durable staging explicitly rejects Windows pending ACL and durable publication support")
	}
}

func nextStorageCLILine(t *testing.T, output string) string {
	t.Helper()
	var lines []string
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "Next:") {
			lines = append(lines, line)
		}
	}
	require.Len(t, lines, 1, "print one actionable next step through cmd.SetOut: %s", output)
	return lines[0]
}

func requireNextArchive(t *testing.T, output, filename string, replay bool) {
	t.Helper()
	line := nextStorageCLILine(t, output)
	require.Contains(t, line, strconv.Quote(filename), "the next source must use the exact recorded inventory filename")
	if replay {
		require.Regexp(t, `(?i)replay|re-present|second.pass|uncached|not cached|download.*again`, line, "distinguish an earlier uncached source from a first-pass download")
	} else {
		require.Regexp(t, `(?i)scan|first.pass|catalog|unseen|not.*processed`, line, "explain why an unscanned archive is next")
		action, _, _ := strings.Cut(line, strconv.Quote(filename))
		require.NotRegexp(t, `(?i)replay|re-present`, action, "do not request a second download before the catalog pass finishes")
	}
	_, reason, found := strings.Cut(line, strconv.Quote(filename)+".")
	require.True(t, found, "quote the exact filename before its reason: %s", line)
	require.NotEmpty(t, strings.TrimSpace(reason), "every requested archive needs a reason")
	require.NotRegexp(t, `https?://`, line)
}

func requireNoNextSource(t *testing.T, output string, knownNames ...string) {
	t.Helper()
	line := nextStorageCLILine(t, output)
	for _, name := range knownNames {
		require.NotContains(t, line, strconv.Quote(name), "a completed source must not be requested again")
	}
	require.NotRegexp(t, `"[^"\n]+\.zip"`, line)
	require.Regexp(t, `(?i)complete|no .*archive|no .*source|no .*download|no .*replay|nothing|done|all .*uploaded|all .*reconciled`, line, "explain that another source ZIP is unnecessary")
}
