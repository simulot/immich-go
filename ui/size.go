package ui

import (
	"fmt"
	"math"
)

func FormatBytes(s int64) string {
	suffixes := []string{"B", "KB", "MB", "GB"}
	bytes := float64(s)
	base := 1024.0
	if bytes < base {
		return fmt.Sprintf("%.0f %s", bytes, suffixes[0])
	}
	exp := int64(0)
	for bytes >= base && exp < int64(len(suffixes)-1) {
		bytes /= base
		exp++
	}
	roundedSize := math.Round(bytes*10) / 10
	return fmt.Sprintf("%.1f %s", roundedSize, suffixes[exp])
}
