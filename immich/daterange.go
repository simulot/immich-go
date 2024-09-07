package immich

import (
	"fmt"
	"time"
)

// DateRange represent the date range for capture date
// either:
// A day:    2022-01-01
// A month:  2022-01
// A year:   2022
// A range:  2022-01-01,2022-12-31
type DateRange struct {
	After, Before         time.Time // todo: make After and Before private
	day, month, year, set bool
}

func (dr DateRange) String() string {
	switch {
	case dr.day:
		return dr.After.Format("2006-01-02")
	case dr.month:
		return dr.After.Format("2006-01")
	case dr.year:
		return dr.After.Format("2006")
	default:
		return dr.After.Format("2006-01-02") + "," + dr.Before.AddDate(0, 0, -1).Format("2006-01-02")
	}
}

func (dr *DateRange) Set(s string) (err error) {
	dr.set = true
	switch len(s) {
	case 0:
		dr.Before = time.Date(999, 12, 31, 0, 0, 0, 0, time.UTC)
	case 4:
		dr.year = true
		dr.After, err = time.ParseInLocation("2006", s, time.UTC)
		if err == nil {
			dr.Before = dr.After.AddDate(1, 0, 0)
			return nil
		}
	case 7:
		dr.month = true
		dr.After, err = time.ParseInLocation("2006-01", s, time.UTC)
		if err == nil {
			dr.Before = dr.After.AddDate(0, 1, 0)
			return nil
		}
	case 10:
		dr.day = true
		dr.After, err = time.ParseInLocation("2006-01-02", s, time.UTC)
		if err == nil {
			dr.Before = dr.After.AddDate(0, 0, 1)
			return nil
		}
	case 21:
		dr.After, err = time.ParseInLocation("2006-01-02", s[:10], time.UTC)
		if err == nil {
			dr.Before, err = time.ParseInLocation("2006-01-02", s[11:], time.UTC)
			if err == nil {
				dr.Before = dr.Before.AddDate(0, 0, 1)
				return nil
			}
		}
	}
	dr.set = false
	return fmt.Errorf("invalid date range:%w", err)
}

func (dr DateRange) IsSet() bool { return dr.set }

func (dr DateRange) InRange(d time.Time) bool {
	if !dr.set || d.IsZero() {
		return true
	}
	//	--------------After----------d------------Before
	return (d.Compare(dr.After) >= 0 && dr.Before.Compare(d) > 0)
}

func (dr *DateRange) Type() string {
	return "date-range"
}
