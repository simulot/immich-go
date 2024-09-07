package cmd

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/simulot/immich-go/helpers/configuration"
	"github.com/simulot/immich-go/ui"
	"github.com/spf13/cobra"
	"github.com/telemachus/humane"
)

func CreateRootCommand(b ui.Banner) *RootImmichFlags {
	rootFlags := RootImmichFlags{
		Command: &cobra.Command{
			Use:   "immich-go",
			Short: "Immich-go is a command line application to interact with the Immich application using its API",
			Long:  `An alternative to the immich-CLI command that doesn't depend on nodejs installation. It tries its best for importing google photos takeout archives.`,
		},
		Banner: b,
	}

	rootFlags.Command.PersistentFlags().StringVar(&rootFlags.ConfigurationFile, "use-configuration", rootFlags.ConfigurationFile, "Specifies the configuration to use")
	rootFlags.Command.PersistentFlags().StringVar(&rootFlags.LogLevel, "log-level", "INFO", "Log level (DEBUG|INFO|WARN|ERROR), default INFO")
	rootFlags.Command.PersistentFlags().StringVar(&rootFlags.LogFile, "log-file", configuration.DefaultLogFile(), "Write log messages into the file")
	rootFlags.Command.PersistentFlags().BoolVar(&rootFlags.JSONLog, "log-json", false, "Output line-delimited JSON file, default FALSE")
	return &rootFlags
}

// RootImmichFlags is the root command flags
type RootImmichFlags struct {
	Command           *cobra.Command // Cobra root command
	ConfigurationFile string         // Path to the configuration file to use
	JSONLog           bool           // Enable JSON structured log
	Banner            ui.Banner      // Immich-go banner
	Log               *slog.Logger   // Logger
	LogFile           string         // Log file name
	LogWriterCloser   io.WriteCloser // the log writer
	LogLevel          string         // Indicate the log level (string)
	Level             slog.Level     // Set the log level
}

func (app *RootImmichFlags) Initialize() error {
	fmt.Println(app.Banner.String())
	var w io.WriteCloser
	if app.LogFile != "" {
		if app.LogWriterCloser == nil {
			err := configuration.MakeDirForFile(app.LogFile)
			if err != nil {
				return err
			}
			w, err = os.OpenFile(app.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o664)
			if err != nil {
				return err
			}
			err = app.Level.UnmarshalText([]byte(strings.ToUpper(app.LogLevel)))
			if err != nil {
				return err
			}
		}
	} else {
		w = os.Stdout
	}
	app.SetLogWriter(w)
	app.LogWriterCloser = w
	app.Log.Info(app.Banner.Version())

	return nil
}

func (app *RootImmichFlags) SetLogWriter(w io.Writer) {
	var handler slog.Handler
	if app.JSONLog {
		handler = slog.NewJSONHandler(w, &slog.HandlerOptions{})
	} else {
		handler = humane.NewHandler(w, &humane.Options{Level: app.Level})
	}
	app.Log = slog.New(handler)
}

/*

func (app *RootImmichFlags) InitSharedFlags() {
	app.ConfigurationFile = configuration.DefaultConfigFile()
	app.LogFile = configuration.DefaultLogFile()
	app.APITrace = false
	app.Debug = false
	app.SkipSSL = false
	app.LogLevel = "INFO"
	app.NoUI = false
	app.JSONLog = false
	app.ClientTimeout = 5 * time.Minute
}

// // SetFlag add common flags to a flagset
// func (app *ImmichServerFlags) SetFlags(fs *flag.FlagSet) {
// 	fs.StringVar(&app.ConfigurationFile, "use-configuration", app.ConfigurationFile, "Specifies the configuration to use")
// 	fs.StringVar(&app.Server, "server", app.Server, "Immich server address (http://<your-ip>:2283 or https://<your-domain>)")
// 	fs.StringVar(&app.API, "api", app.API, "Immich api endpoint (http://container_ip:3301)")
// 	fs.StringVar(&app.Key, "key", app.Key, "API Key")
// 	fs.StringVar(&app.DeviceUUID, "device-uuid", app.DeviceUUID, "Set a device UUID")
// 	fs.StringVar(&app.LogLevel, "log-level", app.LogLevel, "Log level (DEBUG|INFO|WARN|ERROR), default INFO")
// 	fs.StringVar(&app.LogFile, "log-file", app.LogFile, "Write log messages into the file")
// 	fs.BoolFunc("log-json", "Output line-delimited JSON file, default FALSE", myflag.BoolFlagFn(&app.JSONLog, app.JSONLog))
// 	fs.BoolFunc("api-trace", "enable trace of api calls", myflag.BoolFlagFn(&app.APITrace, app.APITrace))
// 	fs.BoolFunc("debug", "enable debug messages", myflag.BoolFlagFn(&app.Debug, app.Debug))
// 	fs.StringVar(&app.TimeZone, "time-zone", app.TimeZone, "Override the system time zone")
// 	fs.BoolFunc("skip-verify-ssl", "Skip SSL verification", myflag.BoolFlagFn(&app.SkipSSL, app.SkipSSL))
// 	fs.BoolFunc("no-ui", "Disable the user interface", myflag.BoolFlagFn(&app.NoUI, app.NoUI))
// 	fs.Func("client-timeout", "Set server calls timeout, default 1m", myflag.DurationFlagFn(&app.ClientTimeout, app.ClientTimeout))
// 	fs.BoolFunc("debug-counters", "generate a CSV file with actions per handled files", myflag.BoolFlagFn(&app.DebugCounters, false))
// }

func (app *RootImmichFlags) Start(ctx context.Context) error {
	var joinedErr error
	if app.Server != "" {
		app.Server = strings.TrimSuffix(app.Server, "/")
	}
	if app.TimeZone != "" {
		_, err := tzone.SetLocal(app.TimeZone)
		joinedErr = errors.Join(joinedErr, err)
	}

	if app.Jnl == nil {
		app.Jnl = fileevent.NewRecorder(nil, app.DebugCounters)
	}

	if app.DebugFileList {
		app.Immich = &fakeimmich.MockedCLient{}
		_ = os.Remove(app.LogFile)
	}

	if app.LogFile != "" {
		if app.LogWriterCloser == nil {
			err := configuration.MakeDirForFile(app.LogFile)
			if err != nil {
				return err
			}
			f, err := os.OpenFile(app.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o664)
			if err != nil {
				return err
			}
			err = app.Level.UnmarshalText([]byte(strings.ToUpper(app.LogLevel)))
			if err != nil {
				return err
			}
			app.SetLogWriter(f)
			app.LogWriterCloser = f
		}
	}

	// If the client isn't yet initialized
	if app.Immich == nil {
		if app.Server == "" && app.API == "" && app.Key == "" {
			conf, err := configuration.ConfigRead(app.ConfigurationFile)
			confExist := err == nil
			if confExist && app.Server == "" && app.Key == "" && app.API == "" {
				app.Server = conf.ServerURL
				app.Key = conf.APIKey
				app.API = conf.APIURL
			}
		}

		switch {
		case app.Server == "" && app.API == "":
			joinedErr = errors.Join(joinedErr, errors.New("missing -server, Immich server address (http://<your-ip>:2283 or https://<your-domain>)"))
		case app.Server != "" && app.API != "":
			joinedErr = errors.Join(joinedErr, errors.New("give either the -server or the -api option"))
		}
		if app.Key == "" {
			joinedErr = errors.Join(joinedErr, errors.New("missing -key"))
		}

		if joinedErr != nil {
			return joinedErr
		}

		// Connection details are saved into the configuration file
		conf := configuration.Configuration{
			ServerURL: app.Server,
			APIKey:    app.Key,
			APIURL:    app.API,
		}
		err := configuration.MakeDirForFile(app.ConfigurationFile)
		if err != nil {
			return err
		}
		err = conf.Write(app.ConfigurationFile)
		if err != nil {
			return fmt.Errorf("can't write into the configuration file: %w", err)
		}
		app.Log.Info("Connection to the server " + app.Server)

		app.Immich, err = immich.NewImmichClient(app.Server, app.Key, immich.OptionVerifySSL(app.SkipSSL), immich.OptionConnectionTimeout(app.ClientTimeout))
		if err != nil {
			return err
		}
		if app.API != "" {
			app.Immich.SetEndPoint(app.API)
		}
		if app.DeviceUUID != "" {
			app.Immich.SetDeviceUUID(app.DeviceUUID)
		}

		if app.APITrace {
			if app.APITraceWriter == nil {
				err := configuration.MakeDirForFile(app.LogFile)
				if err != nil {
					return err
				}
				app.APITraceWriterName = strings.TrimSuffix(app.LogFile, filepath.Ext(app.LogFile)) + ".trace.log"
				app.APITraceWriter, err = os.OpenFile(app.APITraceWriterName, os.O_CREATE|os.O_WRONLY, 0o664)
				if err != nil {
					return err
				}
				app.Immich.EnableAppTrace(app.APITraceWriter)
			}
		}

		err = app.Immich.PingServer(ctx)
		if err != nil {
			return err
		}
		app.Log.Info("Server status: OK")

		user, err := app.Immich.ValidateConnection(ctx)
		if err != nil {
			return err
		}
		app.Log.Info(fmt.Sprintf("Connected, user: %s", user.Email))
	}

	return nil
}

*/
