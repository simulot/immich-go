package loghelper

import "log/slog"

var globalLogger *slog.Logger

func SetGlobalLogger(l *slog.Logger) {
	globalLogger = l
}

func Log(message string, values ...interface{}) {
	if globalLogger != nil {
		globalLogger.Info(message, values...)
	}
}

func Info(message string, values ...interface{}) {
	if globalLogger != nil {
		globalLogger.Info(message, values...)
	}
}

func Error(message string, values ...interface{}) {
	if globalLogger != nil {
		globalLogger.Error(message, values...)
	}
}

func Warn(message string, values ...interface{}) {
	if globalLogger != nil {
		globalLogger.Warn(message, values...)
	}
}

func Debug(message string, values ...interface{}) {
	if globalLogger != nil {
		globalLogger.Debug(message, values...)
	}
}
