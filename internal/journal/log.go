package journal

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

type Log struct {
	needCR       bool
	needSpace    bool
	displayLevel Level
	noColors     bool
	colorStrings map[Level]string
	debug        bool
	out          io.WriteCloser
}

func NewLogger(displayLevel Level, noColors bool, debug bool) *Log {
	l := Log{
		displayLevel: displayLevel,
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

func (l *Log) Close() error {
	if l.out != os.Stdout {
		return l.out.Close()
	}
	return nil
}

func (l *Log) SetDebugFlag(flag bool) {
	l.debug = flag
}

func (l *Log) SetLevel(level Level) {
	l.displayLevel = level
}

func (l *Log) SetColors(flag bool) {
	if l.out != os.Stdout {
		flag = false
	}
	if flag {
		l.colorStrings = colorLevel
		l.noColors = false
	} else {
		l.colorStrings = map[Level]string{}
		l.noColors = true
	}
}

func (l *Log) SetWriter(w io.WriteCloser) {
	if l != nil && w != nil {
		l.out = w
		l.noColors = true
		l.colorStrings = map[Level]string{}
	}
}

func (l *Log) Debug(f string, v ...any) {
	if l == nil || l.out == nil {
		return
	}
	l.Message(Debug, f, v...)
}

type DebugObject interface {
	DebugObject() any
}

func (l *Log) DebugObject(name string, v any) {
	if l == nil || !l.debug {
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

func (l *Log) Info(f string, v ...any) {
	if l == nil || l.out == nil {
		fmt.Printf(f, v...)
		fmt.Println()
		return
	}
	l.Message(Info, f, v...)
}

func (l *Log) OK(f string, v ...any) {
	if l == nil || l.out == nil {
		fmt.Printf(f, v...)
		fmt.Println()
		return
	}
	l.Message(OK, f, v...)
}

func (l *Log) Warning(f string, v ...any) {
	if l == nil || l.out == nil {
		fmt.Printf(f, v...)
		fmt.Println()
		return
	}
	l.Message(Warning, f, v...)
}

func (l *Log) Error(f string, v ...any) {
	if l == nil || l.out == nil {
		fmt.Printf(f, v...)
		fmt.Println()
		return
	}
	l.Message(Error, f, v...)
}

func (l *Log) Fatal(f string, v ...any) {
	if l == nil || l.out == nil {
		fmt.Printf(f, v...)
		fmt.Println()
		return
	}
	l.Message(Fatal, f, v...)
}

func (l *Log) Message(level Level, f string, v ...any) {
	if l == nil || l.out == nil {
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

func (l *Log) Progress(level Level, f string, v ...any) {
	if l == nil || l.out == nil {
		return
	}
	if level > l.displayLevel {
		return
	}
	fmt.Fprintf(l.out, "\r\033[2K"+f, v...)
	l.needCR = true
}

func (l *Log) MessageContinue(level Level, f string, v ...any) {
	if l == nil || l.out == nil {
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

func (l *Log) MessageTerminate(level Level, f string, v ...any) {
	if l == nil || l.out == nil {
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
