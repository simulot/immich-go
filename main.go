package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"

	"github.com/simulot/immich-go/app/root"
)

// immich-go entry point
func main() {
	// ARM/1GB fix: set a soft memory limit so the Go GC does not trigger
	// excessively on constrained devices. Without this, GOGC=100 (default)
	// causes the GC to run every time the heap doubles, which on a 1GB
	// device with 512MB available means frequent 5-20ms stop-the-world
	// pauses during large uploads. 800MB leaves headroom for the OS and
	// kernel while allowing the GC to batch work more efficiently.
	// This can be overridden at runtime with GOMEMLIMIT env var.
	if os.Getenv("GOMEMLIMIT") == "" {
		debug.SetMemoryLimit(800 * 1024 * 1024) // 800 MiB
	}

	ctx := context.Background()
	err := immichGoMain(ctx)
	if err != nil {
		if e := context.Cause(ctx); e != nil {
			err = e
		}
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

	// Watch for ^C to be pressed
	go func() {
		<-signalChannel
		fmt.Println("\nCtrl+C received. Shutting down...")
		cancel(errors.New("Ctrl+C received")) // Cancel the context when Ctrl+C is received
	}()

	c, a := root.RootImmichGoCommand(ctx)
	// let's start
	err := c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
	}
	return err
}
