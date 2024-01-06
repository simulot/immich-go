package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"

	"github.com/simulot/immich-go/cmdduplicate"
	"github.com/simulot/immich-go/cmdmetadata"
	"github.com/simulot/immich-go/cmdstack"
	"github.com/simulot/immich-go/cmdtool"
	"github.com/simulot/immich-go/cmdupload"
	"github.com/simulot/immich-go/helpers/myflag"
	"github.com/simulot/immich-go/helpers/tzone"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/logger"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	var err error
	var log = logger.NewLogger(logger.OK, true, false)
	defer log.Close()
	log.OK("immich-go  %s, commit %s, built at %s\n", version, commit, date)

	// Create a context with cancel function to gracefully handle Ctrl+C events
	ctx, cancel := context.WithCancel(context.Background())

	// Handle Ctrl+C signal (SIGINT)
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt)

	go func() {
		<-signalChannel
		fmt.Println("\nCtrl+C received. Shutting down...")
		cancel() // Cancel the context when Ctrl+C is received
	}()

	select {
	case <-ctx.Done():
		err = ctx.Err()
	default:
		log, err = Run(ctx, log)
	}
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
	log.OK("Done.")
}

type Application struct {
	Server      string // Immich server address (http://<your-ip>:2283/api or https://<your-domain>/api)
	API         string // Immich api endpoint (http://container_ip:3301)
	Key         string // API Key
	DeviceUUID  string // Set a device UUID
	ApiTrace    bool   // Enable API call traces
	NoLogColors bool   // Disable log colors
	LogLevel    string // Idicate the log level
	Debug       bool   // Enable the debug mode
	TimeZone    string // Override default TZ
	SkipSSL     bool   // Skip SSL Verification

	Immich  *immich.ImmichClient // Immich client
	Logger  *logger.Log          // Program's logger
	LogFile string               //Log file

}

func Run(ctx context.Context, log *logger.Log) (*logger.Log, error) {

	var err error
	deviceID, err := os.Hostname()
	if err != nil {
		return log, err
	}

	app := Application{}
	flag.StringVar(&app.Server, "server", "", "Immich server address (http://<your-ip>:2283 or https://<your-domain>)")
	flag.StringVar(&app.API, "api", "", "Immich api endpoint (http://container_ip:3301)")
	flag.StringVar(&app.Key, "key", "", "API Key")
	flag.StringVar(&app.DeviceUUID, "device-uuid", deviceID, "Set a device UUID")
	flag.BoolFunc("no-colors-log", "Disable colors on logs", myflag.BoolFlagFn(&app.NoLogColors, runtime.GOOS == "windows"))
	flag.StringVar(&app.LogLevel, "log-level", "ok", "Log level (Error|Warning|OK|Info), default OK")
	flag.StringVar(&app.LogFile, "log-file", "", "Write log messages into the file")
	flag.BoolFunc("api-trace", "enable api call traces", myflag.BoolFlagFn(&app.ApiTrace, false))
	flag.BoolFunc("debug", "enable debug messages", myflag.BoolFlagFn(&app.Debug, false))
	flag.StringVar(&app.TimeZone, "time-zone", "", "Override the system time zone")
	flag.BoolFunc("skip-verify-ssl", "Skip SSL verification", myflag.BoolFlagFn(&app.SkipSSL, false))
	flag.Parse()

	app.Server = strings.TrimSuffix(app.Server, "/")

	_, err = tzone.SetLocal(app.TimeZone)
	if err != nil {
		return log, err
	}

	if len(app.LogFile) > 0 {
		flog, err := os.Create(app.LogFile)
		if err != nil {
			return log, fmt.Errorf("can't open the log file: %w", err)
		}
		log.SetWriter(flog)
		log.OK("immich-go  %s, commit %s, built at %s\n", version, commit, date)
	}

	switch {
	case len(app.Server) == 0 && len(app.API) == 0:
		err = errors.Join(err, errors.New("missing -server, Immich server address (http://<your-ip>:2283 or https://<your-domain>)"))
	case len(app.Server) > 0 && len(app.API) > 0:
		err = errors.Join(err, errors.New("give either the -server or the -api option"))
	}
	if len(app.Key) == 0 {
		err = errors.Join(err, errors.New("missing -key"))
	}

	logLevel, e := logger.StringToLevel(app.LogLevel)
	if err != nil {
		err = errors.Join(err, e)
	}

	if len(flag.Args()) == 0 {
		err = errors.Join(err, errors.New("missing command upload|duplicate|stack"))
	}

	log.SetLevel(logLevel)
	log.SetColors(!app.NoLogColors)
	log.SetDebugFlag(app.Debug)

	app.Logger = log

	if err != nil {
		return app.Logger, err
	}

	app.Immich, err = immich.NewImmichClient(app.Server, app.Key, app.SkipSSL)
	if err != nil {
		return app.Logger, err
	}
	if app.API != "" {
		app.Immich.SetEndPoint(app.API)
	}
	if app.ApiTrace {
		app.Immich.EnableAppTrace(true)
	}

	err = app.Immich.PingServer(ctx)
	if err != nil {
		return app.Logger, err
	}
	app.Logger.OK("Server status: OK")

	user, err := app.Immich.ValidateConnection(ctx)
	if err != nil {
		return app.Logger, err
	}
	app.Logger.Info("Connected, user: %s", user.Email)

	cmd := flag.Args()[0]
	switch cmd {
	case "upload":
		err = cmdupload.UploadCommand(ctx, app.Immich, app.Logger, flag.Args()[1:])
	case "duplicate":
		err = cmdduplicate.DuplicateCommand(ctx, app.Immich, app.Logger, flag.Args()[1:])
	case "metadata":
		err = cmdmetadata.MetadataCommand(ctx, app.Immich, app.Logger, flag.Args()[1:])
	case "stack":
		err = cmdstack.NewStackCommand(ctx, app.Immich, app.Logger, flag.Args()[1:])
	case "tool":
		err = cmdtool.CommandTool(ctx, app.Immich, app.Logger, flag.Args()[1:])
	default:
		err = fmt.Errorf("unknwon command: %q", cmd)
	}
	return app.Logger, err
}
