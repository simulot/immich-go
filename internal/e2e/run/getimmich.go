package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	e2eutils "github.com/simulot/immich-go/internal/e2e/e2eUtils"
)

func main() {
	err := run()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
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
		fmt.Println("Use docker compose file: ", composeFile)
	}
	ic := e2eutils.NewImmichController(composeFile)
	err = ic.ResetImmich(ctx)
	if err != nil {
		return err
	}
	return nil
}
