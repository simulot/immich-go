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
}

func TestApplicationJnlBackwardCompatibility(t *testing.T) {
	ctx := context.Background()
	cmd := &cobra.Command{}
	app := New(ctx, cmd)

	// Jnl should still work (backward compatibility)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	recorder := fileevent.NewRecorder(logger)

	app.SetJnl(recorder)

	if app.Jnl() == nil {
		t.Error("Jnl should not be nil after SetJnl")
	}

	if app.Jnl() != recorder {
		t.Error("Jnl should return the same instance")
	}
}

func TestApplicationBothJnlAndProcessor(t *testing.T) {
	ctx := context.Background()
	cmd := &cobra.Command{}
	app := New(ctx, cmd)

	// Test that both Jnl and FileProcessor can coexist
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	// Set Jnl (old way)
	jnlRecorder := fileevent.NewRecorder(logger)
	app.SetJnl(jnlRecorder)

	// Set FileProcessor (new way)
	tracker := assettracker.New()
	processorRecorder := fileevent.NewRecorder(logger)
	processor := fileprocessor.New(tracker, processorRecorder)
	app.SetFileProcessor(processor)

	// Both should be accessible
	if app.Jnl() == nil {
		t.Error("Jnl should still be accessible")
	}
	if app.FileProcessor() == nil {
		t.Error("FileProcessor should be accessible")
	}

	// They should be different instances (for now, during migration)
	if app.Jnl() == app.FileProcessor().Logger() {
		t.Log("Note: Jnl and FileProcessor.Logger are the same instance (this is OK during migration)")
	}
}
