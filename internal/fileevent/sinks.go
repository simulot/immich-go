package fileevent

import (
	"context"
	"log/slog"
)

// SlogSink sends events to a slog.Logger for file logging.
type SlogSink struct {
	logger *slog.Logger
}

// NewSlogSink creates a new sink that logs to slog.
func NewSlogSink(logger *slog.Logger) *SlogSink {
	return &SlogSink{logger: logger}
}

// HandleEvent logs the event using slog.
func (s *SlogSink) HandleEvent(ctx context.Context, code Code, file string, size int64, args map[string]any) {
	if s.logger == nil {
		return
	}

	level := _logLevels[code]

	// Convert map to variadic args for slog
	logArgs := []any{}
	if file != "" {
		logArgs = append(logArgs, "file", file)
	}

	// Check for error/warning overrides and build args
	for k, v := range args {
		logArgs = append(logArgs, k, v)
		if k == "error" {
			level = slog.LevelError
		}
		if k == "warning" {
			level = slog.LevelWarn
		}
	}

	s.logger.Log(ctx, level, code.String(), logArgs...)
}
