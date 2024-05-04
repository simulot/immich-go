package cmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strings"

	"github.com/simulot/immich-go/helpers/configuration"
	"github.com/simulot/immich-go/helpers/fileevent"
	"github.com/simulot/immich-go/helpers/myflag"
	"github.com/simulot/immich-go/helpers/tzone"
	"github.com/simulot/immich-go/immich"
	"github.com/telemachus/humane"
)

// SharedFlags collect all parameters that are common to all commands
type SharedFlags struct {
	ConfigurationFile string // Path to the configuration file to use
	Server            string // Immich server address (http://<your-ip>:2283/api or https://<your-domain>/api)
	API               string // Immich api endpoint (http://container_ip:3301)
	Key               string // API Key
	DeviceUUID        string // Set a device UUID
	APITrace          bool   // Enable API call traces
	NoLogColors       bool   // Disable log colors
	LogLevel          string // Indicate the log level
	Debug             bool   // Enable the debug mode
	TimeZone          string // Override default TZ
	SkipSSL           bool   // Skip SSL Verification

	Immich  immich.ImmichInterface // Immich client
	Log     *slog.Logger           // Logger
	Jnl     *fileevent.Recorder    // Program's logger
	LogFile string                 // Log file name
	out     io.WriteCloser         // the log writer
}

func (app *SharedFlags) InitSharedFlags() {
	app.ConfigurationFile = configuration.DefaultFile()
	app.NoLogColors = runtime.GOOS == "windows"
	app.APITrace = false
	app.Debug = false
	app.SkipSSL = false
	app.LogLevel = "INFO"
}

// SetFlag add common flags to a flagset
func (app *SharedFlags) SetFlags(fs *flag.FlagSet) {
	fs.StringVar(&app.ConfigurationFile, "use-configuration", app.ConfigurationFile, "Specifies the configuration to use")
	fs.StringVar(&app.Server, "server", app.Server, "Immich server address (http://<your-ip>:2283 or https://<your-domain>)")
	fs.StringVar(&app.API, "api", "", "Immich api endpoint (http://container_ip:3301)")
	fs.StringVar(&app.Key, "key", app.Key, "API Key")
	fs.StringVar(&app.DeviceUUID, "device-uuid", app.DeviceUUID, "Set a device UUID")
	fs.BoolFunc("no-colors-log", "Disable colors on logs", myflag.BoolFlagFn(&app.NoLogColors, app.NoLogColors))
	fs.StringVar(&app.LogLevel, "log-level", app.LogLevel, "Log level (DEBUG|INFO|WARN|ERROR), default INFO")
	fs.StringVar(&app.LogFile, "log-file", app.LogFile, "Write log messages into the file")
	fs.BoolFunc("api-trace", "enable api call traces", myflag.BoolFlagFn(&app.APITrace, app.APITrace))
	fs.BoolFunc("debug", "enable debug messages", myflag.BoolFlagFn(&app.Debug, app.Debug))
	fs.StringVar(&app.TimeZone, "time-zone", app.TimeZone, "Override the system time zone")
	fs.BoolFunc("skip-verify-ssl", "Skip SSL verification", myflag.BoolFlagFn(&app.SkipSSL, app.SkipSSL))
}

func (app *SharedFlags) Start(ctx context.Context) error {
	var joinedErr error
	if app.Server != "" {
		app.Server = strings.TrimSuffix(app.Server, "/")
	}
	if app.TimeZone != "" {
		_, err := tzone.SetLocal(app.TimeZone)
		joinedErr = errors.Join(joinedErr, err)
	}

	if app.Jnl == nil {
		app.Jnl = fileevent.NewRecorder(nil)
	}

	if app.LogFile != "" {
		if app.out == nil {
			f, err := os.OpenFile(app.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o664)
			if err != nil {
				joinedErr = errors.Join(joinedErr, err)
			} else {
				var level slog.Level
				err = level.UnmarshalText([]byte(strings.ToUpper(app.LogLevel)))
				if err != nil {
					joinedErr = errors.Join(joinedErr, err)
				} else {
					app.Log = slog.New(humane.NewHandler(f, &humane.Options{Level: level}))
					app.Jnl.SetLogger(app.Log)
				}
			}
			app.out = f
		}
	}

	// at this point, exits if there is an error
	if joinedErr != nil {
		return joinedErr
	}

	// If the client isn't yet initialized
	if app.Immich == nil {
		if app.Server == "" && app.API == "" && app.Key == "" {
			conf, err := configuration.Read(app.ConfigurationFile)
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
			return joinedErr
		}

		// Connection details are saved into the configuration file
		conf := configuration.Configuration{
			ServerURL: app.Server,
			APIKey:    app.Key,
			APIURL:    app.API,
		}
		err := conf.Write(app.ConfigurationFile)
		if err != nil {
			err = fmt.Errorf("can't write into the configuration file: %w", err)
			joinedErr = errors.Join(joinedErr, err)
			return joinedErr
		}

		app.Immich, err = immich.NewImmichClient(app.Server, app.Key, app.SkipSSL)
		if err != nil {
			return err
		}
		if app.API != "" {
			app.Immich.SetEndPoint(app.API)
		}
		if app.APITrace {
			app.Immich.EnableAppTrace(true)
		}
		if app.DeviceUUID != "" {
			app.Immich.SetDeviceUUID(app.DeviceUUID)
		}

		err = app.Immich.PingServer(ctx)
		if err != nil {
			return err
		}
		fmt.Println("Server status: OK")

		user, err := app.Immich.ValidateConnection(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("Connected, user: %s\n", user.Email)
	}
	return nil
}
