package incremental

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRepairedSidecarClearsOnlyItsOwnReadFailure(t *testing.T) {
	i, err := Open(filepath.Join(t.TempDir(), "state.json"), scope(), &fakeClient{})
	require.NoError(t, err)
	defer i.Close()
	regular, supplemental := mediaPath+".json", mediaPath+".supplemental-metadata.json"
	bad := archivePlanZIP(t, "bad.zip", archivePlanEntry{regular, sidecar, true}, archivePlanEntry{supplemental, sidecar, true})
	require.NoError(t, i.ImportZIP(context.Background(), bad))
	md := i.state.Metadata[mediaPath]
	require.Len(t, md.Problems, 2)
	const persistentReason = "unsafe sidecar title; item deferred"
	md.Problems = addString(md.Problems, regular+": "+persistentReason)
	i.state.Metadata[mediaPath] = md
	fixed := archive(t, map[string]string{regular: sidecar})
	require.NoError(t, i.ImportZIP(context.Background(), fixed))
	problems := i.state.Metadata[mediaPath].Problems
	require.Len(t, problems, 2, "repair must preserve another source's read failure and the non-transient problem")
	require.Contains(t, problems, regular+": "+persistentReason)
	for _, reason := range problems {
		require.False(t, strings.HasPrefix(reason, regular+": cannot read sidecar: "))
	}
	require.Contains(t, strings.Join(problems, "\n"), supplemental+": cannot read sidecar: ")
}

func TestRepairedAlbumSidecarDoesNotLeaveAnEmptyMediaRecord(t *testing.T) {
	i, err := Open(filepath.Join(t.TempDir(), "state.json"), scope(), &fakeClient{})
	require.NoError(t, err)
	defer i.Close()
	const name = "Takeout/Album/metadata.json"
	const body = `{"title":"Album"}`
	require.NoError(t, i.ImportZIP(context.Background(), archivePlanZIP(t, "bad.zip", archivePlanEntry{name, body, true})))
	require.Equal(t, 1, i.Summary().PendingMedia)
	require.NoError(t, i.ImportZIP(context.Background(), archive(t, map[string]string{name: body})))
	require.Zero(t, i.Summary().PendingMedia)
	require.NotContains(t, i.state.Metadata, sidecarTarget(name))
}
