package exif

import (
	"github.com/simulot/immich-go/internal/tzone"
	"github.com/spf13/cobra"
)

type ExifToolFlags struct {
	UseExifTool bool
	ExifPath    string
	Timezone    tzone.Timezone
}

func AddExifToolFlags(cmd *cobra.Command, flags *ExifToolFlags) {
	flags.Timezone.Set("Local")
	cmd.Flags().BoolVar(&flags.UseExifTool, "exiftool-enabled", false, "Enable the use of the external 'exiftool' program (if installed and available in the system path) to extract EXIF metadata")
	cmd.Flags().StringVar(&flags.ExifPath, "exiftool-path", "", "Path to the ExifTool executable (default: search in system's PATH)")
	cmd.Flags().Var(&flags.Timezone, "exiftool-timezone", "Timezone to use when parsing exif timestamps without timezone Options: LOCAL (use the system's local timezone), UTC (use UTC timezone), or a valid timezone name (e.g. America/New_York)")
}
