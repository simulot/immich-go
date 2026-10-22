package archive

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/spf13/cobra"
)

// streamFS gives files that are not *os.File, as an Immich download is: Asset.OpenFile copies
// them into a temporary file of the cache reader.
type streamFS struct{ fs.FS }

type streamFile struct{ fs.File }

func (s streamFS) Open(name string) (fs.File, error) {
	f, err := s.FS.Open(name)
	if err != nil {
		return nil, err
	}
	return streamFile{f}, nil
}

type oneGroup struct{ g *assets.Group }

func (r oneGroup) Browse(context.Context) chan *assets.Group {
	c := make(chan *assets.Group, 1)
	c <- r.g
	close(c)
	return c
}

// An asset the writer refuses (here, a download that is not the asset) is closed all the same:
// the cache reader's temporary copy of the download is removed.
func TestRunClosesAssetOnWriteError(t *testing.T) {
	cache := t.TempDir()
	t.Setenv("IMMICHGO_TEMPDIR", cache)
	srcDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(srcDir, "a"), []byte("not the asset's bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &assets.Asset{
		File:        fshelper.FSName(streamFS{os.DirFS(srcDir)}, "a"),
		CaptureDate: time.Date(2021, 7, 4, 12, 30, 0, 0, time.UTC),
		Checksum:    "c2hhMQ==",
	}
	a.SetNameInfo(assets.NameInfo{Base: "IMG_1.JPG"})
	a.FromApplication = &assets.Metadata{FileName: "IMG_1.JPG", Checksum: a.Checksum}

	ctx := context.Background()
	archive := t.TempDir()
	ac := &ArchiveCmd{ArchivePath: archive, app: app.New(ctx, nil)}
	ac.app.Log().Logger = slog.New(slog.DiscardHandler)
	cmd := &cobra.Command{}
	cmd.SetContext(ctx)
	if err := ac.Run(cmd, oneGroup{assets.NewGroup(assets.GroupByNone, a)}); err != nil {
		t.Fatal(err)
	}

	err := filepath.WalkDir(archive, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			t.Errorf("the refused asset was written: %s", p)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	left, err := os.ReadDir(filepath.Join(cache, "immich-go", "temp"))
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 0 {
		t.Fatalf("cache files left after a refused asset: %v", left)
	}
}

// cancelledBrowse cancels the run as it starts, and closes its channel, as an adapter does when
// the run is cancelled.
type cancelledBrowse struct{ cancel context.CancelFunc }

func (r cancelledBrowse) Browse(context.Context) chan *assets.Group {
	r.cancel()
	c := make(chan *assets.Group)
	close(c)
	return c
}

// A cancelled run is an error, also when the adapter has closed its channel by the time Run reads
// it. Run's select then has two ready cases, ctx.Done() and the closed channel, and Go picks one
// at random: 100 runs take the closed channel at least once, but for a chance of 2^-100.
func TestRunCancelledIsAnError(t *testing.T) {
	archive := t.TempDir()
	for range 100 {
		ctx, cancel := context.WithCancel(context.Background())
		ac := &ArchiveCmd{ArchivePath: archive, app: app.New(ctx, nil)}
		ac.app.Log().Logger = slog.New(slog.DiscardHandler)
		cmd := &cobra.Command{}
		cmd.SetContext(ctx)
		if err := ac.Run(cmd, cancelledBrowse{cancel}); !errors.Is(err, context.Canceled) {
			t.Fatalf("a cancelled run returned %v, want %v", err, context.Canceled)
		}
	}
}
