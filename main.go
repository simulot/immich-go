package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"immich-go/cmdduplicate"
	"immich-go/cmdmetadata"
	"immich-go/cmdupload"
	"immich-go/immich"
	"immich-go/immich/logger"
	"os"
	"os/signal"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	fmt.Printf("immich-go  %s, commit %s, built at %s\n", version, commit, date)
	var err error
	var log = logger.NewLogger(logger.OK, true)
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
	EndPoint   string               // Immich server address (http://<your-ip>:2283/api or https://<your-domain>/api)
	Key        string               // API Key
	DeviceUUID string               // Set a device UUID
	Immich     *immich.ImmichClient // Immich client
	Logger     *logger.Logger       // Program's logger
	ApiTrace   bool

	NoLogColors bool // Disable log colors
	LogLevel    string
}

func Run(ctx context.Context, log *logger.Logger) (*logger.Logger, error) {
	var err error
	deviceID, err := os.Hostname()
	if err != nil {
		return log, err
	}

	app := Application{}
	flag.StringVar(&app.EndPoint, "server", "", "Immich server address (http://<your-ip>:2283 or https://<your-domain>)")
	flag.StringVar(&app.Key, "key", "", "API Key")
	flag.StringVar(&app.DeviceUUID, "device-uuid", deviceID, "Set a device UUID")
	flag.BoolVar(&app.NoLogColors, "no-colors-log", false, "Disable colors on logs")
	flag.StringVar(&app.LogLevel, "log-level", "ok", "Log level (Error|Warning|OK|Info|Debug), default OK")
	flag.BoolVar(&app.ApiTrace, "api-trace", false, "enable api call traces")
	flag.Parse()
	if len(app.EndPoint) == 0 {
		err = errors.Join(err, errors.New("missing -server"))
	}
	if len(app.Key) == 0 {
		err = errors.Join(err, errors.New("missing -key"))
	}

	logLevel, e := logger.StringToLevel(app.LogLevel)
	if err != nil {
		err = errors.Join(err, e)
	}

	if len(flag.Args()) == 0 {
		err = errors.Join(err, errors.New("missing command"))
	}

	app.Logger = logger.NewLogger(logLevel, app.NoLogColors)

	if err != nil {
		return app.Logger, err
	}

	app.Immich, err = immich.NewImmichClient(app.EndPoint, app.Key, app.DeviceUUID, app.ApiTrace)
	if err != nil {
		return app.Logger, err
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
	default:
		err = fmt.Errorf("unknwon command: %q", cmd)
	}
	return app.Logger, err
}
