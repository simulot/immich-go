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
	needCR bool
}

func (l *Logger) Message(level Level, f string, v ...any) {
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
