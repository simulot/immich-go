package logger

import (
	"fmt"

	"github.com/ttacon/chalk"
)

type Level int

const (
	Fatal Level = iota
	Error
	Warning
	OK
	Info
)

var colorLevel = map[Level]string{
	Fatal:   chalk.Red.String(),
	Error:   chalk.Red.String(),
	Warning: chalk.Yellow.String(),
	OK:      chalk.Green.String(),
	Info:    chalk.White.String(),
}

type Logger struct {
	needCR       bool
	displayLevel Level
}

func NewLogger(DisplayLevel Level) *Logger {
	return &Logger{
		displayLevel: DisplayLevel,
	}
}

func (l *Logger) Info(f string, v ...any) {
	l.Message(Info, f, v...)
}
func (l *Logger) OK(f string, v ...any) {
	l.Message(OK, f, v...)
}
func (l *Logger) Warning(f string, v ...any) {
	l.Message(Warning, f, v...)
}
func (l *Logger) Error(f string, v ...any) {
	l.Message(Error, f, v...)
}
func (l *Logger) Fatal(f string, v ...any) {
	l.Message(Fatal, f, v...)
}

func (l *Logger) Message(level Level, f string, v ...any) {
	if level > l.displayLevel {
		return
	}
	if l.needCR {
		fmt.Println()
		l.needCR = false
	}
	fmt.Print(colorLevel[level])
	fmt.Printf(f, v...)
	fmt.Println(chalk.ResetColor)
}

func (l *Logger) Progress(f string, v ...any) {
	fmt.Printf("\r"+f, v...)
	l.needCR = true
}

func (l *Logger) MessageContinue(level Level, f string, v ...any) {
	if level > l.displayLevel {
		return
	}
	if l.needCR {
		fmt.Println()
		l.needCR = false
	}
	fmt.Print(colorLevel[level], " ")
	fmt.Printf(f, v...)
}

func (l *Logger) MessageTerminate(level Level, f string, v ...any) {
	if level > l.displayLevel {
		return
	}
	fmt.Print(colorLevel[level], " ")
	fmt.Printf(f, v...)
	fmt.Println(chalk.ResetColor)
}
