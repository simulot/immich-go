package cliflags

import (
	"fmt"
	"strings"
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
