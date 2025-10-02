package cliflags

import (
	"fmt"
	"strconv"
	"strings"
)

type OnErrorsFlag int

const (
	OnErrorsStop OnErrorsFlag = iota
	OnErrorsStopAfter
	OnErrorsNeverStop = -1
)

func (f OnErrorsFlag) String() string {
	switch {
	case f == OnErrorsStop:
		return "stop"
	case f == OnErrorsNeverStop:
		return "continue"
	case f >= OnErrorsStopAfter:
		return fmt.Sprintf("stop after %d errors", f)
	default:
		return "unknown"
	}
}

func (f *OnErrorsFlag) Set(value string) error {
	switch strings.ToLower(value) {
	case "stop":
		*f = OnErrorsStop
	case "continue":
		*f = OnErrorsNeverStop
	default:
		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid value for on-server-errors: %s", value)
		}
		*f = OnErrorsFlag(n)
	}
	return nil
}

func (OnErrorsFlag) Type() string {
	return "OnErrorsFlag"
}
