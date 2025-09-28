package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"runtime"
	"strings"
	"time"

	"github.com/phsym/console-slog"
	slogmulti "github.com/samber/slog-multi"
	"github.com/simulot/immich-go/immich/httptrace"
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

	apiTracer      *httptrace.Tracer
	apiTraceWriter *os.File
	apiTraceName   string
}

func AddLogFlags(ctx context.Context, cmd *cobra.Command, app *Application) {
	log := app.Log()
	cmd.PersistentFlags().StringVar(&log.Level, "log-level", "INFO", "Log level (DEBUG|INFO|WARN|ERROR), default INFO")
	cmd.PersistentFlags().StringVarP(&log.File, "log-file", "l", "", "Write log messages into the file")
	cmd.PersistentFlags().StringVar(&log.Type, "log-type", "text", "Log formatted  as text of JSON file")

	// Bind log flags to Viper
	_ = viper.BindPFlag("logging.level", cmd.PersistentFlags().Lookup("log-level"))
	_ = viper.BindPFlag("logging.file", cmd.PersistentFlags().Lookup("log-file"))

	cmd.PersistentPreRunE = ChainRunEFunctions(cmd.PersistentPreRunE, log.LoadConfiguration, ctx, cmd, app)
	cmd.PersistentPreRunE = ChainRunEFunctions(cmd.PersistentPreRunE, log.Open, ctx, cmd, app)
	cmd.PersistentPostRunE = ChainRunEFunctions(cmd.PersistentPostRunE, log.Close, ctx, cmd, app)
}

func (log *Log) LoadConfiguration(ctx context.Context, cmd *cobra.Command, app *Application) error {
	// Load configuration values from Viper into log options
	config, err := configuration.GetConfiguration()
	if err != nil {
		return err
	}

	// Apply configuration values (only if not set by flags)
	if !cmd.PersistentFlags().Changed("log-level") {
		log.Level = config.Logging.Level
	}

	if !cmd.PersistentFlags().Changed("log-file") && config.Logging.File != "" {
		log.File = config.Logging.File
	}

	return nil
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
	for c := cmd; c != nil; c = c.Parent() {
		switch c.Name() {
		case "version", "completion":
			// no log, nor banner for those commands
			return nil
		}
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
	log.Info("Resolved flag values (after configuration file and environment variable processing):")
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		val := log.getResolvedFlagValue(flag)
		source := log.getFlagSource(flag)

		if strings.Contains(flag.Name, "api-key") && len(val) > 4 {
			val = strings.Repeat("*", len(val)-4) + val[len(val)-4:]
		}

		if val != "" {
			log.Info("", "--"+flag.Name, val, "source", source)
		} else {
			log.Info("", "--"+flag.Name, "<not set>", "source", "default")
		}
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

	log.Logger = slog.New(NewFilteredHandler(slogmulti.Fanout(handlers...)))
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
	if log.apiTraceWriter != nil {
		log.apiTracer.Close()
		log.Message("Check the API-TRACE file: %s", log.apiTraceName)
		log.apiTraceWriter.Close()
	}

	if closer, ok := log.mainWriter.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func (log *Log) GetSLog() *slog.Logger {
	return log.Logger
}

func (log *Log) OpenAPITrace() error {
	if log.apiTraceWriter == nil {
		var err error
		log.apiTraceName = strings.TrimSuffix(log.File, path.Ext(log.File)) + ".trace.log"
		log.apiTraceWriter, err = os.OpenFile(log.apiTraceName, os.O_CREATE|os.O_WRONLY, 0o664)
		if err != nil {
			return err
		}
		log.Message("Check the API-TRACE file: %s", log.apiTraceName)
		log.apiTracer = httptrace.NewTracer(log.apiTraceWriter)
	}
	return nil
}

func (log *Log) APITracer() *httptrace.Tracer {
	return log.apiTracer
}

// getResolvedFlagValue returns the final resolved value for a flag after considering
// CLI flags, environment variables, and configuration file values
func (log *Log) getResolvedFlagValue(flag *pflag.Flag) string {
	// First, try to get the value from the flag itself (CLI argument)
	val := flag.Value.String()

	// If flag is not set, try to get from Viper (which handles config file + env vars)
	if val == "" || !flag.Changed {
		// Try various Viper key mappings
		viperKeys := []string{
			flag.Name,
			strings.ReplaceAll(flag.Name, "-", "_"),
			strings.ReplaceAll(flag.Name, "-", "."),
		}

		for _, key := range viperKeys {
			if viperVal := viper.GetString(key); viperVal != "" {
				val = viperVal
				break
			}
		}
	}

	return val
}

// getFlagSource determines where a flag's value came from
func (log *Log) getFlagSource(flag *pflag.Flag) string {
	// If flag was explicitly set via CLI, it takes precedence
	if flag.Changed {
		return "CLI flag"
	}

	// Check if value comes from environment variable (flag name style)
	envKey := "IMMICHGO_" + strings.ToUpper(strings.ReplaceAll(flag.Name, "-", "_"))
	if os.Getenv(envKey) != "" {
		return "environment variable"
	}

	// Check if value comes from config file by testing common Viper key mappings
	// We need to check the actual configuration structure keys used by Viper
	configKeys := log.getConfigKeysForFlag(flag.Name)

	for _, key := range configKeys {
		if viper.IsSet(key) {
			// Check if it's from environment variable (config key style)
			envKeyAlt := "IMMICHGO_" + strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
			if os.Getenv(envKeyAlt) != "" {
				return "environment variable"
			}

			viperVal := viper.GetString(key)
			flagVal := log.getResolvedFlagValue(flag)

			// If the viper value matches what we resolved and we have a config file
			if viperVal != "" && viperVal == flagVal && viper.ConfigFileUsed() != "" {
				return "configuration file"
			}
		}
	}

	return "default"
}

// getConfigKeysForFlag maps flag names to their configuration file keys
func (log *Log) getConfigKeysForFlag(flagName string) []string {
	// Map common flags to their config file keys
	configKeyMap := map[string][]string{
		"server":                {"server.url"},
		"api-key":               {"server.api_key"},
		"admin-api-key":         {"server.admin_api_key"},
		"skip-verify-ssl":       {"server.skip_ssl"},
		"client-timeout":        {"server.client_timeout"},
		"device-uuid":           {"server.device_uuid"},
		"time-zone":             {"server.time_zone"},
		"on-server-errors":      {"server.on_server_errors"},
		"dry-run":               {"upload.dry_run"},
		"concurrent-uploads":    {"upload.concurrent_uploads"},
		"overwrite":             {"upload.overwrite"},
		"pause-immich-jobs":     {"upload.pause_immich_jobs"},
		"no-ui":                 {"ui.no_ui"},
		"log-level":             {"logging.level"},
		"log-file":              {"logging.file"},
		"api-trace":             {"logging.api_trace"},
		"date-range":            {"archive.date_range", "stack.date_range"},
		"manage-heic-jpeg":      {"stack.manage_heic_jpeg"},
		"manage-raw-jpeg":       {"stack.manage_raw_jpeg"},
		"manage-burst":          {"stack.manage_burst"},
		"manage-epson-fastfoto": {"stack.manage_epson_fastfoto"},
	}

	if keys, exists := configKeyMap[flagName]; exists {
		return keys
	}

	// Fallback to standard transformations
	return []string{
		flagName,
		strings.ReplaceAll(flagName, "-", "_"),
		strings.ReplaceAll(flagName, "-", "."),
	}
}

// FilteredHandler filterslog messages and filters out context canceled errors
// if err, ok := a.Value.Any().(error); ok {
// if errors.Is(err, context.Canceled) {
type FilteredHandler struct {
	handler slog.Handler
}

var _ slog.Handler = (*FilteredHandler)(nil)

func NewFilteredHandler(handler slog.Handler) slog.Handler {
	return &FilteredHandler{
		handler: handler,
	}
}

func (h *FilteredHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *FilteredHandler) Handle(ctx context.Context, r slog.Record) error {
	// When error level is Error or more serious
	if r.Level >= slog.LevelError {
		keepMe := true
		// parses the attributes
		r.Attrs(func(a slog.Attr) bool {
			if err, ok := a.Value.Any().(error); ok {
				if errors.Is(err, context.Canceled) {
					keepMe = false
					return false
				}
			}
			return true
		})
		if !keepMe {
			return nil
		}
	}
	// Otherwise, log the message
	return h.handler.Handle(ctx, r)
}

func (h *FilteredHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &FilteredHandler{handler: h.handler.WithAttrs(attrs)}
}

func (h *FilteredHandler) WithGroup(name string) slog.Handler {
	return &FilteredHandler{handler: h.handler.WithGroup(name)}
}
