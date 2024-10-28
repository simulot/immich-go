package cliflags

import (
	"fmt"
	"strings"
)

type HeicJpgFlag int

const (
	HeicJpgNothing   HeicJpgFlag = iota
	HeicJpgKeepHeic              // Keep only HEIC files
	HeicJpgKeepJPG               // Keep only JPEG files
	HeicJpgStackHeic             // Stack HEIC and JPEG files, with the HEIC file as the cover
	HeicJpgStackJPG              // Stack HEIC and JPEG files, with the JPEG file as the cover
)

func (h *HeicJpgFlag) Set(value string) error {
	switch strings.ToLower(value) {
	case "":
		*h = HeicJpgNothing
	case "keepheic":
		*h = HeicJpgKeepHeic
	case "keepjpg":
		*h = HeicJpgKeepJPG
	case "stackcoverheic":
		*h = HeicJpgStackHeic
	case "stackcoverjpg":
		*h = HeicJpgStackJPG
	default:
		return fmt.Errorf("invalid value %q for HeicJpgFlag", value)
	}
	return nil
}

func (h HeicJpgFlag) String() string {
	switch h {
	case HeicJpgNothing:
		return ""
	case HeicJpgKeepHeic:
		return "KeepHeic"
	case HeicJpgKeepJPG:
		return "KeepJPG"
	case HeicJpgStackHeic:
		return "StackCoverHeic"
	case HeicJpgStackJPG:
		return "StackCoverJPG"
	default:
		return "Unknown"
	}
}

func (h HeicJpgFlag) Type() string {
	return "HeicJpgFlag"
}
