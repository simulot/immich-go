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
	"github.com/simulot/immich-go/logger"
	"github.com/simulot/immich-go/ui"
	"github.com/telemachus/humane"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func getCommitInfo() string {
	commit := commit // default can be set with: -ldflags "-X 'main.commit=...'"
	var dirty bool = false
	var buildvcs bool = false

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
		os.Exit(1)
	}
}

func Run(ctx context.Context) error {
	log := logger.NewLogger(logger.OK, true, false)
	defer log.Close()

	app := cmd.SharedFlags{
		Log:    slog.New(humane.NewHandler(os.Stdout, &humane.Options{Level: slog.LevelInfo})),
		Banner: ui.NewBanner(version, commit, date),
	}
	fmt.Println(app.Banner.String())
	fs := flag.NewFlagSet("main", flag.ExitOnError)
	app.InitSharedFlags()
	app.SetFlags(fs)

	err := fs.Parse(os.Args[1:])
	if err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		err = errors.Join(err, errors.New("missing command upload|duplicate|stack|tool"))
	}

	if err != nil {
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
		log.Error(err.Error())
	}
	return err
}
