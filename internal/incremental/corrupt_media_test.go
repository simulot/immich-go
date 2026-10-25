package incremental

import (
	"archive/zip"
	"context"
	"hash/crc32"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCorruptMediaPreservesHealthyProgressAndCanRetry(t *testing.T) {
	ctx := context.Background()
	state := filepath.Join(t.TempDir(), "state.json")
	client := &fakeClient{}
	filename := filepath.Join(t.TempDir(), "corrupt.zip")
	f, err := os.Create(filename)
	require.NoError(t, err)
	z := zip.NewWriter(f)
	for _, item := range []struct{ name, contents string }{
		{mediaPath, "healthy photo"}, {mediaPath + ".json", sidecar},
	} {
		w, createErr := z.Create(item.name)
		require.NoError(t, createErr)
		_, writeErr := w.Write([]byte(item.contents))
		require.NoError(t, writeErr)
	}
	const badPath = "Takeout/bad.jpg"
	badData := []byte("corrupt photo")
	raw, err := z.CreateRaw(&zip.FileHeader{
		Name: badPath, Method: zip.Store,
		CRC32:            crc32.ChecksumIEEE(badData) ^ 1,
		CompressedSize64: uint64(len(badData)), UncompressedSize64: uint64(len(badData)),
	})
	require.NoError(t, err)
	_, err = raw.Write(badData)
	require.NoError(t, err)
	require.NoError(t, z.Close())
	require.NoError(t, f.Close())

	i, err := Open(state, scope(), client)
	require.NoError(t, err)
	require.ErrorContains(t, i.ImportZIP(ctx, filename), "zip: checksum error")
	require.Equal(t, 1, client.uploads, "healthy media must proceed despite another item's bad CRC")
	require.Equal(t, 1, client.updates)
	require.Equal(t, 1, i.Summary().Uploaded)
	require.Contains(t, i.state.Problems[badPath][0], "checksum")
	require.NoError(t, i.Close())

	i, err = Open(state, scope(), client)
	require.NoError(t, err)
	defer i.Close()
	require.Equal(t, 1, i.Summary().Uploaded, "healthy progress must survive restart after the error")
	require.NotEmpty(t, i.state.Metadata[mediaPath].Names)
	require.ErrorContains(t, i.ImportZIP(ctx, filename), "zip: checksum error")
	require.Equal(t, 1, client.uploads, "retry must reuse the healthy receipt")

	repaired := archive(t, map[string]string{
		badPath: "repaired photo", badPath + ".json": `{"title":"bad.jpg","photoTakenTime":{"timestamp":"1577836800"}}`,
	})
	require.NoError(t, i.ImportZIP(ctx, repaired))
	require.Equal(t, 2, client.uploads)
	require.Equal(t, 2, i.Summary().Uploaded)
	for _, issue := range i.Issues() {
		require.False(t, issue.Source == badPath && strings.Contains(issue.Reason, "checksum"), "repaired media must clear its transient checksum problem")
	}
}
