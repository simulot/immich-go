package cliflags

import (
	"fmt"
	"strings"
)

type RawJPGFlag int

const (
	RawJPGNothing  RawJPGFlag = iota
	RawJPGKeepRaw             // Keep only raw files
	RawJPGKeepJPG             // Keep only JPEG files
	RawJPGStackRaw            // Stack raw and JPEG files, with the raw file as the cover
	RawJPGStackJPG            // Stack raw and JPEG files, with the JPEG file as the cover
)

func (r *RawJPGFlag) Set(value string) error {
	switch strings.ToLower(value) {
	case "":
		*r = RawJPGNothing
	case "keepraw":
		*r = RawJPGKeepRaw
	case "keepjpg":
		*r = RawJPGKeepJPG
	case "stackcoverraw":
		*r = RawJPGStackRaw
	case "stackcoverjpg":
		*r = RawJPGStackJPG
	default:
		return fmt.Errorf("invalid value %q for RawJPGFlag", value)
	}
	return nil
}

func (r RawJPGFlag) String() string {
	switch r {
	case RawJPGNothing:
		return ""
	case RawJPGKeepRaw:
		return "KeepRaw"
	case RawJPGKeepJPG:
		return "KeepJPG"
	case RawJPGStackRaw:
		return "StackCoverRaw"
	case RawJPGStackJPG:
		return "StackCoverJPG"
	default:
		return "Unknown"
	}
}

func (r RawJPGFlag) Type() string {
	return "RawJPGFlag"
}
