package logger

type NoLogger struct{}

func (NoLogger) Debug(f string, v ...any)                         {}
func (NoLogger) DebugObject(name string, v any)                   {}
func (NoLogger) Info(f string, v ...any)                          {}
func (NoLogger) OK(f string, v ...any)                            {}
func (NoLogger) Warning(f string, v ...any)                       {}
func (NoLogger) Error(f string, v ...any)                         {}
func (NoLogger) Fatal(f string, v ...any)                         {}
func (NoLogger) Message(level Level, f string, v ...any)          {}
func (NoLogger) Progress(level Level, f string, v ...any)         {}
func (NoLogger) MessageContinue(level Level, f string, v ...any)  {}
func (NoLogger) MessageTerminate(level Level, f string, v ...any) {}
