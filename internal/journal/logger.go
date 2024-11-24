package journal

import "io"

type Logger interface {
	Debug(f string, v ...any)
	DebugObject(name string, v any)
	Info(f string, v ...any)
	OK(f string, v ...any)
	Warning(f string, v ...any)
	Error(f string, v ...any)
	Fatal(f string, v ...any)
	Message(level Level, f string, v ...any)
	Progress(level Level, f string, v ...any)
	MessageContinue(level Level, f string, v ...any)
	MessageTerminate(level Level, f string, v ...any)
	SetWriter(io.WriteCloser)
	SetLevel(Level)
	SetColors(bool)
	SetDebugFlag(bool)
}
