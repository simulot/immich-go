package convert

import "time"

const xmpTimeLayout = "2006-01-02T15:04:05.000-07:00"

func TimeStringToTime(t string, l *time.Location) (time.Time, error) {
	return time.ParseInLocation(xmpTimeLayout, t, l)
}

func TimeToString(t time.Time) string {
	return t.Format(xmpTimeLayout)
}
