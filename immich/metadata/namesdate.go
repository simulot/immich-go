package metadata

import (
	"regexp"
	"strconv"
	"time"

	"github.com/simulot/immich-go/helpers/tzone"
)

// TakeTimeFromName extracts time components from the given name string and returns a time.Time value.
// The name string is expected to contain digits representing year, month, day, hour, minute, and second in local.
// Note: Pixel phone names photos with the UTC time
//
// Return a time.Time value created using the parsed time components.
// The location is set to time.UTC for consistency.
// Return the value time.Time{} when there isn't any date in the name, or if the date is invalid like 2023-02-30 20:65:00

var (
	guessTimePattern = regexp.MustCompile(`(\d{4})\D?(\d\d)\D?(\d\d)\D?(\d\d)?\D?(\d\d)?\D?(\d\d)?`)
	nexusBurstRE     = regexp.MustCompile(`^\d{5}IMG_\d{5}_BURST(\d{14})(_COVER)?\..{3}$`)
)

func TakeTimeFromName(name string) time.Time {
	local, err := tzone.Local()
	if err != nil {
		panic(err)
	}

	// check for known exceptions...
	mm := nexusBurstRE.FindStringSubmatch(name)
	if len(mm) > 2 {
		name = mm[1]
	}
	mm = guessTimePattern.FindStringSubmatch(name)
	m := [7]int{}
	if len(mm) >= 4 {
		for i := range mm {
			if i > 0 {
				m[i-1], _ = strconv.Atoi(mm[i])
			}
		}
		t := time.Date(m[0], time.Month(m[1]), m[2], m[3], m[4], m[5], 0, time.UTC)
		if t.Year() != m[0] || t.Month() != time.Month(m[1]) || t.Day() != m[2] ||
			t.Hour() != m[3] || t.Minute() != m[4] || t.Second() != m[5] {
			// Date is invalid, return an error or default time value
			return time.Time{}
		}
		if time.Since(t) < -24*time.Hour {
			// Discard dates in the future
			return time.Time{}
		}
		return t.In(local)
	}
	return time.Time{}
}
