package cmd

import (
	"context"
	"errors"
	"flag"
	"os"
	"runtime"
	"strings"

	"github.com/simulot/immich-go/helpers/myflag"
	"github.com/simulot/immich-go/helpers/tzone"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/logger"
)

// SharedFlags collect all parameters that are common to all commands
type SharedFlags struct {
	Server      string // Immich server address (http://<your-ip>:2283/api or https://<your-domain>/api)
	API         string // Immich api endpoint (http://container_ip:3301)
	Key         string // API Key
	DeviceUUID  string // Set a device UUID
	APITrace    bool   // Enable API call traces
	NoLogColors bool   // Disable log colors
	LogLevel    string // Indicate the log level
	Debug       bool   // Enable the debug mode
	TimeZone    string // Override default TZ
	SkipSSL     bool   // Skip SSL Verification

	Immich  immich.ImmichInterface // Immich client
	Logger  *logger.Journal        // Program's logger
	LogFile string                 // Log file
}

// SetFlag add common flags to a flagset
func (app *SharedFlags) SetFlags(fs *flag.FlagSet) {
	fs.StringVar(&app.Server, "server", "", "Immich server address (http://<your-ip>:2283 or https://<your-domain>)")
	fs.StringVar(&app.API, "api", "", "Immich api endpoint (http://container_ip:3301)")
	fs.StringVar(&app.Key, "key", "", "API Key")
	fs.StringVar(&app.DeviceUUID, "device-uuid", "", "Set a device UUID")
	fs.BoolFunc("no-colors-log", "Disable colors on logs", myflag.BoolFlagFn(&app.NoLogColors, runtime.GOOS == "windows"))
	fs.StringVar(&app.LogLevel, "log-level", "ok", "Log level (Error|Warning|OK|Info), default OK")
	fs.StringVar(&app.LogFile, "log-file", "", "Write log messages into the file")
	fs.BoolFunc("api-trace", "enable api call traces", myflag.BoolFlagFn(&app.APITrace, false))
	fs.BoolFunc("debug", "enable debug messages", myflag.BoolFlagFn(&app.Debug, false))
	fs.StringVar(&app.TimeZone, "time-zone", "", "Override the system time zone")
	fs.BoolFunc("skip-verify-ssl", "Skip SSL verification", myflag.BoolFlagFn(&app.SkipSSL, false))
}

func (app *SharedFlags) Start(ctx context.Context) error {
	var joinedErr, err error
	if app.Server != "" {
		app.Server = strings.TrimSuffix(app.Server, "/")
	}
	if app.TimeZone != "" {
		_, err := tzone.SetLocal(app.TimeZone)
		joinedErr = errors.Join(joinedErr, err)
	}

	if app.LogFile != "" {
		f, err := os.Open(app.LogFile)
		if err != nil {
			joinedErr = errors.Join(joinedErr, err)
		} else {
			app.Logger.Log.SetWriter(f)
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

	if app.LogLevel != "" {
		logLevel, err := logger.StringToLevel(app.LogLevel)
		if err != nil {
			joinedErr = errors.Join(joinedErr, err)
		}
		app.Logger.Log.SetLevel(logLevel)

	}
	app.Logger.Log.SetColors(!app.NoLogColors)
	app.Logger.Log.SetDebugFlag(app.Debug)

	// at this point, exits if there is an error
	if joinedErr != nil {
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
	app.Logger.Log.OK("Server status: OK")

	user, err := app.Immich.ValidateConnection(ctx)
	if err != nil {
		return err
	}
	app.Logger.Log.Info("Connected, user: %s", user.Email)

	return nil
}
