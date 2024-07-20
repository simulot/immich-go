package upload

import (
	"context"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/simulot/immich-go/cmd"
	"github.com/simulot/immich-go/internal/fakefs"
)

// Simulate a takeout archive with the list of zipped files
func simulate_upload(t *testing.T, zipList string, dateFormat string) {
	ic := &icCatchUploadsAssets{
		albums: map[string][]string{},
	}
	ctx := context.Background()

	//	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	serv := cmd.SharedFlags{
		Immich:   ic,
		LogLevel: "INFO",
		// Jnl:    fileevent.NewRecorder(log, false),
		// Log:    log,
	}

	fsOpener := func() ([]fs.FS, error) {
		return fakefs.ScanFileList(zipList, dateFormat)
	}
	args := []string{"-google-photos", "-no-ui", "-debug-counters", "-log-file=" + filepath.Dir(zipList) + "/debug.log"}

	app, err := newCommand(ctx, &serv, args, fsOpener)
	if err != nil {
		t.Errorf("can't instantiate the UploadCmd: %s", err)
		return
	}

	err = app.run(ctx)
	if err != nil {
		t.Errorf("can't run the UploadCmd: %s", err)
		return
	}
}

func TestPixilTakeOut(t *testing.T) {
	initMyEnv(t)

	simulate_upload(t, myEnv["IMMICH_TESTFILES"]+"/Counters/pixil/list.lst", "01-02-2006 15:04")
}
