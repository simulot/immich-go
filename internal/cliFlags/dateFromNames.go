package cliflags

import (
	"fmt"
	"strings"

	"github.com/simulot/immich-go/internal/tzone"
	"github.com/spf13/cobra"
)

type DateMethod string

const (
	DateMethodNone         DateMethod = "NONE"
	DateMethodName         DateMethod = "FILENAME"
	DateMethodEXIF         DateMethod = "EXIF"
	DateMethodNameThenExif DateMethod = "FILENAME-EXIF"
	DateMethodExifThenName DateMethod = "EXIF-FILENAME"
)

func (dm *DateMethod) Set(s string) error {
	s = strings.TrimSpace(strings.ToUpper(s))
	if s == "" {
		s = string(DateMethodNone)
	}
	switch DateMethod(s) {
	case DateMethodNone,
		DateMethodEXIF,
		DateMethodNameThenExif,
		DateMethodExifThenName,
		DateMethodName:
		*dm = DateMethod(s)
		return nil
	default:
		return fmt.Errorf("invalid DateMethod: %s, expecting NONE|FILENAME|EXIF|FILENAME-EXIF|EXIF-FILENAME", s)
	}
}

func (dm *DateMethod) Type() string {
	return "DateMethod"
}

func (dm *DateMethod) String() string {
	return string(*dm)
}

type DateHandlingFlags struct {
	Method           DateMethod
	FilenameTimeZone tzone.Timezone // set with global flag  --time-zone string
}

func AddDateHandlingFlags(cmd *cobra.Command, flags *DateHandlingFlags) {
	flags.Method = DateMethodNone // DateMethodExifThenName
	_ = flags.FilenameTimeZone.Set("Local")

	cmd.Flags().Var(&flags.Method, "capture-date-method", "Specify the method to determine the capture date when not provided in a sidecar file. Options: NONE (do not attempt to determine), FILENAME (extract from filename), EXIF (extract from EXIF metadata), FILENAME-EXIF (try filename first, then EXIF), EXIF-FILENAME (try EXIF first, then filename)")
	// cmd.Flags().Var(&flags.FilenameTimeZone, "filename-timezone", "Specify the timezone to use when detecting the date from the filename. Options: Local (use the system's local timezone), UTC (use UTC timezone), or a valid timezone name (e.g. America/New_York)")
}
