package folder

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/simulot/immich-go/internal/fshelper/osfs"
)

// WriteAsset can be called from several goroutines for assets of different names: the folders the
// writer created and the archived files it hashed are recorded safely. Half of the assets were
// archived by an older release (no checksum in their sidecar): their files are hashed. Run it with
// -race.
func TestWriteAssetConcurrent(t *testing.T) {
	srcDir, dst := t.TempDir(), t.TempDir()
	w, err := NewLocalAssetWriter(osfs.DirFS(dst))
	if err != nil {
		t.Fatal(err)
	}
	const n = 12
	as := make([]*assets.Asset, 0, n)
	want := make([]string, 0, 2*n)
	holds := map[string]string{}
	for i := range n {
		name, body := fmt.Sprintf("IMG_%d.jpg", i), fmt.Sprintf("asset %d", i)
		writeSources(t, srcDir, map[string]string{name: body})
		a := &assets.Asset{
			File:        fshelper.FSName(os.DirFS(srcDir), name),
			FileSize:    len(body),
			CaptureDate: time.Date(2021, time.Month(1+i%3), 4, 12, 30, 0, 0, time.UTC),
		}
		a.SetNameInfo(assets.NameInfo{Base: name})
		a.FromApplication = &assets.Metadata{FileName: name}
		p := path.Join(w.pathOfAsset(a), name)
		if i%2 == 0 {
			writeArchived(t, dst, p, body, "")
		}
		as = append(as, a)
		want = append(want, p, p+".JSON")
		holds[p] = body
	}
	slices.Sort(want)

	errs := make([]error, n)
	var wg sync.WaitGroup
	for i, a := range as {
		wg.Go(func() { errs[i] = errors.Join(w.WriteAsset(context.Background(), a), a.Close()) })
	}
	wg.Wait()
	if err := errors.Join(errs...); err != nil {
		t.Fatal(err)
	}
	if got := listFiles(t, dst); !slices.Equal(got, want) {
		t.Fatalf("files %v, want %v", got, want)
	}
	checkHolds(t, dst, holds)
}
