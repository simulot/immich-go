package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"

	"github.com/simulot/immich-go/app/root"
)

var errInterrupt = errors.New("Ctrl+C received")

// immich-go entry point
func main() {
	err := immichGoMain(context.Background())
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// makes immich-go breakable with ^C and run it
func immichGoMain(ctx context.Context) error {
	// Create a context with cancel function to gracefully handle Ctrl+C events
	ctx, cancel := context.WithCancelCause(ctx)

	// Handle Ctrl+C signal (SIGINT)
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt)
	defer signal.Stop(signalChannel)

	// Watch for ^C to be pressed. The first interrupt asks the command to stop
	// gracefully; the second one forces the process to exit.
	startInterruptHandler(cancel, signalChannel, func(code int) {
		fmt.Println("\nSecond Ctrl+C received. Forcing exit...")
		os.Exit(code)
	})

	c, a := root.RootImmichGoCommand(ctx)
	// let's start
	err := c.ExecuteContext(ctx)
	if errors.Is(err, context.Canceled) {
		if cause := context.Cause(ctx); cause != nil {
			err = cause
		}
	}
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
	}
	return err
}

func startInterruptHandler(cancel context.CancelCauseFunc, signals <-chan os.Signal, exitFn func(int)) {
	go func() {
		if _, ok := <-signals; !ok {
			return
		}
		fmt.Println("\nCtrl+C received. Shutting down...")
		cancel(errInterrupt)
		if _, ok := <-signals; !ok {
			return
		}
		exitFn(130)
	}()
}
