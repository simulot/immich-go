package application

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/simulot/immich-go/commands/version"
	"github.com/simulot/immich-go/helpers/configuration"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Log struct {
	*slog.Logger // Logger

	Type  string // Log format : text|json
	File  string // Log file name
	Level string // Indicate the log level (string)

	writer io.Writer  // the log writer
	sLevel slog.Level // the log level value
}

func AddLogFlags(ctx context.Context, cmd *cobra.Command, app *Application) {
	log := app.Log()
	cmd.PersistentFlags().StringVar(&log.Level, "log-level", "INFO", "Log level (DEBUG|INFO|WARN|ERROR), default INFO")
	cmd.PersistentFlags().StringVarP(&log.File, "log-file", "l", "", "Write log messages into the file")
	cmd.PersistentFlags().StringVar(&log.Type, "log-type", "text", "Log formatted  as text of JSON file")

	cmd.PersistentPreRunE = ChainRunEFunctions(cmd.PersistentPreRunE, log.Open, ctx, cmd, app)
}

func (log *Log) OpenLogFile() error {
	var w io.WriteCloser

	if log.File == "" {
		log.File = configuration.DefaultLogFile()
	}
	if log.File != "" {
		if log.writer == nil {
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
	log.SetLogWriter(w)
	return nil
}

func (log *Log) Open(ctx context.Context, cmd *cobra.Command, app *Application) error {
	fmt.Println(version.Banner())
	err := log.OpenLogFile()
	if err != nil {
		return err
	}
	// List flags
	log.Info(version.GetVersion())

	log.Info(fmt.Sprintf("Command: %s", cmd.Use))
	log.Info("Flags:")
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		val := flag.Value.String()
		if val == "" {
			if v := viper.GetString(flag.Name); v != "" {
				val = v
			}
		}
		if flag.Name == "api-key" && len(val) > 4 {
			val = strings.Repeat("*", len(val)-4) + val[len(val)-4:]
		}
		log.Info(fmt.Sprintf("  --%s: %q", flag.Name, val))
	})

	// List arguments
	log.Info("Arguments:")
	for _, arg := range cmd.Flags().Args() {
		log.Info(fmt.Sprintf("  %q", arg))
	}
	return nil
}

func (log *Log) SetLogWriter(w io.Writer) *slog.Logger {
	var handler slog.Handler

	switch log.Type {
	case "JSON":
		handler = slog.NewJSONHandler(w, &slog.HandlerOptions{})
	default:
		handler = slog.NewTextHandler(w, &slog.HandlerOptions{
			Level: log.sLevel,
		})

		// humane.NewHandler(w, &humane.Options{Level: log.sLevel})
	}
	log.writer = w
	log.Logger = slog.New(handler)
	return log.Logger
}

func (log *Log) Message(msg string, values ...any) {
	s := fmt.Sprintf(msg, values...)
	fmt.Println(s)
	if log.Logger != nil {
		log.Info(s)
	}
}

func (log *Log) Close(cmd *cobra.Command, args []string) error {
	if log.File != "" {
		log.Message("Check the log file: %s", log.File)
	}
	if closer, ok := log.writer.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func (log *Log) GetWriter() io.Writer {
	return log.writer
}
