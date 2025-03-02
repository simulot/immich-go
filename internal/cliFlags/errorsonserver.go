package cliflags

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

type OnServerErrorsFlag int

const (
	OnServerErrorsStop OnServerErrorsFlag = iota
	OnServerErrorsStopAfter
	OnServerErrorsNeverStop = -1
)

func (f OnServerErrorsFlag) String() string {
	switch {
	case f == OnServerErrorsStop:
		return "stop"
	case f == OnServerErrorsNeverStop:
		return "continue"
	case f >= OnServerErrorsStopAfter:
		return fmt.Sprintf("stop after %d errors", f)
	default:
		return "unknown"
	}
}

func (f *OnServerErrorsFlag) Set(value string) error {
	switch strings.ToLower(value) {
	case "stop":
		*f = OnServerErrorsStop
	case "continue":
		*f = OnServerErrorsNeverStop
	default:
		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid value for on-server-errors: %s", value)
		}
		*f = OnServerErrorsFlag(n)
	}
	return nil
}

func (OnServerErrorsFlag) Type() string {
	return "OnServerErrorsFlag"
}

func AddOnServerErrorsFlag(cmd *cobra.Command, flags *OnServerErrorsFlag) {
	cmd.Flags().Var(flags, "on-server-errors", "What to do when server errors occur. (options: stop (default), continue, <n>)")
}
