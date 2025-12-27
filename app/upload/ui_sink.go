package upload

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/ui/core/messages"
	"github.com/simulot/immich-go/internal/ui/core/state"
)

// uiSink sends file events to the UI event pipeline as log lines.
type uiSink struct {
	ctx       context.Context
	publisher messages.Publisher
}

// newUISink creates a new UI sink for file events.
func newUISink(ctx context.Context, publisher messages.Publisher) *uiSink {
	return &uiSink{
		ctx:       ctx,
		publisher: publisher,
	}
}

// HandleEvent receives structured file events and sends them to the UI as log lines.
func (s *uiSink) HandleEvent(ctx context.Context, code fileevent.Code, file string, size int64, args map[string]any) {
	// Determine log level
	level := "INF"
	logLevel := fileevent.GetLogLevel(code)

	// Check for error/warning overrides in args
	if _, hasError := args["error"]; hasError {
		level = "ERR"
	} else if _, hasWarning := args["warning"]; hasWarning {
		level = "WRN"
	} else {
		// Map slog levels to UI levels
		switch logLevel {
		case slog.LevelError:
			level = "ERR"
		case slog.LevelWarn:
			level = "WRN"
		case slog.LevelInfo:
			level = "INF"
		case slog.LevelDebug:
			level = "DBG"
		}
	}

	// Format message similar to slog output
	msg := code.String()
	if file != "" {
		msg = fmt.Sprintf("%s file=%s", msg, file)
	}

	details := map[string]string{
		"event_code":    code.String(),
		"event_code_id": fmt.Sprintf("%d", int(code)),
	}
	if file != "" {
		details["file"] = file
	}
	if size > 0 {
		details["size_bytes"] = fmt.Sprintf("%d", size)
	}

	// Add args to message and details
	for k, v := range args {
		if k == "error" || k == "warning" {
			continue // Already handled in level determination
		}
		sv := fmt.Sprintf("%v", v)
		msg = fmt.Sprintf("%s %s=%s", msg, k, sv)
		details[k] = sv
	}

	// If there's an error, append it
	if errMsg, hasError := args["error"]; hasError {
		msg = fmt.Sprintf("%s error=%v", msg, errMsg)
		details["error"] = fmt.Sprintf("%v", errMsg)
	}

	s.publisher.AppendLog(s.ctx, state.LogEvent{
		Level:     level,
		Message:   msg,
		Timestamp: time.Now(),
		Details:   details,
	})
}
