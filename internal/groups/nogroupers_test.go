package groups_test

import (
	"context"
	"testing"
	"time"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/filetypes"
	"github.com/simulot/immich-go/internal/groups"
)

// With no grouper, every asset comes out alone, as GroupByNone, in order, and the pipeline ends.
func TestPipeGrouperWithoutGroupers(t *testing.T) {
	ic := filenames.NewInfoCollector(time.UTC, filetypes.DefaultSupportedMedia)
	t0 := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	names := []string{"IMG_003.jpg", "IMG_003.raw", "IMG_004.heic", "IMG_004.jpg", "100_0074.JPG", "100_0074.jpeg"}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	in := make(chan *assets.Asset)
	go func() {
		defer close(in)
		for _, n := range names {
			in <- mockAsset(ic, n, t0)
		}
	}()
	gs := groups.NewGrouperPipeline(ctx).PipeGrouper(ctx, in)
	var got []string
	for {
		select {
		case g, ok := <-gs:
			if !ok {
				if len(got) != len(names) {
					t.Fatalf("got %v, want %v", got, names)
				}
				for i := range names {
					if got[i] != names[i] {
						t.Fatalf("got %v, want %v", got, names)
					}
				}
				return
			}
			if g.Grouping != assets.GroupByNone || len(g.Assets) != 1 {
				t.Fatalf("group %v of %d assets, want each asset alone", g.Grouping, len(g.Assets))
			}
			got = append(got, g.Assets[0].File.Name())
		case <-ctx.Done():
			t.Fatalf("the pipeline without groupers did not finish (got %v)", got)
		}
	}
}
