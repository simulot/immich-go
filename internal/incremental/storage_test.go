//go:build !windows

package incremental

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/stretchr/testify/require"
)

func TestStorageInitialThresholdsAndBoundedNotifications(t *testing.T) {
	for _, scenario := range []struct {
		name, level string
		free        uint64
		fallback    bool
	}{
		{"above warning", "ok", storageWarning + 1, false},
		{"warning inclusive", "warning", storageWarning, false},
		{"above danger", "warning", storageDanger + 1, false},
		{"danger inclusive", "danger", storageDanger, false},
		{"reserve inclusive", "danger", storageReserve, true},
		{"below reserve", "danger", storageReserve - 1, true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			base := t.TempDir()
			state, directory := filepath.Join(base, "state.json"), filepath.Join(base, "cache")
			expectedDirectory, err := canonicalLocation(directory)
			require.NoError(t, err)
			events := make([]StorageStatus, 0, 3)
			options := []Option{
				WithStaging(directory, 1<<20),
				WithAvailableBytes(func(probed string) (uint64, error) {
					require.Equal(t, expectedDirectory, probed)
					return scenario.free, nil
				}),
				WithStorageObserver(func(status StorageStatus) { events = append(events, status) }),
			}
			i, err := openStagingTest(state, scope(), &fakeClient{}, options...)
			require.NoError(t, err)
			require.NotEmpty(t, events, "Open emits before any ZIP arguments or reconciliation")
			require.Equal(t, scenario.level, events[0].Level)
			require.Equal(t, scenario.free, events[0].FreeBytes)
			require.Equal(t, scenario.fallback, i.StorageStatus().Fallback)
			count := len(events)
			for range 20 {
				_, err := i.observeStorage()
				require.NoError(t, err)
			}
			require.Len(t, events, count, "chunk checks cannot repeat warning notifications")
			require.NoError(t, i.Close())
			events = events[:0]
			i, err = openStagingTest(state, scope(), &fakeClient{}, options...)
			require.NoError(t, err)
			defer i.Close()
			require.NotEmpty(t, events, "every restart emits its initial status")
			require.Equal(t, scenario.fallback, events[0].Fallback)
		})
	}
}

func TestStorageWholePayloadReserveAndStickyFallback(t *testing.T) {
	for _, enough := range []bool{false, true} {
		name := "projected below reserve"
		if enough {
			name = "projected exactly reserve"
		}
		t.Run(name, func(t *testing.T) {
			const body = "photo bytes"
			free := storageReserve + uint64(len(body))
			if !enough {
				free--
			}
			base := t.TempDir()
			state, directory := filepath.Join(base, "state.json"), filepath.Join(base, "cache")
			client := &fakeClient{}
			i, err := openStagingTest(state, scope(), client, WithStaging(directory, 100), WithAvailableBytes(func(string) (uint64, error) { return free, nil }))
			require.NoError(t, err)
			filename := archive(t, map[string]string{mediaPath: body})
			require.NoError(t, i.ImportZIP(context.Background(), filename))
			require.Zero(t, client.uploads)
			require.Equal(t, !enough, i.Summary().Fallback)
			if enough {
				require.EqualValues(t, len(body), i.Summary().StagedBytes)
			} else {
				require.Empty(t, payloads(t, directory))
				require.False(t, i.state.Media[i.state.Occurrences[mediaPath].Checksum].Attempted)
			}
			require.NoError(t, i.Close())
			i, err = openStagingTest(state, scope(), client)
			require.NoError(t, err)
			defer i.Close()
			require.Equal(t, !enough, i.StorageStatus().Fallback, "more free storage must not automatically reenable staging")
			require.NoError(t, i.ImportZIP(context.Background(), archive(t, map[string]string{mediaPath + ".json": sidecar})))
			if !enough {
				require.Zero(t, client.uploads, "fallback never fabricates a cached payload or receipt")
				require.NoError(t, i.ImportZIP(context.Background(), filename), "a second pass can upload named media with guarded workspace")
			}
			require.Equal(t, 1, client.uploads)
			require.Equal(t, []string{"photo.jpg"}, client.names)
			require.Empty(t, payloads(t, directory))
		})
	}
}

func TestStoragePerChunkConsumptionStopsPendingCopyAndKeepsInventory(t *testing.T) {
	base := t.TempDir()
	state, directory := filepath.Join(base, "state.json"), filepath.Join(base, "cache")
	events := make([]StorageStatus, 0, 4)
	i, err := openStagingTest(state, scope(), &fakeClient{}, WithStaging(directory, 1<<20),
		WithAvailableBytes(func(string) (uint64, error) {
			temps, err := filepath.Glob(filepath.Join(directory, ".tmp-*"))
			require.NoError(t, err)
			for _, name := range temps {
				info, err := os.Stat(name)
				require.NoError(t, err)
				if info.Size() > 0 {
					return storageReserve - 1, nil
				}
			}
			return 100_000_000_000, nil
		}), WithStorageObserver(func(status StorageStatus) { events = append(events, status) }))
	require.NoError(t, err)
	defer i.Close()
	filename := archive(t, map[string]string{mediaPath: strings.Repeat("x", 256<<10)})
	require.NoError(t, i.ImportZIP(context.Background(), filename))
	require.True(t, i.Summary().Fallback)
	require.Zero(t, i.Summary().StagedBytes)
	require.Zero(t, actualStagingBytes(t, directory))
	require.Len(t, i.state.Occurrences, 1)
	require.Len(t, events, 3, "initial ok, danger crossing and persisted fallback are emitted immediately")
	require.True(t, events[2].Fallback)
	require.Contains(t, events[2].Reason, "10 GB reserve")
	require.FileExists(t, filename)
}

func TestStorageProbeFailuresStayErrorsWithoutFallback(t *testing.T) {
	probeErr := errors.New("filesystem measurement unavailable")
	base := t.TempDir()
	state, directory := filepath.Join(base, "state.json"), filepath.Join(base, "cache")
	_, err := openStagingTest(state, scope(), &fakeClient{}, WithStaging(directory, 1<<20), WithAvailableBytes(func(string) (uint64, error) { return 0, probeErr }))
	require.ErrorIs(t, err, probeErr)
	i, err := openStagingTest(state, scope(), &fakeClient{}, WithAvailableBytes(func(string) (uint64, error) {
		temps, err := filepath.Glob(filepath.Join(directory, ".tmp-*"))
		require.NoError(t, err)
		for _, name := range temps {
			info, err := os.Stat(name)
			require.NoError(t, err)
			if info.Size() > 0 {
				return 0, probeErr
			}
		}
		return 100_000_000_000, nil
	}))
	require.NoError(t, err)
	defer i.Close()
	err = i.ImportZIP(context.Background(), archive(t, map[string]string{mediaPath: strings.Repeat("x", 256<<10)}))
	require.ErrorIs(t, err, probeErr)
	require.False(t, i.Summary().Fallback)
	require.Zero(t, actualStagingBytes(t, directory))
	require.Len(t, i.state.Occurrences, 1)
}

type regularFileUploadClient struct {
	*fakeClient
	t *testing.T
}

func (c *regularFileUploadClient) AssetUpload(ctx context.Context, asset *assets.Asset) (immich.AssetResponse, error) {
	f, err := asset.File.Open()
	require.NoError(c.t, err)
	_, regular := f.(*os.File)
	require.True(c.t, regular, "shared uploader must receive an actual file and bypass its own full copy")
	require.NoError(c.t, f.Close())
	opened, err := asset.OpenFile()
	require.NoError(c.t, err)
	_, err = io.Copy(io.Discard, opened)
	require.NoError(c.t, err)
	require.NoError(c.t, opened.Close())
	return c.fakeClient.AssetUpload(ctx, asset)
}

func TestStorageGuardedUploadWorkspaceAndCachedDrainBelowReserve(t *testing.T) {
	sharedTemp := t.TempDir()
	t.Setenv("IMMICHGO_TEMPDIR", sharedTemp)
	client := &regularFileUploadClient{fakeClient: &fakeClient{}, t: t}
	i, state, directory := stagedSession(t, client, 100)
	require.NoError(t, i.ImportZIP(context.Background(), archive(t, map[string]string{mediaPath: "photo", mediaPath + ".json": sidecar})))
	require.Equal(t, 1, client.uploads)
	require.Zero(t, actualStagingBytes(t, directory))
	entries, err := os.ReadDir(sharedTemp)
	require.NoError(t, err)
	require.Empty(t, entries, "the shared uploader must not create another upload temp")
	const other = "Takeout/cached.jpg"
	require.NoError(t, i.ImportZIP(context.Background(), archive(t, map[string]string{other: "cached"})))
	require.NoError(t, i.Close())
	i, err = openStagingTest(state, scope(), client, WithAvailableBytes(func(string) (uint64, error) { return storageReserve - 1, nil }))
	require.NoError(t, err)
	defer i.Close()
	require.True(t, i.StorageStatus().Fallback, "Open below the reserve persists fallback even without media arguments")
	require.NoError(t, i.ImportZIP(context.Background(), archive(t, map[string]string{other + ".json": `{"title":"cached.jpg"}`})))
	require.Equal(t, 2, client.uploads, "cached bytes can upload without another media write below reserve")
	require.Empty(t, payloads(t, directory))
}

func TestStorageStopsDirectUploadWorkspaceBeforeAttemptAndDuringCopy(t *testing.T) {
	for _, during := range []bool{false, true} {
		name := "before workspace"
		if during {
			name = "during workspace"
		}
		t.Run(name, func(t *testing.T) {
			base := t.TempDir()
			state, directory := filepath.Join(base, "state.json"), filepath.Join(base, "cache")
			client := &fakeClient{}
			i, err := openStagingTest(state, scope(), client, WithStaging(directory, 100), WithAvailableBytes(func(string) (uint64, error) {
				if !during {
					return storageReserve + 1, nil
				}
				workspaces, err := filepath.Glob(filepath.Join(directory, ".upload-*"))
				require.NoError(t, err)
				for _, file := range workspaces {
					info, err := os.Stat(file)
					require.NoError(t, err)
					if info.Size() > 0 {
						return storageReserve, nil
					}
				}
				return 100_000_000_000, nil
			}))
			require.NoError(t, err)
			defer i.Close()
			filename := archive(t, map[string]string{mediaPath: strings.Repeat("x", 256<<10), mediaPath + ".json": sidecar})
			require.NoError(t, i.ImportZIP(context.Background(), filename))
			require.True(t, i.Summary().Fallback)
			require.Zero(t, client.uploads)
			require.False(t, i.state.Media[i.state.Occurrences[mediaPath].Checksum].Attempted)
			require.Zero(t, actualStagingBytes(t, directory), "partial upload workspace must be removed")
			require.Contains(t, i.StorageStatus().Reason, "workspace")
			require.FileExists(t, filename)
		})
	}
}

func TestStorageFallbackDoesNotHideFailedPartialCleanup(t *testing.T) {
	base := t.TempDir()
	state, directory := filepath.Join(base, "state.json"), filepath.Join(base, "cache")
	i, err := openStagingTest(state, scope(), &fakeClient{}, WithStaging(directory, 1<<20))
	require.NoError(t, err)
	defer i.Close()
	defer func() { require.NoError(t, os.Chmod(directory, 0o700)) }()
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
	i.availableBytes = func(string) (uint64, error) {
		temps, err := filepath.Glob(filepath.Join(directory, ".tmp-*"))
		require.NoError(t, err)
		for _, temp := range temps {
			info, err := os.Stat(temp)
			require.NoError(t, err)
			if info.Size() > 0 {
				require.NoError(t, os.Chmod(directory, 0o500))
				return storageReserve - 1, nil
			}
		}
		return 100_000_000_000, nil
	}
	filename := archive(t, map[string]string{mediaPath: strings.Repeat("x", 256<<10)})
	err = i.ImportZIP(context.Background(), filename)
	require.ErrorIs(t, err, os.ErrPermission, "graceful fallback cannot swallow an actual cleanup failure")
	require.ErrorContains(t, err, "cleanup unfinished staging file")
	require.True(t, i.Summary().Fallback)
	require.Greater(t, actualStagingBytes(t, directory), int64(0))
	require.Equal(t, actualStagingBytes(t, directory), i.Summary().StagedBytes)
	require.NoError(t, os.Chmod(directory, 0o700))
	require.NoError(t, i.Reconcile(context.Background()), "no-file reconciliation can recover owned partial files")
	require.Zero(t, actualStagingBytes(t, directory))
	require.True(t, i.StorageStatus().Fallback)
	require.FileExists(t, filename)
}
