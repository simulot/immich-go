package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	e2eutils "github.com/simulot/immich-go/internal/e2e/e2eUtils"
)

func main() {
	err := run()
	if err != nil {
		slog.Error("can't get immich", "error", err.Error())
		os.Exit(1)
	}
	slog.Info("immich deploy: done")
}

func run() error {
	ctx := context.Background()

	if len(os.Args) < 2 {
		return errors.New("missing path to immich installation")
	}

	curWD, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("can't get current working directory: %w", err)
	}
	defer func() {
		_ = os.Chdir(curWD)
	}()

	ictl, err := e2eutils.NewImmichController(os.Args[1])
	if err != nil {
		return fmt.Errorf("can't create immich controller: %w", err)
	}

	err = ictl.DeployImmich(ctx)
	if err != nil {
		return err
	}

	return err
}
