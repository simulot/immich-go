package incremental

import (
	"archive/zip"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type archivePlanEntry struct {
	name, body string
	badCRC     bool
}

// Entry order and ZIP headers are deterministic, including duplicate-path and
// corrupt-entry fixtures. Each call gets an independent source directory.
func archivePlanZIP(t *testing.T, name string, entries ...archivePlanEntry) string {
	t.Helper()
	filename := filepath.Join(t.TempDir(), name)
	f, err := os.Create(filename)
	require.NoError(t, err)
	z := zip.NewWriter(f)
	for _, entry := range entries {
		header := &zip.FileHeader{Name: entry.name, Method: zip.Store}
		if entry.badCRC {
			header.CRC32 = crc32.ChecksumIEEE([]byte(entry.body)) ^ 1
			header.CompressedSize64, header.UncompressedSize64 = uint64(len(entry.body)), uint64(len(entry.body))
			w, err := z.CreateRaw(header)
			require.NoError(t, err)
			_, err = w.Write([]byte(entry.body))
			require.NoError(t, err)
		} else {
			w, err := z.CreateHeader(header)
			require.NoError(t, err)
			_, err = w.Write([]byte(entry.body))
			require.NoError(t, err)
		}
	}
	require.NoError(t, z.Close())
	require.NoError(t, f.Close())
	return filename
}

func archivePlanSession(t *testing.T, client *fakeClient) (*Importer, string) {
	t.Helper()
	state := filepath.Join(t.TempDir(), "state.json")
	i, err := Open(state, scope(), client)
	require.NoError(t, err)
	t.Cleanup(func() { _ = i.Close() })
	return i, state
}

func archivePlanCheck(t *testing.T, i *Importer, action, name string) ArchiveAction {
	t.Helper()
	next := i.NextArchive()
	require.Equal(t, action, next.Action, "%+v", next)
	require.Equal(t, name, next.Archive)
	require.NotEmpty(t, next.Reason)
	return next
}

func archivePlanChecksum(body string) string {
	h := sha1.Sum([]byte(body))
	return base64.StdEncoding.EncodeToString(h[:])
}

func TestArchivePlanOrderedScanThenReplayAndResume(t *testing.T) {
	ctx := context.Background()
	client := &fakeClient{}
	i, state := archivePlanSession(t, client)
	names := []string{"Takeout-A.zip", "Takeout-B.zip", "Takeout-C.zip"}
	require.NoError(t, i.ConfigureArchives(names))
	a := archivePlanZIP(t, names[0], archivePlanEntry{mediaPath, "photo", false})
	b := archivePlanZIP(t, names[1], archivePlanEntry{mediaPath + ".json", sidecar, false})
	c := archivePlanZIP(t, names[2], archivePlanEntry{"README.txt", "last part", false})
	archivePlanCheck(t, i, "download", names[0])
	require.NoError(t, i.ImportArchive(ctx, a))
	archivePlanCheck(t, i, "download", names[1])
	require.Zero(t, client.uploads)
	require.NoError(t, i.Close())
	resumed, err := Open(state, scope(), client)
	require.NoError(t, err)
	defer resumed.Close()
	i = resumed
	require.Equal(t, names, i.state.Archives.Expected)
	archivePlanCheck(t, i, "download", names[1])
	require.NoError(t, i.ImportArchive(ctx, b))
	archivePlanCheck(t, i, "download", names[2])
	require.NoError(t, i.Reconcile(ctx))
	require.Zero(t, client.uploads, "catalog-only bytes require their exact source again")
	require.NoError(t, i.ImportArchive(ctx, c))
	archivePlanCheck(t, i, "replay", names[0])
	require.NoError(t, i.ImportArchive(ctx, a))
	require.Equal(t, 1, client.uploads)
	require.Equal(t, 1, client.updates)
	archivePlanCheck(t, i, "complete", "")
	for _, filename := range []string{a, b, c, a} {
		require.NoError(t, i.ImportArchive(ctx, filename))
	}
	require.Equal(t, 1, client.uploads)
	require.Equal(t, 1, client.updates)
	data, err := os.ReadFile(a)
	require.NoError(t, err)
	digest := sha256.Sum256(data)
	require.Equal(t, hex.EncodeToString(digest[:]), i.state.Archives.Archives[names[0]].Digest)
}

func TestArchivePlanReversedArrivalAndDuplicateBytesSkipReplay(t *testing.T) {
	ctx := context.Background()
	client := &fakeClient{}
	i, _ := archivePlanSession(t, client)
	require.NoError(t, i.ConfigureArchives([]string{"A.zip", "B.zip", "C.zip"}))
	a := archivePlanZIP(t, "A.zip", archivePlanEntry{mediaPath, "photo", false})
	b := archivePlanZIP(t, "B.zip", archivePlanEntry{mediaPath + ".json", sidecar, false})
	c := archivePlanZIP(t, "C.zip", archivePlanEntry{"Takeout/Album/copy.jpg", "photo", false})
	require.NoError(t, i.ImportArchive(ctx, b))
	archivePlanCheck(t, i, "download", "A.zip")
	require.NoError(t, i.ImportArchive(ctx, c))
	require.Equal(t, 0, client.uploads, "sidecar association still needs the exact original occurrence")
	require.NoError(t, i.ImportArchive(ctx, a))
	require.Equal(t, 1, client.uploads)
	require.Len(t, i.state.Media, 1)
	require.Len(t, i.state.Occurrences, 2)
	archivePlanCheck(t, i, "complete", "")
	for _, source := range []string{c, a, b} {
		require.NoError(t, i.ImportArchive(ctx, source))
	}
	require.Equal(t, 1, client.uploads)
}

func TestArchivePlanEarliestReplaySkipsBytesUploadedFromAnotherArchive(t *testing.T) {
	ctx := context.Background()
	client := &fakeClient{}
	i, _ := archivePlanSession(t, client)
	require.NoError(t, i.ConfigureArchives([]string{"A.zip", "B.zip", "C.zip"}))
	a := archivePlanZIP(t, "A.zip", archivePlanEntry{mediaPath, "first", false})
	b := archivePlanZIP(t, "B.zip", archivePlanEntry{mediaPath, "first", false}, archivePlanEntry{mediaPath + ".json", sidecar, false})
	c := archivePlanZIP(t, "C.zip", archivePlanEntry{"Takeout/second.jpg", "second", false}, archivePlanEntry{"Takeout/second.jpg.json", `{"title":"second.jpg","photoTakenTime":{"timestamp":"1577836800"}}`, false})
	require.NoError(t, i.ImportArchive(ctx, a))
	require.NoError(t, i.ImportArchive(ctx, b))
	archivePlanCheck(t, i, "download", "C.zip")
	require.NoError(t, i.ImportArchive(ctx, c))
	archivePlanCheck(t, i, "complete", "")
	require.Equal(t, 2, client.uploads, "A must not be requested after B uploaded its bytes")
}

func TestArchivePlanMultipleReplaysFollowManifestOrder(t *testing.T) {
	ctx := context.Background()
	client := &fakeClient{}
	i, _ := archivePlanSession(t, client)
	require.NoError(t, i.ConfigureArchives([]string{"Z.zip", "A.zip", "M.zip"}))
	const second = "Takeout/second.jpg"
	z := archivePlanZIP(t, "Z.zip", archivePlanEntry{mediaPath, "first", false})
	a := archivePlanZIP(t, "A.zip", archivePlanEntry{second, "second", false})
	m := archivePlanZIP(t, "M.zip", archivePlanEntry{mediaPath + ".json", sidecar, false}, archivePlanEntry{second + ".json", `{"title":"second.jpg","photoTakenTime":{"timestamp":"1577836800"}}`, false})
	for _, filename := range []string{a, z, m} {
		require.NoError(t, i.ImportArchive(ctx, filename))
	}
	archivePlanCheck(t, i, "replay", "Z.zip")
	require.NoError(t, i.ImportArchive(ctx, z))
	archivePlanCheck(t, i, "replay", "A.zip")
	require.NoError(t, i.ImportArchive(ctx, a))
	archivePlanCheck(t, i, "complete", "")
	require.Equal(t, 2, client.uploads)
}

func TestArchivePlanCurrentCatalogHoldPrecedesUnseenOrReplay(t *testing.T) {
	i, _ := archivePlanSession(t, &fakeClient{})
	require.NoError(t, i.ConfigureArchives([]string{"A.zip", "B.zip", "C.zip"}))
	a := archivePlanZIP(t, "A.zip", archivePlanEntry{mediaPath, "bad first", true})
	b := archivePlanZIP(t, "B.zip", archivePlanEntry{mediaPath, "bad second", true})
	require.Error(t, i.ImportArchive(context.Background(), a))
	require.Error(t, i.ImportArchive(context.Background(), b))
	archivePlanCheck(t, i, "keep/retry", "B.zip")
}

func TestArchivePlanUnknownInventoryAndFirstManifestAfterSeen(t *testing.T) {
	i, _ := archivePlanSession(t, &fakeClient{})
	guide := archivePlanCheck(t, i, "inventory", "")
	require.Contains(t, guide.Reason, "--archive-list")
	a := archivePlanZIP(t, "A.zip", archivePlanEntry{mediaPath, "photo", false})
	require.NoError(t, i.ImportArchive(context.Background(), a))
	archivePlanCheck(t, i, "inventory", "")
	require.ErrorContains(t, i.ConfigureArchives([]string{"B.zip"}), "omits already recorded")
	require.Empty(t, i.state.Archives.Expected)
	names := []string{"A.zip", "B.zip"}
	require.NoError(t, i.ConfigureArchives(names))
	names[1] = "caller-mutated.zip"
	archivePlanCheck(t, i, "download", "B.zip")
	require.NoError(t, i.ConfigureArchives([]string{"A.zip", "B.zip"}))
	for _, changed := range [][]string{{"B.zip", "A.zip"}, {"A.zip"}, {"A.zip", "B.zip", "C.zip"}} {
		require.ErrorContains(t, i.ConfigureArchives(changed), "differs from the persisted")
	}
}

func TestArchivePlanManifestValidationIsAtomicAndCaseSensitive(t *testing.T) {
	for _, names := range [][]string{
		nil,
		{},
		{"A.zip", "A.zip"},
		{"../A.zip"},
		{"a/A.zip"},
		{`a\A.zip`},
		{"https://example.test/A.zip"},
		{"/A.zip"},
		{"a.tar"},
		{" A.zip"},
		{"A.zip "},
		{"A\n.zip"},
		{".zip"},
		{"a:part.zip"},
		{"A\u202e.zip"},
	} {
		t.Run(fmt.Sprint(names), func(t *testing.T) {
			i, state := archivePlanSession(t, &fakeClient{})
			before, err := os.ReadFile(state)
			require.NoError(t, err)
			require.Error(t, i.ConfigureArchives(names))
			after, err := os.ReadFile(state)
			require.NoError(t, err)
			require.Equal(t, before, after)
			require.Nil(t, i.state.Archives)
		})
	}
	i, _ := archivePlanSession(t, &fakeClient{})
	require.NoError(t, i.ConfigureArchives([]string{"Part.zip", "part.zip", "LAST.ZIP"}))
	archivePlanCheck(t, i, "download", "Part.zip")
}

func TestArchivePlanPreflightRejectsUnlistedAndChangedBytesBeforeMutation(t *testing.T) {
	ctx := context.Background()
	client := &fakeClient{}
	i, state := archivePlanSession(t, client)
	require.NoError(t, i.ConfigureArchives([]string{"A.zip"}))
	a := archivePlanZIP(t, "A.zip", archivePlanEntry{mediaPath, "photo", false}, archivePlanEntry{mediaPath + ".json", sidecar, false})
	require.NoError(t, i.ImportArchive(ctx, a))
	before, err := os.ReadFile(state)
	require.NoError(t, err)
	for _, test := range []struct{ name, expected string }{{"B.zip", "not listed"}, {"a.zip", "not listed"}, {"A.zip", "changed compressed bytes"}, {"a.txt", "ZIP basename"}} {
		t.Run(test.name, func(t *testing.T) {
			other := archivePlanZIP(t, test.name, archivePlanEntry{mediaPath, "different export bytes", false}, archivePlanEntry{mediaPath + ".json", `{"title":"different.jpg"}`, false})
			require.ErrorContains(t, i.ImportArchive(ctx, other), test.expected)
			after, err := os.ReadFile(state)
			require.NoError(t, err)
			require.Equal(t, before, after)
			require.Equal(t, 1, client.uploads)
			require.Empty(t, i.state.Occurrences[mediaPath].ConflictingChecksums)
		})
	}
}

func TestArchivePlanRenamingBadSourceDoesNotEscapeHold(t *testing.T) {
	i, _ := archivePlanSession(t, &fakeClient{})
	require.NoError(t, i.ConfigureArchives([]string{"A.zip", "B.zip"}))
	a := archivePlanZIP(t, "A.zip", archivePlanEntry{mediaPath, "bad photo", true})
	require.ErrorContains(t, i.ImportArchive(context.Background(), a), "checksum error")
	data, err := os.ReadFile(a)
	require.NoError(t, err)
	b := filepath.Join(t.TempDir(), "B.zip")
	require.NoError(t, os.WriteFile(b, data, 0o600))
	require.ErrorContains(t, i.ImportArchive(context.Background(), b), "original filename")
	archivePlanCheck(t, i, "keep/retry", "A.zip")
	require.Len(t, i.state.Archives.Archives, 1)
}

func TestArchivePlanRepairedFailedSourcePreservesHealthyProvenanceAndReceipts(t *testing.T) {
	for _, badSidecar := range []bool{false, true} {
		t.Run(fmt.Sprint(badSidecar), func(t *testing.T) {
			ctx := context.Background()
			client := &fakeClient{}
			i, state := archivePlanSession(t, client)
			require.NoError(t, i.ConfigureArchives([]string{"A.zip"}))
			const healthy = "Takeout/healthy.jpg"
			entries := []archivePlanEntry{
				{healthy, "healthy photo", false},
				{healthy + ".json", `{"title":"healthy.jpg","photoTakenTime":{"timestamp":"1577836800"}}`, false},
				{mediaPath, "photo", !badSidecar},
				{mediaPath + ".json", sidecar, badSidecar},
			}
			a := archivePlanZIP(t, "A.zip", entries...)
			require.ErrorContains(t, i.ImportArchive(ctx, a), "checksum error")
			original := i.state.Archives.Archives["A.zip"]
			require.Equal(t, 1, client.uploads)
			require.NoError(t, i.Close())
			resumed, err := Open(state, scope(), client)
			require.NoError(t, err)
			defer resumed.Close()
			entries[2].badCRC, entries[3].badCRC = false, false
			repaired := archivePlanZIP(t, "A.zip", entries...)
			require.NoError(t, resumed.ImportArchive(ctx, repaired))
			record := resumed.state.Archives.Archives["A.zip"]
			require.NotEqual(t, original.Digest, record.Digest)
			require.True(t, record.Scanned)
			require.Empty(t, record.CatalogError)
			require.Equal(t, 2, client.uploads, "healthy first-pass receipt must be reused")
			require.Equal(t, 2, client.updates)
			archivePlanCheck(t, resumed, "complete", "")
			require.NoError(t, resumed.ImportArchive(ctx, repaired))
			require.Equal(t, 2, client.uploads)
			require.ErrorContains(t, resumed.ImportArchive(ctx, a), "after a completed catalog")
		})
	}
}

func TestArchivePlanRepairRejectsChangedHealthyBytesOrMetadataWithoutMutation(t *testing.T) {
	for _, change := range []string{"changed media", "omitted media", "changed sidecar", "omitted sidecar", "still corrupt"} {
		t.Run(change, func(t *testing.T) {
			client := &fakeClient{}
			i, state := archivePlanSession(t, client)
			require.NoError(t, i.ConfigureArchives([]string{"A.zip"}))
			entries := []archivePlanEntry{{mediaPath, "healthy", false}, {mediaPath + ".json", sidecar, false}, {"Takeout/bad.jpg", "bad", true}}
			a := archivePlanZIP(t, "A.zip", entries...)
			require.Error(t, i.ImportArchive(context.Background(), a))
			before, err := os.ReadFile(state)
			require.NoError(t, err)
			original := i.state.Archives.Archives["A.zip"]
			entries[2].badCRC = false
			switch change {
			case "changed media":
				entries[0].body = "other export"
			case "omitted media":
				entries = entries[1:]
			case "changed sidecar":
				entries[1].body = strings.Replace(sidecar, "from Google", "another export", 1)
			case "omitted sidecar":
				entries = append(entries[:1], entries[2:]...)
			case "still corrupt":
				entries[2].badCRC = true
				entries = append(entries, archivePlanEntry{"README.txt", "different compressed bytes", false})
			}
			replacement := archivePlanZIP(t, "A.zip", entries...)
			require.Error(t, i.ImportArchive(context.Background(), replacement))
			after, err := os.ReadFile(state)
			require.NoError(t, err)
			require.Equal(t, before, after, "incompatible repair must precede ImportZIP mutations")
			require.Equal(t, original, i.state.Archives.Archives["A.zip"])
			require.Equal(t, 1, client.uploads)
			archivePlanCheck(t, i, "keep/retry", "A.zip")
		})
	}
}

func TestArchivePlanReadIdentityFailureKeepsHealthyRepairProvenance(t *testing.T) {
	i, _ := archivePlanSession(t, &fakeClient{})
	require.NoError(t, i.ConfigureArchives([]string{"A.zip"}))
	entries := []archivePlanEntry{{mediaPath, "photo", true}, {mediaPath + ".json", sidecar, false}}
	a := archivePlanZIP(t, "A.zip", entries...)
	require.Error(t, i.ImportArchive(context.Background(), a))
	before := i.state.Archives.Archives["A.zip"]
	require.NoError(t, os.Remove(a))
	require.Error(t, i.ImportArchive(context.Background(), a))
	after := i.state.Archives.Archives["A.zip"]
	require.Equal(t, before.Digest, after.Digest)
	require.Equal(t, before.Sidecars, after.Sidecars)
	require.NoError(t, i.validateArchives())
	entries[0].badCRC = false
	require.NoError(t, i.ImportArchive(context.Background(), archivePlanZIP(t, "A.zip", entries...)))
	archivePlanCheck(t, i, "complete", "")
}

func TestArchivePlanCompletedIdentityCannotBeReopenedByAnUnreadableLocalCopy(t *testing.T) {
	ctx := context.Background()
	client := &fakeClient{}
	i, state := archivePlanSession(t, client)
	require.NoError(t, i.ConfigureArchives([]string{"A.zip"}))
	entries := make([]archivePlanEntry, 0, 3)
	entries = append(entries, archivePlanEntry{mediaPath, "photo", false}, archivePlanEntry{mediaPath + ".json", sidecar, false})
	a := archivePlanZIP(t, "A.zip", entries...)
	require.NoError(t, i.ImportArchive(ctx, a))
	data, err := os.ReadFile(a)
	require.NoError(t, err)
	require.NoError(t, os.Remove(a))
	require.Error(t, i.ImportArchive(ctx, a))
	archivePlanCheck(t, i, "keep/retry", "A.zip")
	require.NoError(t, i.Close())
	resumed, err := Open(state, scope(), client)
	require.NoError(t, err)
	defer resumed.Close()
	other := archivePlanZIP(t, "A.zip", append(entries, archivePlanEntry{"README.txt", "different archive identity", false})...)
	require.ErrorContains(t, resumed.ImportArchive(ctx, other), "after a completed catalog")
	require.NoError(t, os.WriteFile(a, data, 0o600))
	require.NoError(t, resumed.ImportArchive(ctx, a))
	archivePlanCheck(t, resumed, "complete", "")
}

func TestArchivePlanCatalogFailuresHoldSourceAndPreserveHealthyProgress(t *testing.T) {
	const bad = "Takeout/bad.jpg"
	tests := []struct {
		name, want string
		extra      []archivePlanEntry
	}{
		{"media CRC", "checksum error", []archivePlanEntry{{bad, "bad", true}}},
		{"sidecar CRC", "checksum error", []archivePlanEntry{{bad + ".json", sidecar, true}}},
		{"unsafe media", "unsafe archive path", []archivePlanEntry{{"../unsafe.jpg", "bad", false}}},
		{"unsafe sidecar", "unsafe archive path", []archivePlanEntry{{"../unsafe.jpg.json", sidecar, false}}},
		{"duplicate media", "duplicate ZIP entry", []archivePlanEntry{{bad, "first", false}, {bad, "second", false}}},
		{"duplicate sidecar", "duplicate ZIP entry", []archivePlanEntry{{bad + ".json", sidecar, false}, {bad + ".json", sidecar, false}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			client := &fakeClient{}
			i, state := archivePlanSession(t, client)
			require.NoError(t, i.ConfigureArchives([]string{"A.zip", "B.zip"}))
			entries := []archivePlanEntry{{mediaPath, "healthy photo", false}, {mediaPath + ".json", sidecar, false}}
			a := archivePlanZIP(t, "A.zip", append(entries, test.extra...)...)
			require.ErrorContains(t, i.ImportArchive(ctx, a), test.want)
			require.Equal(t, 1, client.uploads)
			require.Equal(t, 1, client.updates)
			require.False(t, i.state.Archives.Archives["A.zip"].Scanned)
			guide := archivePlanCheck(t, i, "keep/retry", "A.zip")
			require.Contains(t, guide.Reason, "retry it locally")
			require.FileExists(t, a)
			require.NoError(t, i.Close())
			resumed, err := Open(state, scope(), client)
			require.NoError(t, err)
			defer resumed.Close()
			archivePlanCheck(t, resumed, "keep/retry", "A.zip")
			require.ErrorContains(t, resumed.ImportArchive(ctx, a), test.want)
			require.Equal(t, 1, client.uploads, "healthy durable receipts survive a bad source retry")
		})
	}
}

func TestArchivePlanConflictingSourceRecordsActualBytesAndHealthySibling(t *testing.T) {
	ctx := context.Background()
	client := &fakeClient{}
	i, _ := archivePlanSession(t, client)
	require.NoError(t, i.ConfigureArchives([]string{"A.zip", "B.zip"}))
	a := archivePlanZIP(t, "A.zip", archivePlanEntry{mediaPath, "original bytes", false}, archivePlanEntry{mediaPath + ".json", sidecar, false})
	require.NoError(t, i.ImportArchive(ctx, a))
	const healthy = "Takeout/healthy.jpg"
	b := archivePlanZIP(t, "B.zip", archivePlanEntry{mediaPath, "different bytes", false}, archivePlanEntry{healthy, "healthy bytes", false}, archivePlanEntry{healthy + ".json", `{"title":"healthy.jpg","photoTakenTime":{"timestamp":"1577836800"}}`, false})
	require.ErrorContains(t, i.ImportArchive(ctx, b), "actual checksum")
	require.Equal(t, 2, client.uploads, "an unrelated healthy source must proceed")
	r := i.state.Archives.Archives["B.zip"]
	require.Contains(t, r.Media, archivePlanChecksum("different bytes"))
	require.NotContains(t, r.Media, archivePlanChecksum("original bytes"))
	require.Equal(t, archivePlanChecksum("different bytes"), r.Sources[mediaPath])
	require.Equal(t, archivePlanChecksum("original bytes"), i.state.Occurrences[mediaPath].Checksum)
	require.NotContains(t, i.state.Media, archivePlanChecksum("different bytes"))
	archivePlanCheck(t, i, "keep/retry", "B.zip")
}

func TestArchivePlanReadyCachedAckFailureReconcilesBeforeMoreDownloads(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("staging intentionally unavailable on Windows")
	}
	ctx := context.Background()
	client := &fakeClient{loseReceipt: true}
	base := t.TempDir()
	state := filepath.Join(base, "state.json")
	i, err := Open(state, scope(), client, WithStaging(filepath.Join(base, "cache"), 100), WithAvailableBytes(func(string) (uint64, error) { return 100_000_000_000, nil }))
	require.NoError(t, err)
	require.NoError(t, i.ConfigureArchives([]string{"A.zip", "B.zip", "C.zip"}))
	a := archivePlanZIP(t, "A.zip", archivePlanEntry{mediaPath, "photo", false})
	b := archivePlanZIP(t, "B.zip", archivePlanEntry{mediaPath + ".json", sidecar, false})
	require.NoError(t, i.ImportArchive(ctx, a))
	require.ErrorContains(t, i.ImportArchive(ctx, b), "lost upload acknowledgement")
	require.True(t, i.state.Archives.Archives["B.zip"].Scanned)
	require.Empty(t, i.state.Archives.Archives["B.zip"].CatalogError)
	guide := archivePlanCheck(t, i, "reconcile", "")
	require.Contains(t, guide.Reason, "no ZIP download")
	require.NoError(t, os.Remove(a))
	require.NoError(t, os.Remove(b))
	require.NoError(t, i.Close())
	i, err = Open(state, scope(), client, WithAvailableBytes(func(string) (uint64, error) { return 100_000_000_000, nil }))
	require.NoError(t, err)
	defer i.Close()
	archivePlanCheck(t, i, "reconcile", "")
	require.NoError(t, i.Reconcile(ctx))
	require.Equal(t, 2, client.uploads)
	require.Len(t, client.byBytes, 1)
	require.Zero(t, client.updates, "a lost receipt must remain unowned")
	archivePlanCheck(t, i, "download", "C.zip")
	c := archivePlanZIP(t, "C.zip", archivePlanEntry{"README.txt", "end", false})
	require.NoError(t, i.ImportArchive(ctx, c))
	guide = archivePlanCheck(t, i, "attention", "")
	require.Contains(t, guide.Reason, "unowned")
}

type archivePlanCacheFixture struct {
	state, directory, source, payload string
	client                            *fakeClient
}

// Keep the original server receipt and its unknown acknowledgement across all
// restarts. Only the disposable local payload is damaged by recovery tests.
func archivePlanCachedAckLoss(t *testing.T, tracked bool) archivePlanCacheFixture {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("staging intentionally unavailable on Windows")
	}
	base := t.TempDir()
	f := archivePlanCacheFixture{
		state: filepath.Join(base, "state.json"), directory: filepath.Join(base, "cache"),
		client: &fakeClient{loseReceipt: true},
	}
	i, err := Open(f.state, scope(), f.client, WithStaging(f.directory, 100), WithAvailableBytes(func(string) (uint64, error) { return 100_000_000_000, nil }))
	require.NoError(t, err)
	t.Cleanup(func() { _ = i.Close() })
	importFile := i.ImportZIP
	if tracked {
		require.NoError(t, i.ConfigureArchives([]string{"A.zip", "B.zip", "C.zip"}))
		importFile = i.ImportArchive
	}
	f.source = archivePlanZIP(t, "A.zip", archivePlanEntry{mediaPath, "photo", false})
	metadata := archivePlanZIP(t, "B.zip", archivePlanEntry{mediaPath + ".json", sidecar, false})
	require.NoError(t, importFile(context.Background(), f.source))
	require.ErrorContains(t, importFile(context.Background(), metadata), "lost upload acknowledgement")
	payload, err := payloadName(archivePlanChecksum("photo"))
	require.NoError(t, err)
	f.payload = filepath.Join(f.directory, payload)
	require.NoError(t, i.Close())
	return f
}

func damageArchivePlanCache(t *testing.T, payload, damage string) {
	t.Helper()
	switch damage {
	case "missing":
		require.NoError(t, os.Remove(payload))
	case "size":
		require.NoError(t, os.WriteFile(payload, []byte("incorrect-size"), 0o600))
	case "checksum":
		require.NoError(t, os.WriteFile(payload, []byte("other"), 0o600))
	default:
		t.Fatalf("unknown cache damage %q", damage)
	}
}

func openArchivePlanCache(t *testing.T, f archivePlanCacheFixture) *Importer {
	t.Helper()
	i, err := Open(f.state, scope(), f.client, WithAvailableBytes(func(string) (uint64, error) { return 100_000_000_000, nil }))
	require.NoError(t, err)
	t.Cleanup(func() { _ = i.Close() })
	return i
}

func TestArchivePlanDamagedCacheNamesSourceAndRecoversWithoutOwningUnknownUpload(t *testing.T) {
	for _, damage := range []string{"missing", "size", "checksum"} {
		for _, recovery := range []string{"restore cache", "re-present source"} {
			t.Run(damage+"/"+recovery, func(t *testing.T) {
				ctx := context.Background()
				f := archivePlanCachedAckLoss(t, true)
				damageArchivePlanCache(t, f.payload, damage)
				i := openArchivePlanCache(t, f)
				checksum := archivePlanChecksum("photo")
				original := i.state.Media[checksum]
				require.True(t, original.Staged)
				require.True(t, original.Attempted)
				require.Empty(t, original.AssetID)
				require.False(t, original.Owned)
				var damagedBytes []byte
				if damage != "missing" {
					var err error
					damagedBytes, err = os.ReadFile(f.payload)
					require.NoError(t, err)
				}
				stateBefore, err := os.ReadFile(f.state)
				require.NoError(t, err)
				for range 2 {
					require.ErrorContains(t, i.Reconcile(ctx), "staged payload")
					guide := archivePlanCheck(t, i, "keep/retry", "A.zip")
					require.Contains(t, guide.Reason, "Restore")
					require.Contains(t, guide.Reason, "re-present")
					require.Contains(t, guide.Reason, "re-download")
					require.NotContains(t, guide.Reason, "no ZIP download")
					require.Equal(t, original, i.state.Media[checksum], "guidance and failed retries preserve all receipt/ownership fields")
					if damage == "missing" {
						require.NoFileExists(t, f.payload)
					} else {
						after, err := os.ReadFile(f.payload)
						require.NoError(t, err)
						require.Equal(t, damagedBytes, after, "guidance never discards damaged payloads")
					}
				}
				stateAfter, err := os.ReadFile(f.state)
				require.NoError(t, err)
				require.Equal(t, stateBefore, stateAfter)
				require.NoError(t, i.Close())
				i = openArchivePlanCache(t, f)
				archivePlanCheck(t, i, "keep/retry", "A.zip")
				if recovery == "restore cache" {
					require.NoError(t, os.WriteFile(f.payload, []byte("photo"), 0o600))
					archivePlanCheck(t, i, "reconcile", "")
					require.NoError(t, i.Reconcile(ctx))
				} else {
					// ImportZIP reports its first failed cache drain even when the
					// source replay subsequently obtains a durable duplicate receipt.
					require.ErrorContains(t, i.ImportArchive(ctx, f.source), "staged payload")
					require.NoError(t, i.Reconcile(ctx))
				}
				m := i.state.Media[checksum]
				require.Equal(t, "asset-1", m.AssetID)
				require.False(t, m.Staged)
				require.False(t, m.Owned, "repair cannot confer ownership after an unknown upload outcome")
				require.Nil(t, m.Baseline)
				require.Nil(t, m.Applied)
				require.NoFileExists(t, f.payload, "cleanup follows the durable upload receipt")
				archivePlanCheck(t, i, "download", "C.zip")
				require.NoError(t, i.ImportArchive(ctx, archivePlanZIP(t, "C.zip", archivePlanEntry{"README.txt", "last archive", false})))
				archivePlanCheck(t, i, "attention", "")
				for range 2 {
					require.NoError(t, i.ImportArchive(ctx, f.source))
					require.NoError(t, i.Reconcile(ctx))
				}
				require.Equal(t, 2, f.client.uploads)
				require.Len(t, f.client.byBytes, 1)
				require.Zero(t, f.client.updates)
			})
		}
	}
}

func TestArchivePlanDamagedCacheWithoutProvenanceBlocksWithoutInventingFilename(t *testing.T) {
	for _, damage := range []string{"missing", "size", "checksum"} {
		t.Run(damage, func(t *testing.T) {
			f := archivePlanCachedAckLoss(t, false)
			damageArchivePlanCache(t, f.payload, damage)
			i := openArchivePlanCache(t, f)
			require.Nil(t, i.state.Archives, "legacy ImportZIP has no archive-filename provenance")
			for range 2 {
				require.ErrorContains(t, i.Reconcile(context.Background()), "staged payload")
				guide := archivePlanCheck(t, i, "blocked", "")
				require.Contains(t, guide.Reason, "no source ZIP filename was recorded")
				require.Contains(t, guide.Reason, "original media ZIP")
				require.Contains(t, guide.Reason, "Restore")
				require.True(t, i.state.Media[archivePlanChecksum("photo")].Staged)
			}
			require.Equal(t, 1, f.client.uploads)
			require.Zero(t, f.client.updates)
		})
	}
}

func TestArchivePlanMetadataOnlyRetryDoesNotRequireZIP(t *testing.T) {
	ctx := context.Background()
	client := &fakeClient{}
	i, state := archivePlanSession(t, client)
	require.NoError(t, i.ConfigureArchives([]string{"A.zip", "B.zip"}))
	a := archivePlanZIP(t, "A.zip", archivePlanEntry{mediaPath, "photo", false}, archivePlanEntry{mediaPath + ".json", `{"title":"photo.jpg"}`, false})
	b := archivePlanZIP(t, "B.zip", archivePlanEntry{mediaPath + ".json", sidecar, false})
	require.NoError(t, i.ImportArchive(ctx, a))
	client.failUpdate = true
	require.ErrorContains(t, i.ImportArchive(ctx, b), "pending metadata")
	require.True(t, i.state.Archives.Archives["B.zip"].Scanned)
	require.Empty(t, i.state.Archives.Archives["B.zip"].Media)
	archivePlanCheck(t, i, "reconcile", "")
	require.NoError(t, os.Remove(a))
	require.NoError(t, os.Remove(b))
	require.NoError(t, i.Close())
	client.failUpdate = false
	resumed, err := Open(state, scope(), client)
	require.NoError(t, err)
	defer resumed.Close()
	archivePlanCheck(t, resumed, "reconcile", "")
	require.NoError(t, resumed.Reconcile(ctx))
	require.Equal(t, 1, client.uploads)
	archivePlanCheck(t, resumed, "complete", "")
}

func TestArchivePlanDirectUploadHTTPErrorIsScannedAndReplayable(t *testing.T) {
	client := &fakeClient{loseReceipt: true}
	i, _ := archivePlanSession(t, client)
	require.NoError(t, i.ConfigureArchives([]string{"A.zip"}))
	a := archivePlanZIP(t, "A.zip", archivePlanEntry{mediaPath, "photo", false}, archivePlanEntry{mediaPath + ".json", sidecar, false})
	require.ErrorContains(t, i.ImportArchive(context.Background(), a), "lost upload acknowledgement")
	require.True(t, i.state.Archives.Archives["A.zip"].Scanned)
	archivePlanCheck(t, i, "replay", "A.zip")
	require.NoError(t, i.ImportArchive(context.Background(), a))
	archivePlanCheck(t, i, "attention", "")
	require.Zero(t, client.updates)
}

func TestArchivePlanUnresolvedEndBlocksWithoutInfiniteRedownload(t *testing.T) {
	for _, test := range []struct {
		name, want string
		entries    []archivePlanEntry
	}{
		{"missing filename", "canonical filename unresolved", []archivePlanEntry{{mediaPath, "photo", false}}},
		{"orphan metadata", "sidecar has no known media occurrence", []archivePlanEntry{{mediaPath + ".json", sidecar, false}}},
		{"invalid JSON", "sidecar has no known media occurrence", []archivePlanEntry{{mediaPath + ".json", "{invalid", false}}},
		{"missing capture date", "validated capture date", []archivePlanEntry{{mediaPath, "photo", false}, {mediaPath + ".json", `{"title":"photo.jpg"}`, false}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			i, _ := archivePlanSession(t, &fakeClient{})
			require.NoError(t, i.ConfigureArchives([]string{"A.zip"}))
			require.NoError(t, i.ImportArchive(context.Background(), archivePlanZIP(t, "A.zip", test.entries...)))
			for range 2 {
				guide := archivePlanCheck(t, i, "blocked", "")
				require.Contains(t, guide.Reason, test.want)
			}
			require.NotEmpty(t, i.Issues())
		})
	}
}

func TestArchivePlanConflictingNamesAndMetadataDriftNeverReplay(t *testing.T) {
	for _, conflict := range []string{"filename", "server drift"} {
		t.Run(conflict, func(t *testing.T) {
			client := &fakeClient{}
			i, _ := archivePlanSession(t, client)
			require.NoError(t, i.ConfigureArchives([]string{"A.zip", "B.zip"}))
			a := archivePlanZIP(t, "A.zip", archivePlanEntry{mediaPath, "photo", false}, archivePlanEntry{mediaPath + ".json", sidecar, false})
			require.NoError(t, i.ImportArchive(context.Background(), a))
			metadata := sidecar
			if conflict == "filename" {
				metadata = strings.Replace(sidecar, "photo.jpg", "other.jpg", 1)
			} else {
				client.asset.ExifInfo.Description = "user edit"
			}
			b := archivePlanZIP(t, "B.zip", archivePlanEntry{mediaPath + ".json", metadata, false})
			err := i.ImportArchive(context.Background(), b)
			if conflict == "server drift" {
				require.ErrorContains(t, err, "refusing to overwrite")
			} else {
				require.NoError(t, err)
			}
			archivePlanCheck(t, i, "blocked", "")
			require.Equal(t, 1, client.uploads)
		})
	}
}

func TestArchivePlanMalformedPersistedPlansBlockEveryPlanningAPI(t *testing.T) {
	tests := []struct {
		name   string
		change func(*ArchivePlan)
	}{
		{"missing records", func(p *ArchivePlan) { p.Archives = nil }},
		{"unsafe expected", func(p *ArchivePlan) { p.Expected = []string{"../A.zip"} }},
		{"duplicate expected", func(p *ArchivePlan) { p.Expected = []string{"A.zip", "A.zip"} }},
		{"unlisted record", func(p *ArchivePlan) { p.Expected = []string{"B.zip"} }},
		{"missing current", func(p *ArchivePlan) { p.Current = "missing.zip" }},
		{"bad digest", func(p *ArchivePlan) { r := p.Archives["A.zip"]; r.Digest = "invalid"; p.Archives["A.zip"] = r }},
		{"empty digest", func(p *ArchivePlan) { r := p.Archives["A.zip"]; r.Digest = ""; p.Archives["A.zip"] = r }},
		{"bad scan status", func(p *ArchivePlan) {
			r := p.Archives["A.zip"]
			r.CatalogError = "unread source"
			p.Archives["A.zip"] = r
		}},
		{"missing provenance", func(p *ArchivePlan) { r := p.Archives["A.zip"]; r.Sources = nil; p.Archives["A.zip"] = r }},
		{"unknown source checksum", func(p *ArchivePlan) {
			r := p.Archives["A.zip"]
			r.Sources[mediaPath] = archivePlanChecksum("other")
			p.Archives["A.zip"] = r
		}},
		{"unsafe source", func(p *ArchivePlan) {
			r := p.Archives["A.zip"]
			r.Sources["../bad.jpg"] = r.Media[0]
			p.Archives["A.zip"] = r
		}},
		{"duplicate media", func(p *ArchivePlan) {
			r := p.Archives["A.zip"]
			r.Media = append(r.Media, r.Media[0])
			p.Archives["A.zip"] = r
		}},
		{"renamed record", func(p *ArchivePlan) {
			p.Expected = append(p.Expected, "B.zip")
			p.Archives["B.zip"] = p.Archives["A.zip"]
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			i, state := archivePlanSession(t, &fakeClient{})
			require.NoError(t, i.ConfigureArchives([]string{"A.zip"}))
			a := archivePlanZIP(t, "A.zip", archivePlanEntry{mediaPath, "photo", false}, archivePlanEntry{mediaPath + ".json", sidecar, false})
			require.NoError(t, i.ImportArchive(context.Background(), a))
			test.change(i.state.Archives)
			require.NoError(t, i.save())
			require.NoError(t, i.Close())
			resumed, err := Open(state, scope(), &fakeClient{})
			require.NoError(t, err, "optional plan validation is enforced when planning APIs use it")
			defer resumed.Close()
			guide := archivePlanCheck(t, resumed, "blocked", "")
			require.Contains(t, guide.Reason, "Invalid persisted archive plan")
			require.Error(t, resumed.ConfigureArchives([]string{"A.zip"}))
			before, err := os.ReadFile(state)
			require.NoError(t, err)
			require.Error(t, resumed.ImportArchive(context.Background(), a))
			after, err := os.ReadFile(state)
			require.NoError(t, err)
			require.Equal(t, before, after)
		})
	}
}

func TestArchivePlanMissingOptionalStatePreservesV2AndV3CatalogFlows(t *testing.T) {
	for _, version := range []int{2, 3} {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			i, state := archivePlanSession(t, &fakeClient{})
			i.state.Version = version
			require.NoError(t, i.save())
			require.NoError(t, i.Close())
			resumed, err := Open(state, scope(), &fakeClient{})
			require.NoError(t, err)
			defer resumed.Close()
			require.Nil(t, resumed.state.Archives)
			require.NoError(t, resumed.ImportZIP(context.Background(), archivePlanZIP(t, "old.zip", archivePlanEntry{mediaPath, "photo", false})))
			require.Nil(t, resumed.state.Archives, "direct catalog-only API remains usable")
			archivePlanCheck(t, resumed, "inventory", "")
		})
	}
}

func TestArchivePlanFailedPersistenceDoesNotAdvanceOrUpload(t *testing.T) {
	client := &fakeClient{}
	i, state := archivePlanSession(t, client)
	i.filename = filepath.Join(t.TempDir(), "missing", "state.json")
	require.Error(t, i.ConfigureArchives([]string{"A.zip"}))
	require.Nil(t, i.state.Archives)
	i.filename = state
	require.NoError(t, i.ConfigureArchives([]string{"A.zip", "B.zip"}))
	i.filename = filepath.Join(t.TempDir(), "missing", "state.json")
	a := archivePlanZIP(t, "A.zip", archivePlanEntry{mediaPath, "photo", false}, archivePlanEntry{mediaPath + ".json", sidecar, false})
	require.Error(t, i.ImportArchive(context.Background(), a))
	archivePlanCheck(t, i, "keep/retry", "A.zip")
	require.Zero(t, client.uploads)
	i.filename = state
	require.NoError(t, i.ImportArchive(context.Background(), a))
	archivePlanCheck(t, i, "download", "B.zip")
}

func TestArchivePlanUnreadableSourceAndCanceledScanHoldUntilLocalRetry(t *testing.T) {
	for _, canceled := range []bool{false, true} {
		t.Run(fmt.Sprint(canceled), func(t *testing.T) {
			i, state := archivePlanSession(t, &fakeClient{})
			require.NoError(t, i.ConfigureArchives([]string{"A.zip", "B.zip"}))
			a := filepath.Join(t.TempDir(), "A.zip")
			ctx := context.Background()
			if canceled {
				a = archivePlanZIP(t, "A.zip", archivePlanEntry{mediaPath, "photo", false})
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			require.Error(t, i.ImportArchive(ctx, a))
			archivePlanCheck(t, i, "keep/retry", "A.zip")
			require.NoError(t, i.Close())
			resumed, err := Open(state, scope(), &fakeClient{})
			require.NoError(t, err)
			defer resumed.Close()
			archivePlanCheck(t, resumed, "keep/retry", "A.zip")
			if !canceled {
				a = archivePlanZIP(t, "A.zip", archivePlanEntry{mediaPath, "photo", false})
			}
			require.NoError(t, resumed.ImportArchive(context.Background(), a))
			archivePlanCheck(t, resumed, "download", "B.zip")
		})
	}
}

func TestArchivePlanCatalogReadErrorStillCallsImportZIP(t *testing.T) {
	i, _ := archivePlanSession(t, &fakeClient{})
	require.NoError(t, i.ConfigureArchives([]string{"A.zip"}))
	a := archivePlanZIP(t, "A.zip", archivePlanEntry{mediaPath, "photo", false}, archivePlanEntry{".json", "{}", false})
	require.ErrorContains(t, i.ImportArchive(context.Background(), a), "unsafe stem")
	require.Equal(t, archivePlanChecksum("photo"), i.state.Occurrences[mediaPath].Checksum)
	require.Len(t, i.state.Media, 1)
	require.True(t, slices.Contains(i.state.Archives.Archives["A.zip"].Media, archivePlanChecksum("photo")))
	archivePlanCheck(t, i, "keep/retry", "A.zip")
}
