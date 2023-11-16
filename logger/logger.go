package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ttacon/chalk"
)

type Level int

const (
	Fatal Level = iota
	Error
	Warning
	OK
	Info
	Debug
)

func (l Level) String() string {
	switch l {
	case Fatal:
		return "Fatal"
	case Error:
		return "Error"
	case Warning:
		return "Warning"
	case OK:
		return "OK"
	case Info:
		return "Info"
	case Debug:
		return "Debug"
	default:
		return fmt.Sprintf("Log Level %d", l)
	}
}

func StringToLevel(s string) (Level, error) {
	s = strings.ToLower(s)
	for l := Fatal; l <= Debug; l++ {
		if strings.ToLower(l.String()) == s {
			return l, nil
		}
	}
	return Error, fmt.Errorf("unknown log level: %s", s)
}

var colorLevel = map[Level]string{
	Fatal:   chalk.Red.String(),
	Error:   chalk.Red.String(),
	Warning: chalk.Yellow.String(),
	OK:      chalk.Green.String(),
	Info:    chalk.White.String(),
	Debug:   chalk.Cyan.String(),
}

type Logger struct {
	needCR       bool
	needSpace    bool
	displayLevel Level
	noColors     bool
	colorStrings map[Level]string
	debug        bool
	out          io.Writer
}

func NewLogger(DisplayLevel Level, noColors bool, debug bool) *Logger {
	l := Logger{
		displayLevel: DisplayLevel,
		noColors:     noColors,
		colorStrings: map[Level]string{},
		debug:        debug,
		out:          os.Stdout,
	}
	if !noColors {
		l.colorStrings = colorLevel
	}
	return &l
}

func (l *Logger) Writer(w io.Writer) *Logger {
	l.out = w
	return l
}

func (l *Logger) Debug(f string, v ...any) {
	if l.out == nil {
		return
	}
	l.Message(Debug, f, v...)
}

type DebugObject interface {
	DebugObject() any
}

func (l *Logger) DebugObject(name string, v any) {
	if !l.debug {
		return
	}
	if l.out == nil {
		return
	}
	if d, ok := v.(DebugObject); ok {
		v = d.DebugObject()
	}
	b := bytes.NewBuffer(nil)
	enc := json.NewEncoder(b)
	enc.SetIndent("", " ")
	err := enc.Encode(v)
	if err != nil {
		l.Error("can't display object %s: %s", name, err)
		return
	}
	if l.needCR {
		fmt.Println()
		l.needCR = false
	}
	l.needSpace = false
	fmt.Fprint(l.out, l.colorStrings[Debug])
	fmt.Fprintf(l.out, "%s:\n%s", name, b.String())
	if !l.noColors {
		fmt.Fprint(l.out, chalk.ResetColor)
	}
	fmt.Fprintln(l.out)
}
func (l *Logger) Info(f string, v ...any) {
	if l.out == nil {
		return
	}
	l.Message(Info, f, v...)
}
func (l *Logger) OK(f string, v ...any) {
	if l.out == nil {
		return
	}
	l.Message(OK, f, v...)
}
func (l *Logger) Warning(f string, v ...any) {
	if l.out == nil {
		return
	}
	l.Message(Warning, f, v...)
}
func (l *Logger) Error(f string, v ...any) {
	if l.out == nil {
		return
	}
	l.Message(Error, f, v...)
}
func (l *Logger) Fatal(f string, v ...any) {
	if l.out == nil {
		return
	}
	l.Message(Fatal, f, v...)
}

func (l *Logger) Message(level Level, f string, v ...any) {
	if l.out == nil {
		return
	}
	if level > l.displayLevel {
		return
	}
	if l.needCR {
		fmt.Fprintln(l.out)
		l.needCR = false
	}
	l.needSpace = false
	fmt.Fprint(l.out, l.colorStrings[level])
	fmt.Fprintf(l.out, f, v...)
	if !l.noColors {
		fmt.Fprint(l.out, chalk.ResetColor)
	}
	fmt.Fprintln(l.out)
}

func (l *Logger) Progress(level Level, f string, v ...any) {
	if l.out == nil {
		return
	}
	if level > l.displayLevel {
		return
	}
	fmt.Fprintf(l.out, "\r\033[2K"+f, v...)
	l.needCR = true
}

func (l *Logger) MessageContinue(level Level, f string, v ...any) {
	if l.out == nil {
		return
	}
	if level > l.displayLevel {
		return
	}
	if l.needCR {
		fmt.Fprintln(l.out)
		l.needCR = false
	}
	if l.needSpace {
		fmt.Print(" ")
	}
	fmt.Fprint(l.out, l.colorStrings[level])
	fmt.Fprintf(l.out, f, v...)
	l.needSpace = true
	l.needCR = false
}

func (l *Logger) MessageTerminate(level Level, f string, v ...any) {
	if l.out == nil {
		return
	}
	if level > l.displayLevel {
		return
	}
	fmt.Fprint(l.out, l.colorStrings[level])
	fmt.Fprintf(l.out, f, v...)
	if !l.noColors {
		fmt.Fprint(l.out, chalk.ResetColor)
	}
	fmt.Fprintln(l.out)
	l.needSpace = false
	l.needCR = false
}
