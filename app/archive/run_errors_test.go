package archive

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/simulot/immich-go/adapters"
	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/spf13/cobra"
)

// runArchive runs the archive of an adapter into a new folder, with --on-errors set to onErrors,
// and returns the folder and the run's error. newAdapter gets the run's application.
func runArchive(t *testing.T, onErrors string, newAdapter func(*app.Application) adapters.Reader) (string, error) {
	t.Helper()
	ctx := context.Background()
	ac := &ArchiveCmd{ArchivePath: t.TempDir(), app: app.New(ctx, nil)}
	ac.app.Log().Logger = slog.New(slog.DiscardHandler)
	if err := ac.app.OnErrors.Set(onErrors); err != nil {
		t.Fatal(err)
	}
	cmd := &cobra.Command{}
	cmd.SetContext(ctx)
	return ac.ArchivePath, ac.Run(cmd, newAdapter(ac.app))
}

// failingGroup is 8 assets, IMG_0.JPG to IMG_7.JPG: the even ones are refused by the writer (their
// download is not the asset), the odd ones are archived.
func failingGroup(t *testing.T) *assets.Group {
	t.Helper()
	t.Setenv("IMMICHGO_TEMPDIR", t.TempDir())
	srcDir := t.TempDir()
	g := assets.NewGroup(assets.GroupByNone)
	for i := range 8 {
		name := fmt.Sprintf("IMG_%d.JPG", i)
		if err := os.WriteFile(filepath.Join(srcDir, name), []byte("the bytes of "+name), 0o644); err != nil {
			t.Fatal(err)
		}
		a := &assets.Asset{
			File:        fshelper.FSName(streamFS{os.DirFS(srcDir)}, name),
			CaptureDate: time.Date(2021, 7, 4, 12, 30, 0, 0, time.UTC),
		}
		a.SetNameInfo(assets.NameInfo{Base: name})
		if i%2 == 0 {
			a.Checksum = "c2hhMQ=="
			a.FromApplication = &assets.Metadata{FileName: name, Checksum: a.Checksum}
		}
		g.AddAsset(a)
	}
	return g
}

// archivedOdd returns which of IMG_1, IMG_3, IMG_5, IMG_7 are in the archive in dir.
func archivedOdd(dir string) []bool {
	got := make([]bool, 0, 4)
	for _, i := range []int{1, 3, 5, 7} {
		_, err := os.Stat(filepath.Join(dir, "2021", "2021-07", fmt.Sprintf("IMG_%d.JPG", i)))
		got = append(got, err == nil)
	}
	return got
}

// An asset the run could not archive makes the run fail, whatever --on-errors says: a scheduled
// archive with assets missing never exits 0. --on-errors says when the run stops: at the first
// failed asset (stop, the default), after N errors, or never (continue).
func TestRunFailedAssetsAndOnErrors(t *testing.T) {
	for _, tc := range []struct {
		onErrors string
		archived []bool // IMG_1, IMG_3, IMG_5, IMG_7
	}{
		{"stop", []bool{false, false, false, false}},
		{"2", []bool{true, true, false, false}},
		{"continue", []bool{true, true, true, true}},
	} {
		t.Run(tc.onErrors, func(t *testing.T) {
			g := failingGroup(t)
			dir, err := runArchive(t, tc.onErrors, func(*app.Application) adapters.Reader { return oneGroup{g} })
			if err == nil {
				t.Fatal("the run returned no error; want one, assets were not archived")
			}
			if got := archivedOdd(dir); fmt.Sprint(got) != fmt.Sprint(tc.archived) {
				t.Fatalf("IMG_1, IMG_3, IMG_5, IMG_7 archived: %v, want %v (run error: %v)", got, tc.archived, err)
			}
		})
	}
}

// errorBrowse reports an error and closes its channel, as an adapter that fails does.
type errorBrowse struct{ report func(context.Context) }

func (r errorBrowse) Browse(ctx context.Context) chan *assets.Group {
	c := make(chan *assets.Group)
	go func() {
		defer close(c)
		r.report(ctx)
	}()
	return c
}

// An error of the adapter makes the run fail: one it counts against --on-errors, as archive
// from-immich does when the server fails, and one it records, as archive from-folder does for a
// file it cannot read. Also with --on-errors=continue.
func TestRunAdapterErrorIsAnError(t *testing.T) {
	for name, report := range map[string]func(*app.Application, context.Context){
		"counted": func(a *app.Application, _ context.Context) {
			_ = a.ProcessError(errors.New("server error"))
		},
		"recorded": func(a *app.Application, ctx context.Context) {
			a.FileProcessor().RecordNonAsset(ctx, fshelper.FSName(os.DirFS("."), "IMG_1.JPG.xmp"), 0, fileevent.ErrorFileAccess, "error", "unreadable")
		},
	} {
		for _, onErrors := range []string{"stop", "continue"} {
			t.Run(name+" "+onErrors, func(t *testing.T) {
				_, err := runArchive(t, onErrors, func(a *app.Application) adapters.Reader {
					return errorBrowse{func(ctx context.Context) { report(a, ctx) }}
				})
				if err == nil {
					t.Fatal("the run returned no error; want one, the adapter failed")
				}
			})
		}
	}
}
