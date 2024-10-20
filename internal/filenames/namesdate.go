package filenames

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var timeRe = regexp.MustCompile(`(19[89]\d|20\d\d)\D?(0\d|1[0-2])\D?([0-3]\d)\D{0,1}([01]\d|2[0-4])?\D?([0-5]\d)?\D?([0-5]\d)?`)

// TakeTimeFromPath takes the full path of a file and returns a time.Time value that is extracted
// from the given full path. At first it tries to extract from filename, then from each folder
// name (end to start), If no time is found - it will try to extract from the path itself as a
// last resort (e.g. /something/2024/06/06/file123.png).
func TakeTimeFromPath(fullpath string, tz *time.Location) time.Time {
	parts := strings.Split(fullpath, string(os.PathSeparator))

	for i := len(parts) - 1; i >= 0; i-- {
		if t := TakeTimeFromName(parts[i], tz); !t.IsZero() {
			return t
		}
	}

	return TakeTimeFromName(fullpath, tz)
}

// TakeTimeFromName takes the name of a file and returns a time.Time value that is extracted
// from the given file name. It uses the given Timezone to parse the time.
func TakeTimeFromName(s string, tz *time.Location) time.Time {
	timeSegments := timeRe.FindStringSubmatch(s)
	if len(timeSegments) < 4 {
		return time.Time{}
	}

	m := make([]int, 6)
	for i := 1; i < len(timeSegments); i++ {
		m[i-1], _ = strconv.Atoi(timeSegments[i])
	}
	t := time.Date(m[0], time.Month(m[1]), m[2], m[3], m[4], m[5], 0, tz)

	if t.Year() != m[0] || t.Month() != time.Month(m[1]) || t.Day() != m[2] ||
		t.Hour() != m[3] || t.Minute() != m[4] || t.Second() != m[5] {
		return time.Time{}
	}
	if time.Since(t) < -24*time.Hour {
		return time.Time{}
	}
	// Below is not needed as it is enforced by Regex
	// if t.Year() < 1980 {
	// 	continue
	// }
	return t
}
