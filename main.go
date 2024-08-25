package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"

	"github.com/simulot/immich-go/cmd/duplicate"
	"github.com/simulot/immich-go/cmd/metadata"
	"github.com/simulot/immich-go/cmd/stack"
	"github.com/simulot/immich-go/cmd/tool"
	"github.com/simulot/immich-go/cmd/upload"
	"github.com/spf13/cobra"
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

func printVersion() {
	fmt.Printf("immich-go  %s, commit %s, built at %s\n", version, getCommitInfo(), date)
}

func main() {
	var err error

	// Create a context with cancel function to gracefully handle Ctrl+C events
	ctx, cancel := context.WithCancelCause(context.Background())

	// Handle Ctrl+C signal (SIGINT)
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt)

	go func() {
		<-signalChannel
		fmt.Println("\nCtrl+C received. Shutting down...")
		cancel(errors.New("Ctrl+C received")) // Cancel the context when Ctrl+C is received
	}()

	select {
	case <-ctx.Done():
		err = ctx.Err()
	default:
		err = Run(ctx)
	}
	if err != nil {
		if e := context.Cause(ctx); e != nil {
			err = e
		}
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func Run(ctx context.Context) error {
	printVersion()

	rootCmd := &cobra.Command{
		Use:   "immich-go",
		Short: "Immich-go is a command line application to interact with the Immich application using its API",
		Long:  `An alternative to the immich-CLI command that doesn't depend on nodejs installation. It tries its best for importing google photos takeout archives.`,
		// Run: func(cmd *cobra.Command, args []string) {
		// 	// Do Stuff Here
		// },
	}

	rootCmd.PersistentFlags().Add

	fmt.Println(app.Banner.String())

	/*
		app := cmd.ImmichServerFlags{
			Log:    slog.New(humane.NewHandler(os.Stdout, &humane.Options{Level: slog.LevelInfo})),
			Banner: ui.NewBanner(version, commit, date),
		}
		fs := flag.NewFlagSet("main", flag.ExitOnError)
		fs.BoolFunc("version", "Get immich-go version", func(s string) error {
			printVersion()
			os.Exit(0)
			return nil
		})
	*/
	app.InitSharedFlags()
	app.SetFlags(fs)

	err := fs.Parse(os.Args[1:])
	if err != nil {
		app.Log.Error(err.Error())
		return err
	}

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
