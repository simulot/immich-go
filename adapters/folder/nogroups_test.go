package folder

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/simulot/immich-go/adapters"
	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/assettracker"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/fileprocessor"
	"github.com/spf13/cobra"
)

type collectRunner struct {
	groups []*assets.Group
}

func (c *collectRunner) Run(cmd *cobra.Command, r adapters.Reader) error {
	for g := range r.Browse(cmd.Context()) {
		c.groups = append(c.groups, g)
	}
	return nil
}

// from-folder groups nothing: files that immich-go 0.32.0 grouped by name radical and time (a
// RAW and its JPG, two JPGs of one radical, a burst) come out one per group, GroupByNone, so the
// upload neither stacks them nor drops any. The one stack is the one a sidecar records.
func TestFromFolderMakesNoGroups(t *testing.T) {
	dir := t.TempDir()
	t0 := time.Date(2005, 1, 3, 10, 0, 0, 0, time.Local)
	files := []string{
		"IMG_0003.JPG", "IMG_0003.CR2", // RAW + JPG
		"100_0074.JPG", "100_0074.jpeg", // same radical, same second
		"IMG_20231014_183246_BURST001_COVER.jpg", "IMG_20231014_183246_BURST002.jpg", // a named burst
		"IMG_0005.JPG", "IMG_0006.JPG",
	}
	for _, f := range files {
		p := filepath.Join(dir, f)
		if err := os.WriteFile(p, []byte("not really an image: "+f), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, t0, t0); err != nil {
			t.Fatal(err)
		}
	}
	side := `{
  "software": "immich-go version:0.32.0,  commit:x, date:y",
  "fileName": "IMG_0005.JPG",
  "dateTaken": "2005-01-03T10:00:00-07:00",
  "stack": {
    "id": "cccccccc-0000-4000-8000-000000000001",
    "primary": true,
    "assetCount": 2
  }
}
`
	if err := os.WriteFile(filepath.Join(dir, "IMG_0005.JPG.JSON"), []byte(side), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	a := app.New(ctx, nil)
	a.SetLog(&app.Log{Logger: slog.New(slog.DiscardHandler)})
	a.SetFileProcessor(fileprocessor.New(assettracker.New(), fileevent.NewRecorder(slog.New(slog.DiscardHandler))))
	a.ConcurrentTask = 2
	cmd := &cobra.Command{Use: "from-folder"}
	cmd.SetContext(ctx)
	ifc := &ImportFolderCmd{}
	ifc.RegisterFlags(cmd.Flags(), cmd)
	r := &collectRunner{}
	if err := ifc.run(cmd, []string{dir}, a, r); err != nil {
		t.Fatal(err)
	}

	var names []string
	for _, g := range r.groups {
		if g.Grouping != assets.GroupByNone || len(g.Assets) != 1 {
			var in []string
			for _, x := range g.Assets {
				in = append(in, x.File.Name())
			}
			t.Errorf("a group (%v) of %v: from-folder must not group", g.Grouping, in)
		}
		for _, x := range g.Assets {
			names = append(names, x.File.Name())
			st := (*assets.Stack)(nil)
			if x.FromApplication != nil {
				st = x.FromApplication.Stack
			}
			if (x.File.Name() == "IMG_0005.JPG") != (st != nil) {
				t.Errorf("%s: stack %+v", x.File.Name(), st)
			}
			if st != nil && (st.ID != "cccccccc-0000-4000-8000-000000000001" || !st.Primary || st.AssetCount != 2) {
				t.Errorf("%s: stack %+v", x.File.Name(), st)
			}
		}
	}
	sort.Strings(names)
	want := append([]string(nil), files...)
	sort.Strings(want)
	if len(names) != len(want) {
		t.Fatalf("assets %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("assets %v, want %v", names, want)
		}
	}
}
