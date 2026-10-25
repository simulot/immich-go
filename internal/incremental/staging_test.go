//go:build !windows

package incremental

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"hash/crc32"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/stretchr/testify/require"
)

func openStagingTest(filename string, scope Scope, client Client, options ...Option) (*Importer, error) {
	defaults := make([]Option, 0, 1+len(options))
	defaults = append(defaults, WithAvailableBytes(func(string) (uint64, error) { return 100_000_000_000, nil }))
	return Open(filename, scope, client, append(defaults, options...)...)
}

func stagedSession(t *testing.T, client Client, limit int64) (*Importer, string, string) {
	t.Helper()
	base := t.TempDir()
	state, directory := filepath.Join(base, "state.json"), filepath.Join(base, "cache")
	i, err := openStagingTest(state, scope(), client, WithStaging(directory, limit))
	require.NoError(t, err)
	t.Cleanup(func() { _ = i.Close() })
	return i, state, directory
}

func payloads(t *testing.T, directory string) []string {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(directory, "*.payload"))
	require.NoError(t, err)
	return paths
}

func TestStagingDeletedMediaZIPMetadataOnlyAfterRestart(t *testing.T) {
	ctx := context.Background()
	client := &fakeClient{}
	i, state, directory := stagedSession(t, client, 100)
	const body = "original media bytes"
	media := archive(t, map[string]string{mediaPath: body})
	before, err := os.ReadFile(media)
	require.NoError(t, err)
	require.NoError(t, i.ImportZIP(ctx, media))
	require.Zero(t, client.uploads)
	require.Len(t, payloads(t, directory), 1)
	require.EqualValues(t, len(body), i.Summary().StagedBytes)
	require.EqualValues(t, len(body), i.Summary().PendingStagedBytes)
	after, err := os.ReadFile(media)
	require.NoError(t, err)
	require.Equal(t, before, after, "source ZIP must remain unchanged")
	data, err := os.ReadFile(state)
	require.NoError(t, err)
	require.NotContains(t, string(data), body)
	require.NoError(t, i.Close())
	require.NoError(t, os.Remove(media))
	// Omitted options restore the directory and cap rather than stranding bytes.
	i, err = openStagingTest(state, scope(), client)
	require.NoError(t, err)
	defer i.Close()
	require.EqualValues(t, 100, i.Summary().MaxStagingBytes)
	canonical := "original-long-name.jpg"
	require.NoError(t, i.ImportZIP(ctx, archive(t, map[string]string{mediaPath + ".json": `{"title":"` + canonical + `","description":"original metadata","photoTakenTime":{"timestamp":"1577836800"}}`})))
	require.Equal(t, []string{canonical}, client.names)
	require.Equal(t, "original metadata", client.asset.ExifInfo.Description)
	require.Empty(t, payloads(t, directory))
	require.Zero(t, i.Summary().StagedBytes)
	require.Equal(t, 1, i.Summary().Uploaded)
	require.NoError(t, i.Reconcile(ctx))
	require.Equal(t, 1, client.uploads)
}

func TestStagingMetadataFirstAndPatchRetryNeedsNoPayload(t *testing.T) {
	ctx := context.Background()
	client := &fakeClient{failUpdate: true}
	i, state, directory := stagedSession(t, client, 10)
	require.NoError(t, i.ImportZIP(ctx, archive(t, map[string]string{mediaPath + ".json": sidecar})))
	require.ErrorContains(t, i.ImportZIP(ctx, archive(t, map[string]string{mediaPath: "photo"})), "pending metadata")
	require.Equal(t, []string{"photo.jpg"}, client.names)
	require.Empty(t, payloads(t, directory))
	require.NoError(t, i.Close())
	client.failUpdate = false
	i, err := openStagingTest(state, scope(), client)
	require.NoError(t, err)
	defer i.Close()
	require.NoError(t, i.Reconcile(ctx))
	require.Equal(t, 1, client.uploads)
	require.Equal(t, 1, i.Summary().Enriched)
}

func TestStagingDuplicatePathsOnePayloadAndLateDuplicateMetadata(t *testing.T) {
	ctx := context.Background()
	client := &fakeClient{}
	i, state, directory := stagedSession(t, client, 5)
	const duplicate = "Takeout/Album/short(1).jpg"
	media := archive(t, map[string]string{mediaPath: "photo", duplicate: "photo"})
	require.NoError(t, i.ImportZIP(ctx, media))
	require.Len(t, payloads(t, directory), 1)
	require.EqualValues(t, 5, i.Summary().StagedBytes)
	require.Len(t, i.state.Occurrences, 2)
	require.NoError(t, i.ImportZIP(ctx, media), "repeat occurrences must not use another payload budget")
	require.NoError(t, os.Remove(media))
	require.NoError(t, i.Close())
	i, err := openStagingTest(state, scope(), client)
	require.NoError(t, err)
	defer i.Close()
	require.NoError(t, i.ImportZIP(ctx, archive(t, map[string]string{duplicate + ".json": sidecar})))
	require.Equal(t, []string{"photo.jpg"}, client.names)
	require.Equal(t, 1, i.Summary().Enriched)
	require.Empty(t, payloads(t, directory))
}

func TestStagingCapSelectsPersistentFallbackAndPreservesOldData(t *testing.T) {
	ctx := context.Background()
	i, state, directory := stagedSession(t, &fakeClient{}, 5)
	require.NoError(t, i.ImportZIP(ctx, archive(t, map[string]string{mediaPath: "first"})))
	old := payloads(t, directory)[0]
	second := archive(t, map[string]string{"Takeout/second.jpg": "second"})
	require.NoError(t, i.ImportZIP(ctx, second))
	require.True(t, i.Summary().Fallback)
	require.Contains(t, i.StorageStatus().Reason, "staging capacity exhausted")
	require.Contains(t, i.StorageStatus().Reason, "retained 5 bytes")
	require.Contains(t, i.StorageStatus().Reason, "cap 5 bytes")
	data, err := os.ReadFile(old)
	require.NoError(t, err)
	require.Equal(t, "first", string(data))
	require.Len(t, payloads(t, directory), 1)
	require.FileExists(t, second)
	require.NoError(t, i.Close())
	i, err = openStagingTest(state, scope(), &fakeClient{}, WithStaging(directory, 11))
	require.NoError(t, err)
	defer i.Close()
	require.NoError(t, i.ImportZIP(ctx, second))
	require.True(t, i.Summary().Fallback, "increasing the cap does not automatically reenable staging")
	require.Len(t, payloads(t, directory), 1)
	require.EqualValues(t, 5, i.Summary().StagedBytes)
}

func TestStagingDrainsBeforeNewPayloadConsumesCapacity(t *testing.T) {
	ctx := context.Background()
	client := &fakeClient{}
	i, _, directory := stagedSession(t, client, 5)
	require.NoError(t, i.ImportZIP(ctx, archive(t, map[string]string{mediaPath: "first"})))
	require.NoError(t, i.ImportZIP(ctx, archive(t, map[string]string{
		mediaPath + ".json": sidecar, "Takeout/next.jpg": "later",
	})))
	require.Equal(t, 1, client.uploads)
	require.Len(t, payloads(t, directory), 1)
	require.EqualValues(t, 5, i.Summary().PendingStagedBytes)
}

func TestStagingLostUploadAckRetainsBytesAndReconcilesWithoutZIP(t *testing.T) {
	for _, alreadyStaged := range []bool{false, true} {
		t.Run(map[bool]string{false: "direct upload", true: "cached upload"}[alreadyStaged], func(t *testing.T) {
			ctx := context.Background()
			client := &fakeClient{loseReceipt: true}
			i, state, directory := stagedSession(t, client, 100)
			files := map[string]string{mediaPath: "photo"}
			if !alreadyStaged {
				files[mediaPath+".json"] = sidecar
			}
			media := archive(t, files)
			err := i.ImportZIP(ctx, media)
			if alreadyStaged {
				require.NoError(t, err)
				err = i.ImportZIP(ctx, archive(t, map[string]string{mediaPath + ".json": sidecar}))
			}
			require.ErrorContains(t, err, "lost upload acknowledgement")
			require.Len(t, payloads(t, directory), 1)
			require.Equal(t, 1, client.uploads, "must not immediately retry an uncertain upload in the same call")
			require.NoError(t, os.Remove(media))
			require.NoError(t, i.Close())
			i, err = openStagingTest(state, scope(), client)
			require.NoError(t, err)
			defer i.Close()
			require.NoError(t, i.Reconcile(ctx))
			require.Equal(t, 2, client.uploads)
			require.Zero(t, client.updates)
			require.Equal(t, 1, i.Summary().Unowned)
			require.Empty(t, payloads(t, directory))
		})
	}
}

func TestStagingMissingAndCorruptPayloadFailLoudAndCanBeRepaired(t *testing.T) {
	for _, damage := range []string{"missing", "size", "checksum"} {
		t.Run(damage, func(t *testing.T) {
			ctx := context.Background()
			client := &fakeClient{}
			i, state, directory := stagedSession(t, client, 100)
			media := archive(t, map[string]string{mediaPath: "photo"})
			require.NoError(t, i.ImportZIP(ctx, media))
			payload := payloads(t, directory)[0]
			require.NoError(t, i.Close())
			switch damage {
			case "missing":
				require.NoError(t, os.Remove(payload))
			case "size":
				require.NoError(t, os.WriteFile(payload, []byte("shortened"), 0o600))
			case "checksum":
				require.NoError(t, os.WriteFile(payload, []byte("other"), 0o600))
			}
			i, err := openStagingTest(state, scope(), client)
			require.NoError(t, err)
			defer i.Close()
			require.ErrorContains(t, i.Reconcile(ctx), "staged payload")
			require.Zero(t, client.uploads)
			// First repair still reports the cache failure found before the repair.
			require.ErrorContains(t, i.ImportZIP(ctx, media), "staged payload")
			require.NoError(t, i.ImportZIP(ctx, archive(t, map[string]string{mediaPath + ".json": sidecar})))
			require.Equal(t, 1, client.uploads)
			require.Empty(t, payloads(t, directory))
		})
	}
}

func TestStagingSymlinkAndUnownedFilesAreNeverDeleted(t *testing.T) {
	for _, badEntry := range []string{"symlink payload", "foreign file"} {
		t.Run(badEntry, func(t *testing.T) {
			i, state, directory := stagedSession(t, &fakeClient{}, 100)
			require.NoError(t, i.ImportZIP(context.Background(), archive(t, map[string]string{mediaPath: "photo"})))
			payload := payloads(t, directory)[0]
			require.NoError(t, i.Close())
			outside := filepath.Join(t.TempDir(), "precious.jpg")
			require.NoError(t, os.WriteFile(outside, []byte("user file"), 0o600))
			if badEntry == "symlink payload" {
				require.NoError(t, os.Remove(payload))
				require.NoError(t, os.Symlink(outside, payload))
			} else {
				require.NoError(t, os.WriteFile(filepath.Join(directory, "unowned.zip"), []byte("user file"), 0o600))
			}
			_, err := openStagingTest(state, scope(), &fakeClient{})
			require.Error(t, err)
			data, err := os.ReadFile(outside)
			require.NoError(t, err)
			require.Equal(t, "user file", string(data))
			if badEntry == "foreign file" {
				require.FileExists(t, filepath.Join(directory, "unowned.zip"))
			}
		})
	}
}

func TestStagingBindingRejectsOtherStateMovedStateAndChangedDirectory(t *testing.T) {
	i, state, directory := stagedSession(t, &fakeClient{}, 100)
	_, err := openStagingTest(filepath.Join(t.TempDir(), "different.json"), scope(), &fakeClient{}, WithStaging(directory, 100))
	require.ErrorContains(t, err, "staging is locked")
	require.NoError(t, i.Close())
	data, err := os.ReadFile(state)
	require.NoError(t, err)
	otherState := filepath.Join(t.TempDir(), "copied.json")
	require.NoError(t, os.WriteFile(otherState, data, 0o600))
	_, err = openStagingTest(otherState, scope(), &fakeClient{})
	require.ErrorContains(t, err, "copied/moved state")
	_, err = openStagingTest(filepath.Join(t.TempDir(), "new.json"), scope(), &fakeClient{}, WithStaging(directory, 100))
	require.ErrorContains(t, err, "must be empty")
	_, err = openStagingTest(state, scope(), &fakeClient{}, WithStaging(filepath.Join(t.TempDir(), "new-cache"), 100))
	require.ErrorContains(t, err, "differs from the persisted binding")
	i, err = openStagingTest(state, scope(), &fakeClient{})
	require.NoError(t, err)
	require.NoError(t, i.Close())
}

func TestStagingRefusesDirectorySymlinkAndInvalidOptions(t *testing.T) {
	for _, option := range []Option{WithStaging("", 1), WithStaging("cache", 0), WithStaging("cache", -1)} {
		_, err := openStagingTest(filepath.Join(t.TempDir(), "state.json"), scope(), &fakeClient{}, option)
		require.ErrorContains(t, err, "positive")
	}
	base := t.TempDir()
	realDirectory := filepath.Join(base, "real")
	require.NoError(t, os.Mkdir(realDirectory, 0o700))
	link := filepath.Join(base, "link")
	require.NoError(t, os.Symlink(realDirectory, link))
	_, err := openStagingTest(filepath.Join(base, "state.json"), scope(), &fakeClient{}, WithStaging(link, 10))
	require.ErrorContains(t, err, "symlink")
	entries, err := os.ReadDir(realDirectory)
	require.NoError(t, err)
	require.Empty(t, entries)
}

func TestStagingRecoveryPublishReferenceAndReceiptDeletionCrashBoundaries(t *testing.T) {
	ctx := context.Background()
	client := &fakeClient{}
	i, state, directory := stagedSession(t, client, 100)
	require.NoError(t, i.ImportZIP(ctx, archive(t, map[string]string{mediaPath: "photo"})))
	checksum := i.state.Occurrences[mediaPath].Checksum
	m := i.state.Media[checksum]
	m.Staged = false // Crash after rename, before publishing the state reference.
	i.state.Media[checksum] = m
	require.NoError(t, i.save())
	temp := filepath.Join(directory, ".tmp-"+i.state.Staging.ID+"-interrupted")
	require.NoError(t, os.WriteFile(temp, []byte("partial"), 0o600))
	require.NoError(t, i.Close())
	i, err := openStagingTest(state, scope(), client)
	require.NoError(t, err)
	require.True(t, i.state.Media[checksum].Staged)
	require.NoFileExists(t, temp)
	require.EqualValues(t, 5, i.Summary().StagedBytes)
	// Simulate a durable remote receipt preceding cleanup, with or without the
	// payload deletion having reached disk. Neither path may upload again.
	m = i.state.Media[checksum]
	m.CanonicalName, m.Attempted, m.AssetID = "photo.jpg", true, "asset-1"
	i.state.Media[checksum] = m
	require.NoError(t, i.save())
	require.NoError(t, i.Close())
	i, err = openStagingTest(state, scope(), client)
	require.NoError(t, err)
	require.NoError(t, i.Reconcile(ctx))
	require.Empty(t, payloads(t, directory))
	m.Staged = true
	i.state.Media[checksum] = m
	require.NoError(t, i.save())
	require.NoError(t, i.Close())
	i, err = openStagingTest(state, scope(), client)
	require.NoError(t, err)
	defer i.Close()
	require.NoError(t, i.Reconcile(ctx))
	require.False(t, i.state.Media[checksum].Staged)
	require.Zero(t, client.uploads)
}

type stateFailureClient struct {
	*fakeClient
	afterUpload    func()
	beforeBaseline func()
}

func (c *stateFailureClient) AssetUpload(ctx context.Context, asset *assets.Asset) (immich.AssetResponse, error) {
	result, err := c.fakeClient.AssetUpload(ctx, asset)
	if c.afterUpload != nil {
		c.afterUpload()
	}
	return result, err
}

func (c *stateFailureClient) GetAssetInfo(ctx context.Context, id string) (*immich.Asset, error) {
	if c.beforeBaseline != nil {
		c.beforeBaseline()
	}
	return c.fakeClient.GetAssetInfo(ctx, id)
}

func TestStagingFailedReceiptSaveCannotReleaseCacheInSameSession(t *testing.T) {
	for _, failure := range []string{"receipt", "baseline"} {
		t.Run(failure, func(t *testing.T) {
			ctx := context.Background()
			client := &stateFailureClient{fakeClient: &fakeClient{}}
			i, state, directory := stagedSession(t, client, 100)
			require.NoError(t, i.ImportZIP(ctx, archive(t, map[string]string{mediaPath: "photo"})))
			checksum := i.state.Occurrences[mediaPath].Checksum
			breakReplacement := func() {
				require.NoError(t, os.Rename(state, state+".preserved"))
				require.NoError(t, os.Mkdir(state, 0o700))
				client.afterUpload, client.beforeBaseline = nil, nil
			}
			if failure == "receipt" {
				client.afterUpload = breakReplacement
			} else {
				client.beforeBaseline = breakReplacement
			}
			require.Error(t, i.ImportZIP(ctx, archive(t, map[string]string{mediaPath + ".json": sidecar})))
			require.Len(t, payloads(t, directory), 1)
			data, err := os.ReadFile(state + ".preserved")
			require.NoError(t, err)
			var durable State
			require.NoError(t, json.Unmarshal(data, &durable))
			require.NoError(t, os.Remove(state))
			require.NoError(t, os.Rename(state+".preserved", state))
			if failure == "receipt" {
				require.Empty(t, i.state.Media[checksum].AssetID, "in-memory receipt must not authorize deletion after its save failed")
				require.Empty(t, durable.Media[checksum].AssetID)
			} else {
				require.Equal(t, "asset-1", i.state.Media[checksum].AssetID)
				require.Equal(t, "asset-1", durable.Media[checksum].AssetID, "first receipt succeeded before baseline save failed")
			}
			require.NoError(t, i.Reconcile(ctx))
			require.Empty(t, payloads(t, directory))
			if failure == "receipt" {
				require.Equal(t, 2, client.uploads, "only a fresh durable receipt may release cached bytes")
				require.False(t, i.state.Media[checksum].Owned)
				require.Zero(t, client.updates)
			} else {
				require.Equal(t, 1, client.uploads, "durable receipt needs no second upload")
			}
		})
	}
}

func TestStagingV2MigrationPreservesCanonicalNames(t *testing.T) {
	ctx := context.Background()
	base := t.TempDir()
	state := filepath.Join(base, "state.json")
	client := &fakeClient{}
	i, err := openStagingTest(state, scope(), client)
	require.NoError(t, err)
	require.NoError(t, i.ImportZIP(ctx, archive(t, map[string]string{mediaPath: "photo", mediaPath + ".json": sidecar})))
	i.state.Version = 2
	require.NoError(t, i.save())
	require.NoError(t, i.Close())
	i, err = openStagingTest(state, scope(), client, WithStaging(filepath.Join(base, "cache"), 100))
	require.NoError(t, err)
	defer i.Close()
	require.Equal(t, stateVersion, i.state.Version)
	require.Equal(t, "photo.jpg", i.state.Media[i.state.Occurrences[mediaPath].Checksum].CanonicalName)
	require.NoError(t, i.Reconcile(ctx))
	require.Equal(t, 1, client.uploads)
	data, err := os.ReadFile(state)
	require.NoError(t, err)
	var persisted State
	require.NoError(t, json.Unmarshal(data, &persisted))
	require.Equal(t, stateVersion, persisted.Version)
}

func TestStagingInvalidAdvertisedSizeRejectedBeforeBudgetAndHealthyItemProceeds(t *testing.T) {
	ctx := context.Background()
	i, _, directory := stagedSession(t, &fakeClient{}, 5)
	filename := filepath.Join(t.TempDir(), "bad-size.zip")
	f, err := os.Create(filename)
	require.NoError(t, err)
	z := zip.NewWriter(f)
	contents := []byte("actual bytes")
	w, err := z.CreateRaw(&zip.FileHeader{Name: "Takeout/bad.jpg", Method: zip.Store, CRC32: crc32.ChecksumIEEE(contents), CompressedSize64: uint64(len(contents)), UncompressedSize64: 1 << 40})
	require.NoError(t, err)
	_, err = w.Write(contents)
	require.NoError(t, err)
	w, err = z.Create("Takeout/good.jpg")
	require.NoError(t, err)
	_, err = w.Write([]byte("photo"))
	require.NoError(t, err)
	require.NoError(t, z.Close())
	require.NoError(t, f.Close())
	err = i.ImportZIP(ctx, filename)
	require.Error(t, err)
	require.NotContains(t, err.Error(), "staging capacity exhausted")
	require.Len(t, payloads(t, directory), 1)
	require.EqualValues(t, 5, i.Summary().StagedBytes)
	data, err := os.ReadFile(payloads(t, directory)[0])
	require.NoError(t, err)
	require.True(t, bytes.Equal([]byte("photo"), data))
}

func TestStagingSkippedSupportedMediaRequiresKeepingZIPAndPreservesHealthyProgress(t *testing.T) {
	for _, scenario := range []struct {
		name, source, reason      string
		duplicate, original, size bool
	}{
		{name: "duplicate entries", source: mediaPath, duplicate: true, reason: "duplicate ZIP entry path"},
		{name: "different original bytes", source: mediaPath, original: true, reason: "ambiguous original path has different media"},
		{name: "different recorded size", source: mediaPath, size: true, reason: "checksum matches media with a different size"},
		{name: "unsafe parent path", source: "../escape.jpg", reason: "unsafe archive path"},
		{name: "unsafe absolute path", source: "/escape.jpg", reason: "unsafe archive path"},
		{name: "unsafe backslash path", source: `Takeout\escape.jpg`, reason: "unsafe archive path"},
	} {
		for _, staging := range []bool{false, true} {
			mode := "catalog only"
			if staging {
				mode = "staged"
			}
			t.Run(scenario.name+"/"+mode, func(t *testing.T) {
				ctx := context.Background()
				client := &fakeClient{}
				base := t.TempDir()
				state, directory := filepath.Join(base, "state.json"), filepath.Join(base, "cache")
				var options []Option
				if staging {
					options = append(options, WithStaging(directory, 100))
				}
				i, err := openStagingTest(state, scope(), client, options...)
				require.NoError(t, err)
				defer func() { require.NoError(t, i.Close()) }()
				const (
					rejectedBody = "skipped bytes"
					previousBody = "previous bytes"
					healthyBody  = "healthy cached bytes"
				)
				if scenario.original {
					require.NoError(t, i.ImportZIP(ctx, archive(t, map[string]string{scenario.source: previousBody})))
				}
				if scenario.size {
					// Model a persisted logical record that contradicts this otherwise
					// valid ZIP, independently of ZIP CRC/advertised-size validation.
					digest := sha1.Sum([]byte(rejectedBody))
					checksum := base64.StdEncoding.EncodeToString(digest[:])
					i.state.Media[checksum] = Media{Checksum: checksum, Size: uint64(len(rejectedBody) + 1)}
				}
				filename := filepath.Join(t.TempDir(), "source.zip")
				f, err := os.Create(filename)
				require.NoError(t, err)
				z := zip.NewWriter(f)
				entries := []struct{ name, body string }{
					{scenario.source, rejectedBody},
					{"Takeout/healthy.jpg", healthyBody},
					{"Takeout/ready.jpg", "ready upload bytes"},
					{"Takeout/ready.jpg.json", `{"title":"ready.jpg"}`},
				}
				if scenario.duplicate {
					entries = append(entries, struct{ name, body string }{scenario.source, "other skipped bytes"})
				}
				for _, entry := range entries {
					w, err := z.Create(entry.name)
					require.NoError(t, err)
					_, err = w.Write([]byte(entry.body))
					require.NoError(t, err)
				}
				require.NoError(t, z.Close())
				require.NoError(t, f.Close())
				before, err := os.ReadFile(filename)
				require.NoError(t, err)
				var previous Summary
				for attempt := range 3 {
					err := i.ImportZIP(ctx, filename)
					if staging {
						require.ErrorContains(t, err, "keep this ZIP")
						require.ErrorContains(t, err, scenario.reason)
						require.ErrorContains(t, err, filename)
						stagedPaths := payloads(t, directory)
						cached := make([]string, 0, len(stagedPaths))
						for _, payload := range stagedPaths {
							data, err := os.ReadFile(payload)
							require.NoError(t, err)
							cached = append(cached, string(data))
						}
						want := []string{healthyBody}
						if scenario.original {
							want = append(want, previousBody)
						}
						require.ElementsMatch(t, want, cached)
					} else {
						require.NoError(t, err, "catalog-only item deferral remains unchanged")
					}
					require.Equal(t, 1, client.uploads, "healthy receipt must survive retries")
					require.Equal(t, []string{"ready.jpg"}, client.names)
					require.NotEmpty(t, i.state.Problems[scenario.source])
					require.Contains(t, i.state.Problems[scenario.source][0], scenario.reason)
					if attempt > 0 {
						require.Equal(t, previous, i.Summary(), "retries must preserve logical inventory and retained-byte counts")
					}
					previous = i.Summary()
					if attempt == 0 {
						require.NoError(t, i.Close())
						i, err = openStagingTest(state, scope(), client)
						require.NoError(t, err)
					}
				}
				after, err := os.ReadFile(filename)
				require.NoError(t, err)
				require.Equal(t, before, after, "source ZIP must remain unchanged")
			})
		}
	}
}

func TestStagingAggregatesUnretainedSupportedSources(t *testing.T) {
	i, _, directory := stagedSession(t, &fakeClient{}, 100)
	filename := archive(t, map[string]string{
		"../first.jpg":        "first unsupported path",
		`Takeout\second.jpg`:  "second unsupported path",
		"Takeout/healthy.jpg": "healthy bytes",
		"Takeout/ignored.txt": "unsupported extension remains out of scope",
	})
	err := i.ImportZIP(context.Background(), filename)
	require.ErrorContains(t, err, "keep this ZIP")
	require.ErrorContains(t, err, "../first.jpg")
	require.ErrorContains(t, err, `Takeout\\second.jpg`)
	require.NotContains(t, err.Error(), "ignored.txt")
	require.Len(t, payloads(t, directory), 1)
	require.FileExists(t, filename)
}

type stagingCleanupFailureContext struct {
	context.Context
	directory string
	fired     bool
}

func (c *stagingCleanupFailureContext) Err() error {
	if c.fired {
		return context.Canceled
	}
	entries, err := os.ReadDir(c.directory)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), ".tmp-") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Size() > 0 {
			if err := os.Chmod(c.directory, 0o500); err != nil {
				return err
			}
			c.fired = true
			return context.Canceled
		}
	}
	return c.Context.Err()
}

func actualStagingBytes(t *testing.T, directory string) int64 {
	t.Helper()
	entries, err := os.ReadDir(directory)
	require.NoError(t, err)
	var total int64
	for _, entry := range entries {
		if entry.Name() == stagingOwner || entry.Name() == stagingLock {
			continue
		}
		info, err := entry.Info()
		require.NoError(t, err)
		total += info.Size()
	}
	return total
}

func TestStagingFailedTemporaryCleanupRemainsChargedAndCanRecover(t *testing.T) {
	for _, restart := range []bool{false, true} {
		name := "same session"
		if restart {
			name = "restart"
		}
		t.Run(name, func(t *testing.T) {
			const (
				completeBody = "already retained"
				pendingSize  = 128 << 10
				limit        = int64(len(completeBody)) + pendingSize
			)
			i, state, directory := stagedSession(t, &fakeClient{}, limit)
			t.Cleanup(func() { _ = os.Chmod(directory, 0o700) })
			// Some test environments run as root and cannot reproduce unlink
			// denial with directory permissions. Verify this prerequisite first.
			probe := filepath.Join(directory, "permission-probe")
			require.NoError(t, os.WriteFile(probe, []byte("probe"), 0o600))
			require.NoError(t, os.Chmod(directory, 0o500))
			removeErr := os.Remove(probe)
			require.NoError(t, os.Chmod(directory, 0o700))
			if removeErr == nil {
				t.Skip("filesystem privileges bypass directory unlink restrictions")
			}
			require.ErrorIs(t, removeErr, os.ErrPermission)
			require.NoError(t, os.Remove(probe))
			require.NoError(t, i.ImportZIP(context.Background(), archive(t, map[string]string{"Takeout/old.jpg": completeBody})))
			complete := payloads(t, directory)[0]
			filename := archive(t, map[string]string{mediaPath: strings.Repeat("x", pendingSize)})
			ctx := &stagingCleanupFailureContext{Context: context.Background(), directory: directory}
			err := i.ImportZIP(ctx, filename)
			require.ErrorIs(t, err, context.Canceled)
			require.True(t, ctx.fired, "cancellation must follow an actual partial write")
			require.ErrorContains(t, err, "cleanup unfinished staging file")
			require.ErrorIs(t, err, os.ErrPermission)
			actual := actualStagingBytes(t, directory)
			require.Greater(t, actual, int64(len(completeBody)))
			require.Equal(t, actual, i.Summary().StagedBytes, "unfinished retained bytes must stay charged")
			require.EqualValues(t, len(completeBody), i.Summary().PendingStagedBytes)
			// An unsuccessful recovery cannot create another staging copy.
			require.ErrorContains(t, i.ImportZIP(context.Background(), filename), "cleanup unfinished staging file")
			require.Equal(t, actual, actualStagingBytes(t, directory))
			require.Equal(t, actual, i.Summary().StagedBytes)
			require.NoError(t, os.Chmod(directory, 0o700))
			if restart {
				require.NoError(t, i.Close())
				i, err = openStagingTest(state, scope(), &fakeClient{})
				require.NoError(t, err)
				defer i.Close()
				require.EqualValues(t, len(completeBody), i.Summary().StagedBytes)
			}
			require.NoError(t, i.ImportZIP(context.Background(), filename))
			temps, err := filepath.Glob(filepath.Join(directory, ".tmp-*"))
			require.NoError(t, err)
			require.Empty(t, temps)
			require.Len(t, payloads(t, directory), 2)
			require.EqualValues(t, limit, actualStagingBytes(t, directory))
			require.EqualValues(t, limit, i.Summary().StagedBytes)
			require.EqualValues(t, limit, i.Summary().PendingStagedBytes)
			data, err := os.ReadFile(complete)
			require.NoError(t, err)
			require.Equal(t, completeBody, string(data), "complete payloads must not be evicted")
			require.FileExists(t, filename)
		})
	}
}

func TestStagingCapIsIncreaseOnlyAndRejectedChangesPreserveState(t *testing.T) {
	i, state, directory := stagedSession(t, &fakeClient{}, 100)
	require.NoError(t, i.ImportZIP(context.Background(), archive(t, map[string]string{mediaPath: "photo"})))
	payload := payloads(t, directory)[0]
	require.NoError(t, i.Close())
	persistedLimit := int64(100)
	for _, limit := range []int64{1, 5, 99, 100, 101, 100} {
		before, err := os.ReadFile(state)
		require.NoError(t, err)
		reopened, err := openStagingTest(state, scope(), &fakeClient{}, WithStaging(directory, limit))
		if limit < persistedLimit {
			require.ErrorContains(t, err, "cannot decrease")
			require.Nil(t, reopened)
			after, err := os.ReadFile(state)
			require.NoError(t, err)
			require.Equal(t, before, after, "a rejected cap must not change durable configuration")
		} else {
			require.NoError(t, err)
			require.Equal(t, limit, reopened.Summary().MaxStagingBytes)
			require.EqualValues(t, 5, reopened.Summary().StagedBytes)
			require.NoError(t, reopened.Close())
			persistedLimit = limit
		}
		data, err := os.ReadFile(payload)
		require.NoError(t, err)
		require.Equal(t, "photo", string(data))
	}
}

func TestStagingRejectsPersistedCapBelowRetainedUsage(t *testing.T) {
	i, state, directory := stagedSession(t, &fakeClient{}, 5)
	require.NoError(t, i.ImportZIP(context.Background(), archive(t, map[string]string{mediaPath: "photo"})))
	i.state.Staging.MaxBytes = 1 // Model a state written before the cap-decrease fix.
	require.NoError(t, i.save())
	require.NoError(t, i.Close())
	for _, limit := range []int64{0, 2, 5} {
		before, err := os.ReadFile(state)
		require.NoError(t, err)
		var options []Option
		if limit > 0 {
			options = append(options, WithStaging(directory, limit))
		}
		reopened, err := openStagingTest(state, scope(), &fakeClient{}, options...)
		if limit < 5 {
			require.ErrorContains(t, err, "retained 5 bytes exceed staging cap")
			require.Nil(t, reopened)
			after, err := os.ReadFile(state)
			require.NoError(t, err)
			require.Equal(t, before, after)
		} else {
			require.NoError(t, err)
			require.NoError(t, reopened.Close())
		}
		require.EqualValues(t, 5, actualStagingBytes(t, directory))
	}
}
