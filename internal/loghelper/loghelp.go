package loghelper

import (
	"log/slog"
	"sync/atomic"
)

var globalLogger atomic.Pointer[slog.Logger]

func SetGlobalLogger(l *slog.Logger) {
	globalLogger.Store(l)
}

func Log(message string, values ...interface{}) {
	if l := globalLogger.Load(); l != nil {
		l.Info(message, values...)
	}
}

func Info(message string, values ...interface{}) {
	if l := globalLogger.Load(); l != nil {
		l.Info(message, values...)
	}
}

func Error(message string, values ...interface{}) {
	if l := globalLogger.Load(); l != nil {
		l.Error(message, values...)
	}
}

func Warn(message string, values ...interface{}) {
	if l := globalLogger.Load(); l != nil {
		l.Warn(message, values...)
	}
}

func Debug(message string, values ...interface{}) {
	if l := globalLogger.Load(); l != nil {
		l.Debug(message, values...)
	}
}
