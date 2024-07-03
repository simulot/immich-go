package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"

	"github.com/simulot/immich-go/cmd"
	"github.com/simulot/immich-go/cmd/duplicate"
	"github.com/simulot/immich-go/cmd/metadata"
	"github.com/simulot/immich-go/cmd/stack"
	"github.com/simulot/immich-go/cmd/tool"
	"github.com/simulot/immich-go/cmd/upload"
	"github.com/simulot/immich-go/ui"
	"github.com/telemachus/humane"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func getCommitInfo() string {
	dirty := false
	buildvcs := false

	buildinfo, _ := debug.ReadBuildInfo()
	for _, s := range buildinfo.Settings {
		switch s.Key {
		case "vcs.revision":
			buildvcs = true
			commit = s.Value
		case "vcs.modified":
			if s.Value == "true" {
				dirty = true
			}
		case "vcs.time":
			date = s.Value
		}
	}
	if buildvcs && dirty {
		commit += "-dirty"
	}
	return commit
}

func main() {
	var err error

	fmt.Printf("immich-go  %s, commit %s, built at %s\n", version, getCommitInfo(), date)

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
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func Run(ctx context.Context) error {
	app := cmd.SharedFlags{
		Log:    slog.New(humane.NewHandler(os.Stdout, &humane.Options{Level: slog.LevelInfo})),
		Banner: ui.NewBanner(version, commit, date),
	}
	fs := flag.NewFlagSet("main", flag.ExitOnError)
	fs.Func("version", "Get immich-go version", func(s string) error {
		fmt.Println("immich-go", version)
		os.Exit(0)
		return nil
	})

	app.InitSharedFlags()
	app.SetFlags(fs)

	err := fs.Parse(os.Args[1:])
	if err != nil {
		app.Log.Error(err.Error())
		return err
	}
	fmt.Println(app.Banner.String())

	if len(fs.Args()) == 0 {
		err = errors.New("missing command upload|duplicate|stack|tool")
	}

	if err != nil {
		app.Log.Error(err.Error())
		return err
	}

	cmd := fs.Args()[0]
	switch cmd {
	case "upload":
		err = upload.UploadCommand(ctx, &app, fs.Args()[1:])
	case "duplicate":
		err = duplicate.DuplicateCommand(ctx, &app, fs.Args()[1:])
	case "metadata":
		err = metadata.MetadataCommand(ctx, &app, fs.Args()[1:])
	case "stack":
		err = stack.NewStackCommand(ctx, &app, fs.Args()[1:])
	case "tool":
		err = tool.CommandTool(ctx, &app, fs.Args()[1:])
	default:
		err = fmt.Errorf("unknown command: %q", cmd)
	}

	if err != nil {
		app.Log.Error(err.Error())
	}
	fmt.Println("Check the log file: ", app.LogFile)
	if app.APITraceWriter != nil {
		fmt.Println("Check the trace file: ", app.APITraceWriterName)
	}
	return err
}
