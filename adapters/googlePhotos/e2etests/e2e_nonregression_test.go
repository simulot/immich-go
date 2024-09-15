//go:build e2e
// +build e2e

package gp_test

import (
	"context"
	"io/fs"
	"log/slog"
	"os"
	"testing"

	"github.com/joho/godotenv"
	gp "github.com/simulot/immich-go/adapters/googlePhotos"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/metadata"
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
	if myEnv["IMMICH_TESTFILES"] == "" {
		t.Fatal("missing IMMICH_TESTFILES in .env file")
	}
}

type expectedCounts map[fileevent.Code]int64

func checkAgainstFileList(t *testing.T, fileList string, flags *gp.ImportFlags, expected expectedCounts, fsyss []fs.FS) {
	if flags.SupportedMedia == nil {
		flags.SupportedMedia = metadata.DefaultSupportedMedia
	}
	jnl, err := simulate_upload(fileList, flags, fsyss)
	if err != nil {
		t.Error(err)
		return
	}

	counts := jnl.GetCounts()
	for c := fileevent.Code(0); c < fileevent.MaxCode; c++ {
		if v, ok := expected[c]; ok {
			if counts[c] != v {
				t.Errorf("The counter[%s]==%d, expected %d", c.String(), counts[c], expected[c])
			}
		}
	}
}

// Simulate a takeout archive with the list of zipped files
func simulate_upload(testname string, flags *gp.ImportFlags, fsys []fs.FS) (*fileevent.Recorder, error) {
	ctx := context.Background()

	logFile, err := os.Create(testname + ".log")
	if err != nil {
		return nil, err
	}
	defer logFile.Close()

	log := slog.New(slog.NewTextHandler(logFile, nil))
	jnl := fileevent.NewRecorder(log, true)
	adapter, err := gp.NewTakeout(ctx, jnl, flags, fsys...)
	if err != nil {
		return nil, err
	}

	assets, err := adapter.Browse(ctx)
	if err != nil {
		return nil, err
	}
	for a := range assets {
		if a.Err != nil {
			return nil, a.Err
		}
	}

	csvFile, err := os.Create(testname + ".csv")
	if err != nil {
		return nil, err
	}
	defer csvFile.Close()
	err = jnl.WriteFileCounts(csvFile)
	if err != nil {
		return nil, err
	}

	dupFile, err := os.Create(testname + ".dup.csv")
	if err != nil {
		return nil, err
	}
	defer dupFile.Close()
	adapter.DebugDuplicates(dupFile)

	linkedFiles, err := os.Create(testname + ".linked.csv")
	if err != nil {
		return nil, err
	}
	defer linkedFiles.Close()
	adapter.DebugLinkedFiles(linkedFiles)

	trackedFiles, err := os.Create(testname + ".tracked.csv")
	if err != nil {
		return nil, err
	}
	defer trackedFiles.Close()
	adapter.DebugFileTracker(trackedFiles)

	return jnl, nil
}
