package app

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/simulot/immich-go/internal/assettracker"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/fileprocessor"
	"github.com/spf13/cobra"
)

func TestApplicationFileProcessor(t *testing.T) {
	// TEST COMMENTS
	ctx := context.Background()
	cmd := &cobra.Command{}
	app := New(ctx, cmd)

	// Initially nil
	if app.FileProcessor() != nil {
		t.Error("FileProcessor should be nil initially")
	}

	// Create and set a file processor
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	tracker := assettracker.New()
	recorder := fileevent.NewRecorder(logger)
	processor := fileprocessor.New(tracker, recorder)

	app.SetFileProcessor(processor)

	// Should now be set
	if app.FileProcessor() == nil {
		t.Error("FileProcessor should not be nil after SetFileProcessor")
	}

	if app.FileProcessor() != processor {
		t.Error("FileProcessor should return the same instance")
	}

	if app.FileProcessor().Logger() != recorder {
		t.Error("FileProcessor.Logger should return the recorder")
	}
}
