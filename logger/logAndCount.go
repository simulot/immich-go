package logger

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
)

type Sender func(msg tea.Msg)

// LogAndCount decorate the log.Logger and provide the AddEntry function to
//   log events in a log.Logger
//   send those events to a tea.Program

type LogAndCount[M Measure] struct {
	l    *log.Logger
	c    *Counters[M]
	send Sender
}

type MsgLog struct {
	Lvl     log.Level
	Message string
	KeyVals []interface{}
}

type MsgStageSpinner struct {
	Label string
}

func NewLogAndCount[M Measure](l *log.Logger, sender Sender, c *Counters[M]) *LogAndCount[M] {
	return &LogAndCount[M]{
		l:    l,
		c:    c,
		send: sender,
	}
}

func (lc LogAndCount[M]) AddEntry(lvl log.Level, counter M, file string, keyval ...interface{}) {
	lc.c.Add(counter)
	keyvals := append([]interface{}{"file", file}, keyval...)
	lc.l.Log(lvl, counter.String(), keyvals...)

	// Send  errors and warnings to the tea.Program event loop
	lc.send(MsgLog{Lvl: lvl, Message: counter.String() + " file:" + file, KeyVals: keyval})
}

func (lc LogAndCount[M]) Stage(label string) {
	lc.l.Print(label)
	lc.send(MsgStageSpinner{Label: label})
}

// Implements some Log functions to display errors and log everything

func (lc LogAndCount[M]) Print(msg interface{}, keyvals ...interface{}) {
	lc.l.Print(msg, keyvals...)
	lc.send(MsgLog{Lvl: log.InfoLevel, Message: fmt.Sprint(msg), KeyVals: keyvals})
}

func (lc LogAndCount[M]) Printf(format string, args ...interface{}) {
	lc.l.Printf(format, args...)
	lc.send(MsgLog{Lvl: log.InfoLevel, Message: fmt.Sprintf(format, args...)})
}

func (lc LogAndCount[M]) Debug(msg interface{}, keyvals ...interface{}) {
	lc.l.Debug(msg, keyvals...)
}

func (lc LogAndCount[M]) Debugf(format string, args ...interface{}) {
	lc.l.Debugf(format, args...)
}

func (lc LogAndCount[M]) Error(msg interface{}, keyvals ...interface{}) {
	lc.l.Error(msg, keyvals...)
	lc.send(MsgLog{Lvl: log.ErrorLevel, Message: fmt.Sprint(msg)})
}

func (lc LogAndCount[M]) Errorf(format string, args ...interface{}) {
	lc.l.Error(format, args...)
	lc.send(MsgLog{Lvl: log.ErrorLevel, Message: fmt.Sprintf(format, args...)})
}

func (lc LogAndCount[M]) Warn(msg interface{}, keyvals ...interface{}) {
	lc.l.Warn(msg, keyvals...)
	lc.send(MsgLog{Lvl: log.WarnLevel, Message: fmt.Sprint(msg)})
}

func (lc LogAndCount[M]) Warnf(format string, args ...interface{}) {
	lc.l.Debug(format, args...)
	lc.send(MsgLog{Lvl: log.WarnLevel, Message: fmt.Sprintf(format, args...)})
}

func (lc LogAndCount[M]) String() string {
	b := strings.Builder{}
	for c, v := range lc.c.counters {
		b.WriteString(fmt.Sprintf("%s: %d\n", c, v))
	}
	return b.String()
}

func SendNop(tea.Msg) {
}
