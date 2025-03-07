//go:build debugfiles
// +build debugfiles

package debugfiles

import (
	"io"
	"log/slog"
	"runtime"
	"strings"
	"sync/atomic"

	"github.com/simulot/immich-go/internal/gen/syncmap"
)

type _fileOpenTacker struct {
	name       string
	sourceFile string
	line       int
}

func (f _fileOpenTacker) LogValue() slog.Value {
	source := f.sourceFile
	if p := strings.Index(source, "immich-go/"); p > 0 {
		source = source[p:]
	}
	return slog.GroupValue(
		slog.String("name", f.name),
		slog.String("sourceFile", source),
		slog.Int("line", f.line),
	)
}

type _fileTracker struct {
	openFiles       *syncmap.SyncMap[io.Closer, _fileOpenTacker]
	countOpenFiles  int64
	countCloseFiles int64
	log             *slog.Logger
}

var fileTracker *_fileTracker

func EnableTrackFiles(log *slog.Logger) {
	fileTracker = &_fileTracker{
		openFiles: syncmap.New[io.Closer, _fileOpenTacker](),
		log:       log,
	}
	fileTracker.log.Debug("enable track files")
}

func TrackOpenFile(c io.Closer, name string) {
	if fileTracker != nil {
		atomic.AddInt64(&fileTracker.countOpenFiles, 1)
		t := _fileOpenTacker{
			name: name,
		}
		_, file, line, ok := runtime.Caller(1)
		if ok {
			t.sourceFile = file
			t.line = line
		}

		fileTracker.openFiles.Store(c, t)
		fileTracker.log.Debug("open file", "file", t)
	}
}

func TrackCloseFile(c io.Closer) {
	if fileTracker != nil {
		atomic.AddInt64(&fileTracker.countCloseFiles, 1)
		t, ok := fileTracker.openFiles.LoadAndDelete(c)
		if !ok {
			fileTracker.log.Error("file was not tracked", "file", c)
		}
		fileTracker.log.Debug("close file", "file", t)
	}
}

func ReportTrackedFiles() {
	if fileTracker != nil {
		fileTracker.log.Debug("report tracked files", "openFiles", fileTracker.countOpenFiles, "closeFiles", fileTracker.countCloseFiles)
		fileTracker.openFiles.Range(func(key io.Closer, value _fileOpenTacker) bool {
			fileTracker.log.Error("file was not closed", "file", value)
			return true
		})
	}
}
