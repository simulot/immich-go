package incremental

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/stretchr/testify/require"
)

type fakeClient struct {
	uploads, updates                           int
	failUpdate, loseReceipt, loseUpdateReceipt bool
	failBaseline                               bool
	asset                                      immich.Asset
	byBytes                                    map[string]string
	others                                     map[string]immich.Asset
	names                                      []string
	patches                                    []immich.UpdAssetField
}

func (f *fakeClient) AssetUpload(_ context.Context, a *assets.Asset) (immich.AssetResponse, error) {
	r, err := a.File.Open()
	if err != nil {
		return immich.AssetResponse{}, err
	}
	defer r.Close()
	data, err := io.ReadAll(r)
	if err != nil {
		return immich.AssetResponse{}, err
	}
	f.uploads++
	f.names = append(f.names, a.OriginalFileName)
	if f.byBytes == nil {
		f.byBytes = map[string]string{}
		f.others = map[string]immich.Asset{}
	}
	id, exists := f.byBytes[string(data)]
	status := immich.UploadDuplicate
	if !exists {
		status = immich.UploadCreated
		id = fmt.Sprintf("asset-%d", len(f.byBytes)+1)
		f.byBytes[string(data)] = id
		if id == "asset-1" {
			f.asset.OriginalFileName = a.OriginalFileName
		} else {
			f.others[id] = immich.Asset{ID: id, OriginalFileName: a.OriginalFileName}
		}
	}
	if f.loseReceipt {
		f.loseReceipt = false
		return immich.AssetResponse{}, errors.New("lost upload acknowledgement")
	}
	return immich.AssetResponse{ID: id, Status: status}, nil
}

func (f *fakeClient) GetAssetInfo(_ context.Context, id string) (*immich.Asset, error) {
	if f.failBaseline {
		return nil, errors.New("baseline unavailable")
	}
	a := f.asset
	if id != "asset-1" {
		a = f.others[id]
	}
	return &a, nil
}

func (f *fakeClient) UpdateAsset(_ context.Context, id string, p immich.UpdAssetField) (*immich.Asset, error) {
	f.updates++
	f.patches = append(f.patches, p)
	if f.failUpdate {
		return nil, errors.New("temporary failure")
	}
	if p.DateTimeOriginal.IsZero() {
		return nil, errors.New("zero date would corrupt metadata")
	}
	a := f.asset
	if id != "asset-1" {
		a = f.others[id]
	}
	a.ExifInfo.DateTimeOriginal.Time = p.DateTimeOriginal
	if p.Description != "" {
		a.ExifInfo.Description = p.Description
	}
	if id == "asset-1" {
		f.asset = a
	} else {
		f.others[id] = a
	}
	if f.loseUpdateReceipt {
		f.loseUpdateReceipt = false
		return nil, errors.New("lost patch acknowledgement")
	}
	return &a, nil
}

func archive(t *testing.T, files map[string]string) string {
	t.Helper()
	name := filepath.Join(t.TempDir(), "part.zip")
	f, err := os.Create(name)
	require.NoError(t, err)
	z := zip.NewWriter(f)
	for name, data := range files {
		w, err := z.Create(name)
		require.NoError(t, err)
		_, err = w.Write([]byte(data))
		require.NoError(t, err)
	}
	require.NoError(t, z.Close())
	require.NoError(t, f.Close())
	return name
}

const (
	mediaPath = "Takeout/Google Photos/Photos from 2020/photo.jpg"
	sidecar   = `{"title":"photo.jpg","description":"from Google","photoTakenTime":{"timestamp":"1577836800"}}`
)

func scope() Scope {
	return Scope{Server: "https://immich.example", UserID: "user-1", ExportID: "export-1"}
}

func TestSeparateZIPsRestartAndIdempotence(t *testing.T) {
	ctx := context.Background()
	state := filepath.Join(t.TempDir(), "state.json")
	client := &fakeClient{}
	first := archive(t, map[string]string{mediaPath: "photo bytes"})
	second := archive(t, map[string]string{mediaPath + ".json": sidecar})
	i, err := Open(state, scope(), client)
	require.NoError(t, err)
	require.NoError(t, i.ImportZIP(ctx, first))
	require.Zero(t, client.uploads)
	require.Zero(t, client.updates)
	require.NoError(t, i.Close())
	i, err = Open(state, scope(), client)
	require.NoError(t, err)
	defer i.Close()
	require.NoError(t, i.ImportZIP(ctx, second))
	require.Zero(t, client.uploads)
	require.NoError(t, i.ImportZIP(ctx, first))
	require.Equal(t, 1, client.updates)
	require.Equal(t, "from Google", client.asset.ExifInfo.Description)
	require.True(t, client.asset.ExifInfo.DateTimeOriginal.Equal(time.Unix(1577836800, 0)))
	require.NoError(t, i.ImportZIP(ctx, first))
	require.NoError(t, i.ImportZIP(ctx, second))
	require.Equal(t, 1, client.uploads)
	require.Equal(t, 1, client.updates)
}

func TestFailedPatchRetriedWithoutZIP(t *testing.T) {
	ctx := context.Background()
	state := filepath.Join(t.TempDir(), "state.json")
	client := &fakeClient{failUpdate: true}
	i, err := Open(state, scope(), client)
	require.NoError(t, err)
	require.ErrorContains(t, i.ImportZIP(ctx, archive(t, map[string]string{mediaPath: "photo", mediaPath + ".json": sidecar})), "pending metadata")
	require.NoError(t, i.Close())
	client.failUpdate = false
	i, err = Open(state, scope(), client)
	require.NoError(t, err)
	defer i.Close()
	require.NoError(t, i.Reconcile(ctx))
	require.Equal(t, 1, client.uploads)
	require.Equal(t, 2, client.updates)
}

func TestMetadataBeforeMedia(t *testing.T) {
	ctx := context.Background()
	client := &fakeClient{}
	i, err := Open(filepath.Join(t.TempDir(), "state.json"), scope(), client)
	require.NoError(t, err)
	defer i.Close()
	require.NoError(t, i.ImportZIP(ctx, archive(t, map[string]string{mediaPath + ".supplemental-metadata.json": sidecar})))
	require.Zero(t, client.uploads)
	require.NoError(t, i.ImportZIP(ctx, archive(t, map[string]string{mediaPath: "photo"})))
	require.Equal(t, 1, client.updates)
}

func TestLostUploadReceiptNeverEnrichesDuplicate(t *testing.T) {
	ctx := context.Background()
	client := &fakeClient{loseReceipt: true}
	i, err := Open(filepath.Join(t.TempDir(), "state.json"), scope(), client)
	require.NoError(t, err)
	defer i.Close()
	zip := archive(t, map[string]string{mediaPath: "photo", mediaPath + ".json": sidecar})
	require.ErrorContains(t, i.ImportZIP(ctx, zip), "unknown outcomes")
	require.NoError(t, i.ImportZIP(ctx, zip))
	require.Zero(t, client.updates)
	require.False(t, i.state.Media[i.state.Occurrences[mediaPath].Checksum].Owned)
}

func TestScopeAndConcurrentLock(t *testing.T) {
	state := filepath.Join(t.TempDir(), "state.json")
	i, err := Open(state, scope(), &fakeClient{})
	require.NoError(t, err)
	_, err = Open(state, scope(), &fakeClient{})
	require.ErrorContains(t, err, "locked")
	require.NoError(t, i.Close())
	other := scope()
	other.ExportID = "different"
	_, err = Open(state, other, &fakeClient{})
	require.ErrorContains(t, err, "scope")
}

func TestConflictsDoNotOverwrite(t *testing.T) {
	ctx := context.Background()
	client := &fakeClient{}
	i, err := Open(filepath.Join(t.TempDir(), "state.json"), scope(), client)
	require.NoError(t, err)
	defer i.Close()
	require.NoError(t, i.ImportZIP(ctx, archive(t, map[string]string{mediaPath: "photo", mediaPath + ".json": `{"title":"photo.jpg"}`})))
	client.asset.ExifInfo.Description = "user edit"
	require.ErrorContains(t, i.ImportZIP(ctx, archive(t, map[string]string{mediaPath + ".json": sidecar})), "refusing to overwrite")
	require.Zero(t, client.updates)
	require.NoError(t, i.ImportZIP(ctx, archive(t, map[string]string{mediaPath: "different bytes"})))
	require.Contains(t, fmt.Sprint(i.Issues()), "ambiguous original path")
}

func TestUnsupportedMatchingIsReported(t *testing.T) {
	i, err := Open(filepath.Join(t.TempDir(), "state.json"), scope(), &fakeClient{})
	require.NoError(t, err)
	defer i.Close()
	require.NoError(t, i.ImportZIP(context.Background(), archive(t, map[string]string{"Takeout/photo.jp.json": sidecar})))
	require.Contains(t, fmt.Sprint(i.Issues()), "unsupported duplicate/truncated")
}

func TestLostPatchReceiptRetryIsIdempotent(t *testing.T) {
	ctx := context.Background()
	client := &fakeClient{loseUpdateReceipt: true}
	state := filepath.Join(t.TempDir(), "state.json")
	i, err := Open(state, scope(), client)
	require.NoError(t, err)
	require.ErrorContains(t, i.ImportZIP(ctx, archive(t, map[string]string{mediaPath: "photo", mediaPath + ".json": sidecar})), "lost patch acknowledgement")
	require.NoError(t, i.Close())
	i, err = Open(state, scope(), client)
	require.NoError(t, err)
	defer i.Close()
	require.NoError(t, i.Reconcile(ctx))
	require.Equal(t, 2, client.updates)
	require.Equal(t, 1, client.uploads)
}

func TestDirectoryEntriesAndInvalidTimestamp(t *testing.T) {
	client := &fakeClient{}
	i, err := Open(filepath.Join(t.TempDir(), "state.json"), scope(), client)
	require.NoError(t, err)
	defer i.Close()
	require.NoError(t, i.ImportZIP(context.Background(), archive(t, map[string]string{"Takeout/": "", mediaPath: "photo"})))
	require.NoError(t, i.ImportZIP(context.Background(), archive(t, map[string]string{mediaPath + ".json": `{"title":"photo.jpg","photoTakenTime":{"timestamp":"not-a-time"}}`})))
	require.Contains(t, fmt.Sprint(i.Issues()), "invalid capture timestamp")
	require.Zero(t, client.uploads)
}

func TestAppliedMetadataDriftIsReported(t *testing.T) {
	ctx := context.Background()
	client := &fakeClient{}
	i, err := Open(filepath.Join(t.TempDir(), "state.json"), scope(), client)
	require.NoError(t, err)
	defer i.Close()
	require.NoError(t, i.ImportZIP(ctx, archive(t, map[string]string{mediaPath: "photo", mediaPath + ".json": sidecar})))
	client.asset.ExifInfo.DateTimeOriginal.Time = time.Unix(1577836801, 0)
	require.ErrorContains(t, i.Reconcile(ctx), "metadata drifted after enrichment")
	require.Equal(t, 1, client.updates)
}

func TestFailedBaselineReceiptRemainsUnowned(t *testing.T) {
	ctx := context.Background()
	client := &fakeClient{failBaseline: true}
	state := filepath.Join(t.TempDir(), "state.json")
	first := archive(t, map[string]string{mediaPath: "photo", mediaPath + ".json": `{"title":"photo.jpg"}`})
	i, err := Open(state, scope(), client)
	require.NoError(t, err)
	require.ErrorContains(t, i.ImportZIP(ctx, first), "receipt retained as unowned")
	require.NoError(t, i.Close())
	client.failBaseline = false
	i, err = Open(state, scope(), client)
	require.NoError(t, err)
	defer i.Close()
	require.NoError(t, i.ImportZIP(ctx, archive(t, map[string]string{mediaPath + ".json": sidecar})))
	require.NoError(t, i.ImportZIP(ctx, first))
	require.Zero(t, client.updates)
	require.Equal(t, 1, client.uploads)
	require.False(t, i.state.Media[i.state.Occurrences[mediaPath].Checksum].Owned)
}
