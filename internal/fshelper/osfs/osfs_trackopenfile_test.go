//go:build debugfiles

package osfs

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/simulot/immich-go/internal/fshelper/debugfiles"
)

// Issue #1489: dirFS.Open and dirFS.OpenFile called
// debugfiles.TrackOpenFile inside `if err != nil`, so a failed open
// was tracked with a nil file and a successful open was never tracked.
func TestTrackOpenFileOnlyOnSuccess(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	debugfiles.EnableTrackFiles(log)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "present.txt"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	fsys := DirFS(dir)

	// A successful Open must be tracked.
	f, err := fsys.Open("present.txt")
	if err != nil {
		t.Fatalf("Open(present.txt): %v", err)
	}
	defer f.Close()

	// A failed Open must not be tracked.
	if _, err := fsys.Open("missing.txt"); err == nil {
		t.Fatal("Open(missing.txt): expected an error, got nil")
	}

	// A successful OpenFile must be tracked.
	wf, err := fsys.(fshelper.FSCanWrite).OpenFile("present.txt", os.O_RDONLY, 0)
	if err != nil {
		t.Fatalf("OpenFile(present.txt): %v", err)
	}
	defer wf.Close()

	// A failed OpenFile must not be tracked (no O_CREATE, missing file).
	if _, err := fsys.(fshelper.FSCanWrite).OpenFile("missing.txt", os.O_RDONLY, 0); err == nil {
		t.Fatal("OpenFile(missing.txt): expected an error, got nil")
	}

	buf.Reset()
	debugfiles.ReportTrackedFiles()
	out := buf.String()
	if !strings.Contains(out, "openFiles=2") {
		t.Fatalf("report does not count exactly the 2 successful opens: %q", out)
	}
	if n := strings.Count(out, "file was not closed"); n != 2 {
		t.Fatalf("report lists %d unclosed files, want 2 (the successful opens): %q", n, out)
	}
	if strings.Contains(out, "missing.txt") {
		t.Fatalf("a failed open was tracked: %q", out)
	}
}
