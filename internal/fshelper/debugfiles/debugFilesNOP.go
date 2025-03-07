//go:build !debugfiles
// +build !debugfiles

package debugfiles

import (
	"io"
	"log/slog"
)

func EnableTrackFiles(log *slog.Logger) {
}

func TrackOpenFile(c io.Closer, name string) {
}

func TrackCloseFile(c io.Closer) {
}

func ReportTrackedFiles() {
}
