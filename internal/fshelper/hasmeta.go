package fshelper

import (
	"runtime"
	"strings"
)

// hasMeta reports whether path contains any of the magic characters
// recognized by Match.
// shamelessly copied from stdlib/os
func HasMagic(path string) bool {
	magicChars := `*?[`
	if runtime.GOOS != "windows" {
		magicChars = `*?[\`
	}
	return strings.ContainsAny(path, magicChars)
}
