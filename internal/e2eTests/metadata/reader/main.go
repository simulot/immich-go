package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/simulot/immich-go/internal/exif"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: reader <file>")
		return
	}
	err := run(os.Args[1])
	if err != nil {
		fmt.Println(err)
	}
}

func run(file string) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	m, err := exif.MetadataFromDirectRead(f, file, time.Local)
	if err != nil {
		return err
	}

	slog.Info("Metadata", "m", m)

	return nil
}
