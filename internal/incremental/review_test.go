package incremental

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestReviewMediaMustWaitForCanonicalName(t *testing.T) {
	client := &fakeClient{}
	i, err := Open(filepath.Join(t.TempDir(), "state.json"), scope(), client)
	require.NoError(t, err)
	defer i.Close()
	require.NoError(t, i.ImportZIP(context.Background(), archive(t, map[string]string{mediaPath: "same photo"})))
	require.Zero(t, client.uploads, "archive basename must not become the irreversible originalFileName")
}

func TestReviewDuplicatePathMetadataReachesOwnedAsset(t *testing.T) {
	ctx := context.Background()
	client := &fakeClient{}
	i, err := Open(filepath.Join(t.TempDir(), "state.json"), scope(), client)
	require.NoError(t, err)
	defer i.Close()
	album := "Takeout/Google Photos/Album/photo.jpg"
	require.NoError(t, i.ImportZIP(ctx, archive(t, map[string]string{mediaPath: "same photo", mediaPath + ".json": `{"title":"photo.jpg"}`})))
	require.NoError(t, i.ImportZIP(ctx, archive(t, map[string]string{album: "same photo"})))
	require.NoError(t, i.ImportZIP(ctx, archive(t, map[string]string{album + ".json": sidecar})))
	require.Equal(t, 1, client.uploads, "same bytes at another source path must reuse the owned logical asset")
	require.Equal(t, "from Google", client.asset.ExifInfo.Description)
}

func TestDuplicateOccurrencesAlbumFirstYearFirstAndLateSidecar(t *testing.T) {
	const album = "Takeout/Google Photos/Album/photo.jpg"
	const titleOnly = `{"title":"photo.jpg"}`
	for _, firstPath := range []string{album, mediaPath} {
		t.Run(firstPath, func(t *testing.T) {
			ctx := context.Background()
			client := &fakeClient{}
			state := filepath.Join(t.TempDir(), "state.json")
			i, err := Open(state, scope(), client)
			require.NoError(t, err)
			secondPath := album
			if firstPath == album {
				secondPath = mediaPath
			}
			require.NoError(t, i.ImportZIP(ctx, archive(t, map[string]string{firstPath: "identical bytes", firstPath + ".json": titleOnly})))
			require.Zero(t, client.updates, "a filename-only sidecar must not send zero-date metadata")
			require.NoError(t, i.Close())
			i, err = Open(state, scope(), client)
			require.NoError(t, err)
			defer i.Close()
			require.NoError(t, i.ImportZIP(ctx, archive(t, map[string]string{secondPath: "identical bytes"})))
			require.Equal(t, 1, client.uploads)
			// Only the non-owning source occurrence gets the full sidecar, after
			// its media ZIP has already been processed and discarded.
			require.NoError(t, i.ImportZIP(ctx, archive(t, map[string]string{secondPath + ".json": sidecar})))
			require.Len(t, i.state.Media, 1)
			require.Len(t, i.state.Occurrences, 2)
			require.Equal(t, i.state.Occurrences[firstPath].Checksum, i.state.Occurrences[secondPath].Checksum)
			require.Equal(t, 1, client.uploads)
			require.Equal(t, 1, client.updates)
			require.Equal(t, "from Google", client.asset.ExifInfo.Description)
			require.Equal(t, 1, i.Summary().Enriched)
			require.Zero(t, i.Summary().Unowned)
		})
	}
}

func TestDeferredDuplicateMetadataNamesRePresentedMediaAfterRestart(t *testing.T) {
	ctx := context.Background()
	const album = "Takeout/Google Photos/Album/truncated(1).jpg"
	canonical := "a-much-longer-original-title-than-the-truncated-archive-basename.jpg"
	body := "media contents must never enter persisted state"
	client := &fakeClient{}
	state := filepath.Join(t.TempDir(), "state.json")
	i, err := Open(state, scope(), client)
	require.NoError(t, err)
	mediaZIP := archive(t, map[string]string{mediaPath: body, album: body})
	require.NoError(t, i.ImportZIP(ctx, mediaZIP))
	require.Zero(t, client.uploads)
	require.Equal(t, 1, i.Summary().Deferred)
	require.Equal(t, 2, i.Summary().Occurrences)
	data, err := os.ReadFile(state)
	require.NoError(t, err)
	require.NotContains(t, string(data), body)
	require.NoError(t, i.Close())
	i, err = Open(state, scope(), client)
	require.NoError(t, err)
	require.NoError(t, i.ImportZIP(ctx, archive(t, map[string]string{album + ".supplemental-metadata.json": `{"title":"` + canonical + `","photoTakenTime":{"timestamp":"1577836800"},"description":"from duplicate"}`})))
	require.Zero(t, client.uploads, "metadata-only import cannot resurrect discarded media")
	require.NoError(t, i.Close())
	i, err = Open(state, scope(), client)
	require.NoError(t, err)
	defer i.Close()
	// Either known occurrence can provide the bytes after metadata discovery.
	require.NoError(t, i.ImportZIP(ctx, archive(t, map[string]string{mediaPath: body})))
	require.Equal(t, []string{canonical}, client.names)
	require.Equal(t, "from duplicate", client.asset.ExifInfo.Description)
	require.Equal(t, 1, i.Summary().Enriched)
}

func TestPreexistingLogicalAssetNeverAcquiresOwnershipFromAnotherPath(t *testing.T) {
	ctx := context.Background()
	client := &fakeClient{byBytes: map[string]string{"photo": "asset-1"}}
	client.asset.OriginalFileName = "existing-name.jpg"
	client.asset.ExifInfo.Description = "existing user metadata"
	i, err := Open(filepath.Join(t.TempDir(), "state.json"), scope(), client)
	require.NoError(t, err)
	defer i.Close()
	const album = "Takeout/Google Photos/Album/photo.jpg"
	require.NoError(t, i.ImportZIP(ctx, archive(t, map[string]string{mediaPath: "photo", mediaPath + ".json": `{"title":"photo.jpg"}`})))
	require.NoError(t, i.ImportZIP(ctx, archive(t, map[string]string{album: "photo", album + ".json": sidecar})))
	require.Equal(t, 1, client.uploads)
	require.Zero(t, client.updates)
	require.Equal(t, "existing-name.jpg", client.asset.OriginalFileName)
	require.Equal(t, "existing user metadata", client.asset.ExifInfo.Description)
	require.Equal(t, 1, i.Summary().Unowned)
	require.Equal(t, 1, i.Summary().UnownedMetadata)
	require.Len(t, i.state.Media, 1)
	require.Len(t, i.Issues(), 2, "both source occurrences must expose withheld metadata")
}

func TestUnsupportedOrUnsafeItemDoesNotBlockUnrelatedMedia(t *testing.T) {
	for _, bad := range []string{
		`{"title":"../unsafe.jpg"}`,
		`{"title":"unsafe\\name.jpg"}`,
		`{"title":"unsafe\nname.jpg"}`,
		`{"title":"wrong-extension.mp4"}`,
		`{"title":""}`,
		`{"title":".jpg"}`,
		`{"title":"photo.jpg","photoTakenTime":{"timestamp":"0"}}`,
		`{"title":"photo.jpg","photoTakenTime":{"timestamp":"not-a-date"}}`,
		`{"title":"photo.jpg","photoTakenTime":{"timestamp":"9223372036854775807"}}`,
		`{malformed`,
	} {
		t.Run(bad, func(t *testing.T) {
			client := &fakeClient{}
			i, err := Open(filepath.Join(t.TempDir(), "state.json"), scope(), client)
			require.NoError(t, err)
			defer i.Close()
			require.NoError(t, i.ImportZIP(context.Background(), archive(t, map[string]string{
				mediaPath: "bad item", mediaPath + ".json": bad,
				"Takeout/good.jpg": "good item", "Takeout/good.jpg.json": `{"title":"good-original.jpg","photoTakenTime":{"timestamp":"1577836800"}}`,
			})))
			require.Equal(t, []string{"good-original.jpg"}, client.names)
			require.Equal(t, 1, client.updates)
			require.Equal(t, 1, i.Summary().Deferred)
			require.NotEmpty(t, i.Issues())
		})
	}
}

func TestUnsupportedDuplicateSuffixIsReportedWithoutGuessing(t *testing.T) {
	client := &fakeClient{}
	i, err := Open(filepath.Join(t.TempDir(), "state.json"), scope(), client)
	require.NoError(t, err)
	defer i.Close()
	require.NoError(t, i.ImportZIP(context.Background(), archive(t, map[string]string{
		"Takeout/photo(1).jpg":                            "unresolved photo",
		"Takeout/photo.jpg.supplemental-metadata(1).json": sidecar,
		"Takeout/good.jpg":                                "good photo", "Takeout/good.jpg.json": `{"title":"good.jpg"}`,
	})))
	require.Equal(t, []string{"good.jpg"}, client.names)
	require.Equal(t, 1, i.Summary().Deferred)
	require.Contains(t, fmt.Sprint(i.Issues()), "unsupported duplicate/truncated")
}

func TestSameArchiveDuplicateMetadataResolvesBeforeAnyUpload(t *testing.T) {
	client := &fakeClient{}
	i, err := Open(filepath.Join(t.TempDir(), "state.json"), scope(), client)
	require.NoError(t, err)
	defer i.Close()
	const album = "Takeout/Album/photo.jpg"
	require.NoError(t, i.ImportZIP(context.Background(), archive(t, map[string]string{
		mediaPath: "same", album: "same", album + ".json": sidecar,
	})))
	require.Equal(t, 1, client.uploads)
	require.Equal(t, 1, client.updates)
	require.Len(t, i.state.Media, 1)
}

func TestConflictingFieldsNeverChooseByArrivalOrder(t *testing.T) {
	const album = "Takeout/Album/photo.jpg"
	for _, candidate := range []string{
		`{"title":"other-name.jpg","description":"from Google","photoTakenTime":{"timestamp":"1577836800"}}`,
		`{"title":"photo.jpg","description":"other description","photoTakenTime":{"timestamp":"1577836800"}}`,
		`{"title":"photo.jpg","description":"from Google","photoTakenTime":{"timestamp":"1577836801"}}`,
	} {
		for _, reverse := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/reverse=%v", candidate, reverse), func(t *testing.T) {
				client := &fakeClient{}
				i, err := Open(filepath.Join(t.TempDir(), "state.json"), scope(), client)
				require.NoError(t, err)
				defer i.Close()
				metas := []map[string]string{{mediaPath + ".json": sidecar}, {album + ".json": candidate}}
				if reverse {
					metas[0], metas[1] = metas[1], metas[0]
				}
				for _, meta := range metas {
					require.NoError(t, i.ImportZIP(context.Background(), archive(t, meta)))
				}
				require.NoError(t, i.ImportZIP(context.Background(), archive(t, map[string]string{mediaPath: "same", album: "same"})))
				require.Zero(t, client.uploads)
				require.Equal(t, 1, i.Summary().Conflicts)
				require.Contains(t, fmt.Sprint(i.Issues()), "conflicting")
			})
		}
	}
}

func TestConflictingLaterNameIsReportedAndCannotRename(t *testing.T) {
	ctx := context.Background()
	client := &fakeClient{}
	i, err := Open(filepath.Join(t.TempDir(), "state.json"), scope(), client)
	require.NoError(t, err)
	defer i.Close()
	require.NoError(t, i.ImportZIP(ctx, archive(t, map[string]string{mediaPath: "same", mediaPath + ".json": sidecar})))
	require.NoError(t, i.ImportZIP(ctx, archive(t, map[string]string{
		"Takeout/Album/photo.jpg": "same", "Takeout/Album/photo.jpg.json": `{"title":"different-name.jpg","description":"late description"}`,
	})))
	require.Equal(t, []string{"photo.jpg"}, client.names)
	require.Equal(t, "photo.jpg", client.asset.OriginalFileName)
	require.Equal(t, 1, client.updates)
	require.Contains(t, fmt.Sprint(i.Issues()), "originalFileName cannot be renamed")
}

func TestAbsentFieldsMergeAndNeverClearPreviouslyKnownFields(t *testing.T) {
	ctx := context.Background()
	client := &fakeClient{}
	i, err := Open(filepath.Join(t.TempDir(), "state.json"), scope(), client)
	require.NoError(t, err)
	defer i.Close()
	require.NoError(t, i.ImportZIP(ctx, archive(t, map[string]string{mediaPath: "same", mediaPath + ".json": `{"title":"photo.jpg","description":"from Google"}`})))
	require.Zero(t, client.updates)
	require.NoError(t, i.ImportZIP(ctx, archive(t, map[string]string{
		"Takeout/Album/photo.jpg": "same", "Takeout/Album/photo.jpg.json": `{"title":"photo.jpg","photoTakenTime":{"timestamp":"1577836800"}}`,
	})))
	require.Equal(t, "from Google", client.asset.ExifInfo.Description)
	require.Equal(t, 1, client.updates)
	require.NoError(t, i.ImportZIP(ctx, archive(t, map[string]string{mediaPath + ".json": `{"title":"photo.jpg"}`})))
	require.Equal(t, 1, client.updates)
	require.Equal(t, "from Google", client.asset.ExifInfo.Description)
	for _, patch := range client.patches {
		require.False(t, patch.DateTimeOriginal.IsZero())
	}
}

func TestPostApplyDriftIsCheckedBeforeAddingNewFields(t *testing.T) {
	ctx := context.Background()
	client := &fakeClient{}
	i, err := Open(filepath.Join(t.TempDir(), "state.json"), scope(), client)
	require.NoError(t, err)
	defer i.Close()
	require.NoError(t, i.ImportZIP(ctx, archive(t, map[string]string{mediaPath: "same", mediaPath + ".json": `{"title":"photo.jpg","photoTakenTime":{"timestamp":"1577836800"}}`})))
	client.asset.ExifInfo.DateTimeOriginal.Time = time.Unix(1577836801, 0)
	require.ErrorContains(t, i.ImportZIP(ctx, archive(t, map[string]string{
		"Takeout/Album/photo.jpg": "same", "Takeout/Album/photo.jpg.json": sidecar,
	})), "metadata drifted after enrichment")
	require.Equal(t, 1, client.updates)
	require.Empty(t, client.asset.ExifInfo.Description)
}

func TestOldStateRefusedAndPreserved(t *testing.T) {
	state := filepath.Join(t.TempDir(), "state.json")
	old := []byte(`{"Version":1,"Scope":{"Server":"https://immich.example","UserID":"user-1","ExportID":"export-1"},"Media":{},"Metadata":{}}`)
	require.NoError(t, os.WriteFile(state, old, 0o600))
	_, err := Open(state, scope(), &fakeClient{})
	require.ErrorContains(t, err, "unsupported incremental state version 1")
	require.ErrorContains(t, err, "permanently misnamed uploads")
	require.ErrorContains(t, err, "new state file/export ID")
	actual, readErr := os.ReadFile(state)
	require.NoError(t, readErr)
	require.Equal(t, old, actual)
	_, err = os.Stat(state + ".lock")
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestLostUploadAckRemainsUnownedAfterRestartAndAnotherPath(t *testing.T) {
	ctx := context.Background()
	client := &fakeClient{loseReceipt: true}
	state := filepath.Join(t.TempDir(), "state.json")
	i, err := Open(state, scope(), client)
	require.NoError(t, err)
	require.ErrorContains(t, i.ImportZIP(ctx, archive(t, map[string]string{mediaPath: "same", mediaPath + ".json": sidecar})), "unknown outcomes")
	require.NoError(t, i.Close())
	i, err = Open(state, scope(), client)
	require.NoError(t, err)
	defer i.Close()
	require.Equal(t, 1, i.Summary().Unowned)
	require.NoError(t, i.ImportZIP(ctx, archive(t, map[string]string{
		"Takeout/Album/photo.jpg": "same", "Takeout/Album/photo.jpg.json": sidecar,
	})))
	require.Equal(t, 2, client.uploads)
	require.Zero(t, client.updates)
	require.Equal(t, 1, i.Summary().Unowned)
	require.Len(t, i.state.Media, 1)
}

func TestDifferentBytesProduceIndependentOwnedAssets(t *testing.T) {
	client := &fakeClient{}
	i, err := Open(filepath.Join(t.TempDir(), "state.json"), scope(), client)
	require.NoError(t, err)
	defer i.Close()
	require.NoError(t, i.ImportZIP(context.Background(), archive(t, map[string]string{
		mediaPath: "first bytes", mediaPath + ".json": sidecar,
		"Takeout/Album/photo.jpg": "second bytes", "Takeout/Album/photo.jpg.json": sidecar,
	})))
	require.Equal(t, 2, client.uploads)
	require.Equal(t, 2, client.updates)
	require.Len(t, i.state.Media, 2)
	require.Zero(t, i.Summary().Unowned)
}

func TestFilenameUsesUpstreamTitleNormalization(t *testing.T) {
	client := &fakeClient{}
	i, err := Open(filepath.Join(t.TempDir(), "state.json"), scope(), client)
	require.NoError(t, err)
	defer i.Close()
	require.NoError(t, i.ImportZIP(context.Background(), archive(t, map[string]string{mediaPath: "photo", mediaPath + ".json": `{"title":"original:title?.jpg"}`})))
	require.Equal(t, []string{"original_title_.jpg"}, client.names)
	require.Zero(t, client.updates)
}

func TestUnsafeSidecarStemAndArchivePathsRemainRestartable(t *testing.T) {
	client := &fakeClient{}
	state := filepath.Join(t.TempDir(), "state.json")
	i, err := Open(state, scope(), client)
	require.NoError(t, err)
	for _, source := range []string{".json", "Takeout/.json", "../escape.jpg.json", "C:/absolute.jpg.json", `Takeout\photo.jpg.json`} {
		require.NoError(t, i.ImportZIP(context.Background(), archive(t, map[string]string{source: sidecar})))
	}
	require.NoError(t, i.Close())
	i, err = Open(state, scope(), client)
	require.NoError(t, err)
	defer i.Close()
	require.Zero(t, client.uploads)
	require.Len(t, i.Issues(), 5)
}

func TestUnknownOutcomeCannotGainOwnershipEvenIfRetryCreatesAsset(t *testing.T) {
	ctx := context.Background()
	client := &fakeClient{loseReceipt: true}
	state := filepath.Join(t.TempDir(), "state.json")
	i, err := Open(state, scope(), client)
	require.NoError(t, err)
	z := archive(t, map[string]string{mediaPath: "same", mediaPath + ".json": sidecar})
	require.ErrorContains(t, i.ImportZIP(ctx, z), "unknown outcomes")
	require.NoError(t, i.Close())
	// Model an unknown first outcome with no surviving server asset. A retry's
	// created response still cannot prove what happened to the first request.
	client.byBytes = nil
	i, err = Open(state, scope(), client)
	require.NoError(t, err)
	defer i.Close()
	require.NoError(t, i.ImportZIP(ctx, z))
	require.Zero(t, client.updates)
	require.Equal(t, 1, i.Summary().Unowned)
}

func TestOrphanSidecarConflictsAreVisibleBeforeMediaArrives(t *testing.T) {
	i, err := Open(filepath.Join(t.TempDir(), "state.json"), scope(), &fakeClient{})
	require.NoError(t, err)
	defer i.Close()
	require.NoError(t, i.ImportZIP(context.Background(), archive(t, map[string]string{
		mediaPath + ".json":                       `{"title":"first.jpg"}`,
		mediaPath + ".supplemental-metadata.json": `{"title":"second.jpg"}`,
	})))
	require.Equal(t, 1, i.Summary().Conflicts)
	require.Contains(t, fmt.Sprint(i.Issues()), "conflicting canonical filenames")
}

func TestIncompleteV2StateIsRefusedAndPreserved(t *testing.T) {
	state := filepath.Join(t.TempDir(), "state.json")
	body := []byte(`{"Version":2}`)
	require.NoError(t, os.WriteFile(state, body, 0o600))
	_, err := Open(state, scope(), &fakeClient{})
	require.ErrorContains(t, err, "state scope or contents")
	actual, err := os.ReadFile(state)
	require.NoError(t, err)
	require.Equal(t, body, actual)
}
