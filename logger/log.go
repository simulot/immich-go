package logger

import (
	"os"
	"time"

	"github.com/charmbracelet/log"
)

type Logger interface {
	Debug(msg interface{}, keyvals ...interface{})
	Debugf(format string, args ...interface{})
	Error(msg interface{}, keyvals ...interface{})
	Errorf(format string, args ...interface{})
	Info(msg interface{}, keyvals ...interface{})
	Infof(format string, args ...interface{})
	Print(msg interface{}, keyvals ...interface{})
	Printf(format string, args ...interface{})
	Log(level log.Level, msg interface{}, keyvals ...interface{})
	Logf(level log.Level, format string, args ...interface{})
}

func NewLogger(logLevel string, noColors bool) *log.Logger {
	styles := log.DefaultStyles()
	// styles.Levels[log.ErrorLevel] = lipgloss.NewStyle().
	// 	SetString("ERROR  ").
	// 	Padding(0, 1, 0, 1).
	// 	Background(lipgloss.Color("196")). // Light Red
	// 	Foreground(lipgloss.Color("15"))   // White
	// styles.Levels[log.WarnLevel] = lipgloss.NewStyle().
	// 	SetString("WARNING").
	// 	Padding(0, 1, 0, 1).
	// 	Background(lipgloss.Color("214")). // Kind of Orange
	// 	Foreground(lipgloss.Color("0"))    // Black
	// styles.Levels[log.WarnLevel] = lipgloss.NewStyle().
	// 	SetString("INFO   ").
	// 	Padding(0, 1, 0, 1).
	// 	Background(lipgloss.Color("70")). // Kind of Dark green
	// 	Foreground(lipgloss.Color("0"))   // Black
	// styles.Levels[log.WarnLevel] = lipgloss.NewStyle().
	// 	SetString("DEBUG  ").
	// 	Padding(0, 1, 0, 1).
	// 	Background(lipgloss.Color("128")). // Kind of Purple
	// 	Foreground(lipgloss.Color("15"))   // White

	lv, err := log.ParseLevel(logLevel)
	if err != nil {
		lv = log.InfoLevel
	}

	l := log.NewWithOptions(os.Stderr, log.Options{
		TimeFormat: time.DateTime,
		Level:      lv,
	})
	l.SetStyles(styles)

	return l
}
