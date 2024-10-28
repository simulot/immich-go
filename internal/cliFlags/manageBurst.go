package cliflags

import (
	"fmt"
	"strings"
)

type BurstFlag int

const (
	BurstNothing  BurstFlag = iota
	BurstStack              // Stack burst photos
	BurstkKeepRaw           // Stack burst, keep raw photos when when have JPEG and raw
	BurstKeepJPEG           // Stack burst, keep JPEG photos when when have JPEG and raw
)

func (b *BurstFlag) Set(value string) error {
	switch strings.ToLower(value) {
	case "":
		*b = BurstNothing
	case "stack":
		*b = BurstStack
	case "stackkeepraw":
		*b = BurstkKeepRaw
	case "stackkeepjpeg":
		*b = BurstKeepJPEG
	default:
		return fmt.Errorf("invalid value %q for BurstFlag", value)
	}
	return nil
}

func (b BurstFlag) String() string {
	switch b {
	case BurstNothing:
		return ""
	case BurstStack:
		return "Stack"
	case BurstkKeepRaw:
		return "StackKeepRaw"
	case BurstKeepJPEG:
		return "StackKeepJPEG"
	default:
		return "Unknown"
	}
}

func (b BurstFlag) Type() string {
	return "BurstFlag"
}
