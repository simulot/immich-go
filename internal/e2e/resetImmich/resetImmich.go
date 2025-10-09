package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	e2eutils "github.com/simulot/immich-go/internal/e2e/e2eUtils"
)

func main() {
	err := run()
	if err != nil {
		slog.Error("exit with error", "error", err.Error())
		fmt.Println(err.Error())
		os.Exit(1)
	}
	slog.Info("immich reset: done")
}

func run() error {
	ctx := context.Background()
	force := flag.Bool("force", false, "force reset")
	flag.Parse()
	if !*force {
		fmt.Println("use -force to reset")
		return nil
	}
	composeFile, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("can't get current working directory: %w", err)
	}
	if len(flag.Args()) > 0 {
		composeFile = flag.Args()[0]
		slog.Info("path to docker compose", "file", composeFile)
	}
	ic, err := e2eutils.NewImmichController(composeFile)
	if err != nil {
		return fmt.Errorf("can't create immich controller: %w", err)
	}
	err = ic.ResetImmich(ctx)
	if err != nil {
		return err
	}
	err = ic.WaitAPI(ctx)
	if err != nil {
		return err
	}
	err = ic.WaitAPI(ctx)

	return nil
}
