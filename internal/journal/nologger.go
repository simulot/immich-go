package journal

import "io"

type NoLog struct{}

func (NoLog) Debug(f string, v ...any)                         {}
func (NoLog) DebugObject(name string, v any)                   {}
func (NoLog) Info(f string, v ...any)                          {}
func (NoLog) OK(f string, v ...any)                            {}
func (NoLog) Warning(f string, v ...any)                       {}
func (NoLog) Error(f string, v ...any)                         {}
func (NoLog) Fatal(f string, v ...any)                         {}
func (NoLog) Message(level Level, f string, v ...any)          {}
func (NoLog) Progress(level Level, f string, v ...any)         {}
func (NoLog) MessageContinue(level Level, f string, v ...any)  {}
func (NoLog) MessageTerminate(level Level, f string, v ...any) {}
func (NoLog) SetWriter(io.WriteCloser)                         {}
func (NoLog) SetLevel(Level)                                   {}
func (NoLog) SetColors(bool)                                   {}
func (NoLog) SetDebugFlag(bool)                                {}
