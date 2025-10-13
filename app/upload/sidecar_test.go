package upload

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"slices"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/exif/sidecars/xmpsidecar"
	"github.com/simulot/immich-go/internal/fshelper"
)

// Inspect the sidecar at the upload boundary: CLI tags must already be present
// when the initial upload is sent, including when replacing an existing asset.
type sidecarUploadClient struct {
	immich.ImmichInterface
	sidecar []byte
}

func (c *sidecarUploadClient) AssetUpload(_ context.Context, a *assets.Asset) (immich.AssetResponse, error) {
	var err error
	c.sidecar, err = fs.ReadFile(a.FromSideCar.File.FS(), a.FromSideCar.File.Name())
	return immich.AssetResponse{ID: "uploaded", Status: immich.UploadDuplicate}, err
}

func TestSidecarTagsAreReadyBeforeUpload(t *testing.T) {
	for _, replace := range []bool{false, true} {
		t.Run(fmt.Sprintf("replace=%t", replace), func(t *testing.T) {
			uc, _ := newTestUpCmd(t)
			uc.app.Log().SetLogWriter(io.Discard)
			uc.TagViaSidecar = true
			uc.Tags = []string{"cli/tag"}
			uc.SessionTag = true
			uc.session = "session/tag"
			t.Cleanup(func() { _ = uc.finishing(context.Background()) })
			client := &sidecarUploadClient{}
			uc.client.Immich = client
			a := &assets.Asset{
				OriginalFileName: "photo.jpg",
				File: fshelper.FSName(fstest.MapFS{
					"photo.jpg": &fstest.MapFile{Data: []byte("photo")},
				}, "photo.jpg"),
				Description: "Keep this description",
				CaptureDate: time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC),
				Latitude:    42, Longitude: 12, Rating: 4, Favorite: true,
				Tags: []assets.Tag{{Name: "tag", Value: "folder/tag"}},
			}
			if _, err := a.GetChecksum(); err != nil {
				t.Fatal(err)
			}
			before := *a
			var err error
			if replace {
				err = uc.replaceAsset(context.Background(), a, &assets.Asset{ID: "old"})
			} else {
				err = uc.uploadAsset(context.Background(), a)
			}
			if err != nil {
				t.Fatal(err)
			}
			assertSidecarTags(t, client.sidecar, []string{"cli/tag", "folder/tag", "session/tag"})
			if a.Description != before.Description || a.CaptureDate != before.CaptureDate ||
				a.Latitude != before.Latitude || a.Longitude != before.Longitude ||
				a.Rating != before.Rating || a.Favorite != before.Favorite {
				t.Fatal("generating a tags-only sidecar changed the asset metadata")
			}
			// A nil tag cache makes any accidental API tagging fail immediately.
			uc.tagsCache.Close()
			uc.tagsCache = nil
			uc.manageAssetTags(context.Background(), a)
		})
	}
}

func TestSidecarMergesExistingTagsAndPreservesSource(t *testing.T) {
	uc, _ := newTestUpCmd(t)
	uc.app.Log().SetLogWriter(io.Discard)
	uc.TagViaSidecar = true
	t.Cleanup(func() { _ = uc.finishing(context.Background()) })
	source, err := os.ReadFile("../../internal/exif/sidecars/xmpsidecar/DATA/159d9172-2a1e-4d95-aef1-b5133549927b.jpg.xmp")
	if err != nil {
		t.Fatal(err)
	}
	f := fstest.MapFS{"photo.xmp": &fstest.MapFile{Data: source}}
	md := &assets.Metadata{File: fshelper.FSName(f, "photo.xmp")}
	if err := xmpsidecar.ReadXMP(bytes.NewReader(source), md); err != nil {
		t.Fatal(err)
	}
	a := &assets.Asset{OriginalFileName: "photo.jpg", FromSideCar: md}
	a.AddTag("new/tag")
	if err := uc.prepareTagsSidecar(context.Background(), a); err != nil {
		t.Fatal(err)
	}
	updated, err := fs.ReadFile(a.FromSideCar.File.FS(), a.FromSideCar.File.Name())
	if err != nil {
		t.Fatal(err)
	}
	assertSidecarTags(t, updated, []string{"activities/outdoors", "new/tag"})
	if a.FromSideCar.Rating != md.Rating || a.FromSideCar.Description != md.Description ||
		!a.FromSideCar.DateTaken.Equal(md.DateTaken) {
		t.Fatal("existing sidecar metadata changed")
	}
	if !bytes.Equal(source, f["photo.xmp"].Data) {
		t.Fatal("source sidecar was modified")
	}
}

func TestConcurrentSidecarsWithIdenticalFilenames(t *testing.T) {
	uc, _ := newTestUpCmd(t)
	uc.app.Log().SetLogWriter(io.Discard)
	uc.TagViaSidecar = true
	t.Cleanup(func() { _ = uc.finishing(context.Background()) })
	const count = 16
	all := make([]*assets.Asset, count)
	errs := make([]error, count)
	var wg sync.WaitGroup
	for i := range all {
		all[i] = &assets.Asset{OriginalFileName: "photo.jpg"}
		all[i].AddTag(fmt.Sprintf("folder/%d", i))
		wg.Go(func() { errs[i] = uc.prepareTagsSidecar(context.Background(), all[i]) })
	}
	wg.Wait()
	files := map[string]bool{}
	for i, a := range all {
		if errs[i] != nil {
			t.Fatal(errs[i])
		}
		name := a.FromSideCar.File.Name()
		if files[name] {
			t.Fatalf("assets share temporary file %q", name)
		}
		files[name] = true
		content, err := fs.ReadFile(a.FromSideCar.File.FS(), name)
		if err != nil {
			t.Fatal(err)
		}
		assertSidecarTags(t, content, []string{fmt.Sprintf("folder/%d", i)})
	}
	dir := uc.tagSidecarDir
	if err := uc.finishing(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("temporary directory still exists: %v", err)
	}
}

func assertSidecarTags(t *testing.T, content []byte, want []string) {
	t.Helper()
	md := &assets.Metadata{}
	if err := xmpsidecar.ReadXMP(bytes.NewReader(content), md); err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(md.Tags))
	for _, tag := range md.Tags {
		got = append(got, tag.Value)
	}
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Fatalf("sidecar tags = %v, want %v", got, want)
	}
}
