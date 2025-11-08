package cliflags

import (
	"fmt"
	"time"
)

// DateRange represent the date range for capture date

type DateRange struct {
	After, Before         time.Time // todo: make After and Before private
	day, month, year, set bool
	tz                    *time.Location
	s                     string
}

// InitDateRange initialize a DateRange with a string (for tests)
func InitDateRange(tz *time.Location, s string) DateRange {
	dr := DateRange{
		tz: tz,
	}
	_ = dr.Set(s)
	return dr
}

// IsSet returns whether the date range is set
func (dr DateRange) IsSet() bool { return dr.set }

func (dr DateRange) String() string {
	if dr.set {
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
	} else {
		return "unset"
	}
}

// MarshalJSON implements json.Marshaler
func (dr DateRange) MarshalJSON() ([]byte, error) {
	return []byte(`"` + dr.String() + `"`), nil
}

// UnmarshalJSON implements json.Unmarshaler
func (dr *DateRange) UnmarshalJSON(data []byte) error {
	if len(data) < 2 || data[0] != '"' || data[len(data)-1] != '"' {
		return fmt.Errorf("invalid JSON string for DateRange")
	}
	s := string(data[1 : len(data)-1])
	return dr.Set(s)
}

// MarshalYAML implements yaml.Marshaler
func (dr DateRange) MarshalYAML() (interface{}, error) {
	return dr.String(), nil
}

// UnmarshalYAML implements yaml.Unmarshaler
func (dr *DateRange) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	return dr.Set(s)
}

// MarshalText implements encoding.TextMarshaler
func (dr DateRange) MarshalText() ([]byte, error) {
	return []byte(dr.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler
func (dr *DateRange) UnmarshalText(data []byte) error {
	return dr.Set(string(data))
}

func (dr *DateRange) SetTZ(tz *time.Location) {
	dr.tz = tz
	if dr.set {
		_ = dr.Set(dr.s)
	}
}

// Implements the flags interface
// A day:    2022-01-01
// A month:  2022-01
// A year:   2022
// A range:  2022-01-01,2022-12-31
func (dr *DateRange) Set(s string) (err error) {
	if dr.tz == nil {
		dr.tz = time.Local
	}
	switch len(s) {
	case 4:
		dr.year = true
		dr.After, err = time.ParseInLocation("2006", s, dr.tz)
		if err != nil {
			return fmt.Errorf("invalid date range:%w", err)
		}
		dr.Before = dr.After.AddDate(1, 0, 0)
	case 7:
		dr.month = true
		dr.After, err = time.ParseInLocation("2006-01", s, dr.tz)
		if err != nil {
			return fmt.Errorf("invalid date range:%w", err)
		}
		dr.Before = dr.After.AddDate(0, 1, 0)
	case 10:
		dr.day = true
		dr.After, err = time.ParseInLocation("2006-01-02", s, dr.tz)
		if err != nil {
			return fmt.Errorf("invalid date range:%w", err)
		}
		dr.Before = dr.After.AddDate(0, 0, 1)
	case 21:
		dr.After, err = time.ParseInLocation("2006-01-02", s[:10], dr.tz)
		if err != nil {
			return fmt.Errorf("invalid date range:%w", err)
		}
		dr.Before, err = time.ParseInLocation("2006-01-02", s[11:], dr.tz)
		if err != nil {
			return fmt.Errorf("invalid date range:%w", err)
		}
		dr.Before = dr.Before.AddDate(0, 0, 1)
	default:
		dr.set = false
		return fmt.Errorf("invalid date range:%s", s)
	}

	if dr.Before.Before(dr.After) {
		dr.set = false
		return fmt.Errorf("invalid date range:%s", s)
	}
	dr.set = true
	dr.s = s
	return nil
}

// InRange checks if a given date is within the range
func (dr DateRange) InRange(d time.Time) bool {
	if !dr.set {
		return true
	}
	//	--------------After----------d------------Before
	return (d.Compare(dr.After) >= 0 && dr.Before.Compare(d) > 0)
}

func (dr DateRange) Type() string {
	return "date-range"
}
