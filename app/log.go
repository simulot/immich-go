package app

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/phsym/console-slog"
	slogmulti "github.com/samber/slog-multi"
	"github.com/simulot/immich-go/internal/configuration"
	"github.com/simulot/immich-go/internal/fshelper/debugfiles"
	"github.com/simulot/immich-go/internal/loghelper"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Log struct {
	*slog.Logger // Logger

	Type  string // Log format : text|json
	File  string // Log file name
	Level string // Indicate the log level (string)

	sLevel slog.Level // the log level value

	mainWriter    io.Writer // the log writer to file
	consoleWriter io.Writer
}

func AddLogFlags(ctx context.Context, cmd *cobra.Command, app *Application) {
	log := app.Log()
	cmd.PersistentFlags().StringVar(&log.Level, "log-level", "INFO", "Log level (DEBUG|INFO|WARN|ERROR), default INFO")
	cmd.PersistentFlags().StringVarP(&log.File, "log-file", "l", "", "Write log messages into the file")
	cmd.PersistentFlags().StringVar(&log.Type, "log-type", "text", "Log formatted  as text of JSON file")

	cmd.PersistentPreRunE = ChainRunEFunctions(cmd.PersistentPreRunE, log.Open, ctx, cmd, app)
	cmd.PersistentPostRunE = ChainRunEFunctions(cmd.PersistentPostRunE, log.Close, ctx, cmd, app)
}

func (log *Log) OpenLogFile() error {
	var w io.WriteCloser

	if log.File == "" {
		log.File = configuration.DefaultLogFile()
	}
	if log.File != "" {
		if log.mainWriter == nil {
			err := configuration.MakeDirForFile(log.File)
			if err != nil {
				return err
			}
			w, err = os.OpenFile(log.File, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o664)
			if err != nil {
				return err
			}
			err = log.sLevel.UnmarshalText([]byte(strings.ToUpper(log.Level)))
			if err != nil {
				return err
			}
			log.Message("Log file: %s", log.File)
		}
	} else {
		w = os.Stdout
	}
	log.setHandlers(w, nil)
	loghelper.SetGlobalLogger(log.Logger)
	return nil
}

func (log *Log) Open(ctx context.Context, cmd *cobra.Command, app *Application) error {
	if cmd.Name() == "version" {
		// No log for version command
		return nil
	}
	fmt.Println(Banner())
	err := log.OpenLogFile()
	if err != nil {
		return err
	}
	// List flags
	log.Info(GetVersion())
	log.Info("Running environment:", "architecture", runtime.GOARCH, "os", runtime.GOOS)

	cmdStack := []string{cmd.Name()}
	for c := cmd.Parent(); c != nil; c = c.Parent() {
		cmdStack = append([]string{c.Name()}, cmdStack...)
	}

	log.Info(fmt.Sprintf("Command: %s", strings.Join(cmdStack, " ")))
	log.Info("Flags:")
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		val := flag.Value.String()
		if val == "" {
			if v := viper.GetString(flag.Name); v != "" {
				val = v
			}
		}
		if strings.Contains(flag.Name, "api-key") && len(val) > 4 {
			val = strings.Repeat("*", len(val)-4) + val[len(val)-4:]
		}
		log.Info("", "--"+flag.Name, val)
	})

	// List arguments
	log.Info("Arguments:")
	for _, arg := range cmd.Flags().Args() {
		log.Info(fmt.Sprintf("  %q", arg))
	}
	if log.sLevel == slog.LevelDebug {
		debugfiles.EnableTrackFiles(log.Logger)
	}

	return nil
}

/*
func replaceAttr(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.LevelKey {
		level := a.Value.Any().(slog.Level)
		a.Value = slog.StringValue(fmt.Sprintf("%-7s", level.String()))
	}
	return a
}
*/

func (log *Log) setHandlers(file, con io.Writer) {
	handlers := []slog.Handler{}

	log.mainWriter = file
	if log.Type == "JSON" {
		handlers = append(handlers, slog.NewJSONHandler(log.mainWriter, &slog.HandlerOptions{
			Level: log.sLevel,
		}))
	} else {
		handlers = append(handlers, console.NewHandler(log.mainWriter, &console.HandlerOptions{
			// ReplaceAttr: replaceAttr,
			Level:      log.sLevel,
			TimeFormat: time.DateTime,
			NoColor:    true,
			Theme:      console.NewDefaultTheme(),
		}))
	}

	log.consoleWriter = con
	if log.consoleWriter != nil {
		handlers = append(handlers, console.NewHandler(log.consoleWriter, &console.HandlerOptions{
			// ReplaceAttr: replaceAttr,
			Level:      log.sLevel,
			TimeFormat: time.DateTime,
			NoColor:    false,
			Theme:      console.NewDefaultTheme(),
		}))
	}

	log.Logger = slog.New(slogmulti.Fanout(handlers...))
}

func (log *Log) SetLogWriter(w io.Writer) *slog.Logger {
	log.setHandlers(log.mainWriter, w)
	return log.Logger
}

func (log *Log) Message(msg string, values ...any) {
	s := fmt.Sprintf(msg, values...)
	fmt.Println(s)
	if log.Logger != nil {
		log.Info(s)
	}
}

func (log *Log) Close(ctx context.Context, cmd *cobra.Command, app *Application) error {
	if cmd.Name() == "version" {
		// No log for version command
		return nil
	}
	debugfiles.ReportTrackedFiles()
	if log.File != "" {
		log.Message("Check the log file: %s", log.File)
	}
	if closer, ok := log.mainWriter.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func (log *Log) GetSLog() *slog.Logger {
	return log.Logger
}
