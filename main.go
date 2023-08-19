package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"immich-go/immich"
	"immich-go/immich/logger"
	"immich-go/upcmd"
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
	app := &Application{
		Logger: logger.NewLogger(logger.Info),
	}
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
		err = Run(ctx)
	}
	if err != nil {
		app.Logger.Error(err.Error())
		os.Exit(1)
	}
	app.Logger.OK("Done.")
}

type Application struct {
	EndPoint   string               // Immich server address (http://<your-ip>:2283/api or https://<your-domain>/api)
	Key        string               // API Key
	DeviceUUID string               // Set a device UUID
	Immich     *immich.ImmichClient // Immich client
	Logger     *logger.Logger       // Program's logger

}

func Run(ctx context.Context) error {
	var err error
	deviceID, err := os.Hostname()
	if err != nil {
		return err
	}

	app := Application{
		Logger: logger.NewLogger(logger.Info),
	}
	flag.StringVar(&app.EndPoint, "server", "", "Immich server address (http://<your-ip>:2283 or https://<your-domain>)")
	flag.StringVar(&app.Key, "key", "", "API Key")
	flag.StringVar(&app.DeviceUUID, "device-uuid", deviceID, "Set a device UUID")

	flag.Parse()
	if len(app.EndPoint) == 0 {
		err = errors.Join(err, errors.New("missing -server"))
	}

	if len(app.Key) == 0 {
		err = errors.Join(err, errors.New("missing -key"))
	}

	if len(flag.Args()) == 0 {
		err = errors.Join(err, errors.New("missing command"))
	}

	if err != nil {
		return err
	}
	app.Immich, err = immich.NewImmichClient(app.EndPoint, app.Key, app.DeviceUUID)
	if err != nil {
		return err
	}

	err = app.Immich.PingServer(ctx)
	if err != nil {
		return err
	}
	app.Logger.OK("Server status: OK")

	user, err := app.Immich.ValidateConnection(ctx)
	if err != nil {
		return err
	}
	app.Logger.Info("Connected, user: %s", user.Email)

	cmd := flag.Args()[0]
	switch cmd {
	case "upload":
		err = upcmd.UploadCommand(ctx, app.Immich, app.Logger, flag.Args()[1:])
	default:
		err = fmt.Errorf("unknwon command: %q", cmd)
	}
	return err
}
