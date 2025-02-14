//go:build e2e
// +build e2e

package gp_test

import (
	"context"
	"io/fs"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	gp "github.com/simulot/immich-go/adapters/googlePhotos"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/filetypes"
)

var myEnv map[string]string

func initMyEnv(t *testing.T) {
	if len(myEnv) > 0 {
		return
	}
	var err error
	e, err := godotenv.Read("../../../e2e.env")
	if err != nil {
		t.Fatalf("cant initialize environment variables: %s", err)
	}
	myEnv = e
	if myEnv["IMMICHGO_TESTFILES"] == "" {
		t.Fatal("missing IMMICHGO_TESTFILES in .env file")
	}
}

type expectedCounts map[fileevent.Code]int64

func simulateAndCheck(t *testing.T, fileList string, flags *gp.ImportFlags, expected expectedCounts, fsyss []fs.FS) {
	if flags.SupportedMedia == nil {
		flags.SupportedMedia = filetypes.DefaultSupportedMedia
	}
	flags.InfoCollector = filenames.NewInfoCollector(time.Local, flags.SupportedMedia)
	jnl, err := simulate_upload(fileList, flags, fsyss)
	if err != nil {
		t.Error(err)
		return
	}

	counts := jnl.GetCounts()

	shouldUpload := counts[fileevent.DiscoveredImage] +
		counts[fileevent.DiscoveredVideo] -
		counts[fileevent.AnalysisLocalDuplicate] -
		counts[fileevent.DiscoveredDiscarded]
	if !flags.KeepJSONLess {
		shouldUpload -= counts[fileevent.AnalysisMissingAssociatedMetadata]
	}
	diff := shouldUpload - counts[fileevent.Uploaded]
	if diff != 0 {
		t.Errorf("The counter[Uploaded]==%d, expected %d, diff %d", counts[fileevent.Uploaded], shouldUpload, diff)
	}

	for c := fileevent.Code(0); c < fileevent.MaxCode; c++ {
		if v, ok := expected[c]; ok {
			if counts[c] != v {
				t.Errorf("The counter[%s]==%d, expected %d", c.String(), counts[c], expected[c])
			}
		}
	}
}

// Simulate takeout archive upload
func simulate_upload(testname string, flags *gp.ImportFlags, fsys []fs.FS) (*fileevent.Recorder, error) {
	ctx := context.Background()

	logFile, err := os.Create(testname + ".json")
	if err != nil {
		return nil, err
	}
	defer logFile.Close()

	log := slog.New(slog.NewJSONHandler(logFile, nil))
	jnl := fileevent.NewRecorder(log)
	adapter, err := gp.NewTakeout(ctx, jnl, flags, fsys...)
	if err != nil {
		return nil, err
	}

	assetsGroups := adapter.Browse(ctx)
	for g := range assetsGroups {
		for i, a := range g.Assets {
			jnl.Record(ctx, fileevent.Uploaded, a)
			if i >= 0 {
				for _, album := range a.Albums {
					jnl.Record(ctx, fileevent.UploadAddToAlbum, a, "album", album.Title)
				}
			}
		}
	}

	jnl.Report()

	trackerFile, err := os.Create(testname + ".tracker.csv")
	if err != nil {
		return nil, err
	}
	defer trackerFile.Close()
	adapter.DebugFileTracker(trackerFile)

	linkedFiles, err := os.Create(testname + ".linked.csv")
	if err != nil {
		return nil, err
	}
	defer linkedFiles.Close()

	return jnl, nil
}
