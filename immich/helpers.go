package immich

import (
	"fmt"
	"io"
	"io/fs"
	"time"

	"github.com/rwcarlsen/goexif/exif"
)

// ExtractDateTaken extracts the date taken from the EXIF data using the goexif/exif package.
func ExtractDateTaken(fsys fs.FS, filePath string) (time.Time, error) {
	// Open the file
	file, err := fsys.Open(filePath)
	if err != nil {
		return time.Time{}, err
	}
	defer file.Close()

	// Decode the EXIF data
	x, err := exif.Decode(io.LimitReader(file, 64*1024))
	if err != nil && exif.IsCriticalError(err) {
		return time.Time{}, fmt.Errorf("can't get DateTaken: %w", err)
	}

	// Get the date taken from the EXIF data
	tm, err := x.DateTime()
	if err != nil {
		return time.Time{}, fmt.Errorf("can't get DateTaken: %w", err)
	}
	t := time.Date(tm.Year(), tm.Month(), tm.Day(), tm.Hour(), tm.Minute(), tm.Second(), tm.Nanosecond(), time.Local)
	return t, nil
}

type DateRange struct {
	After, Before         time.Time
	day, month, year, set bool
}

func (dr DateRange) String() string {
	if dr.day {
		return dr.After.Format("2006-01-02")
	} else if dr.month {
		return dr.After.Format("2006-01")
	} else if dr.year {
		return dr.After.Format("2006")
	}
	return dr.After.Format("2006-01-02") + "," + dr.Before.AddDate(0, 0, -1).Format("2006-01-02")
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
			return
		}
	case 7:
		dr.month = true
		dr.After, err = time.ParseInLocation("2006-01", s, time.UTC)
		if err == nil {
			dr.Before = dr.After.AddDate(0, 1, 0)
			return
		}
	case 10:
		dr.day = true
		dr.After, err = time.ParseInLocation("2006-01-02", s, time.UTC)
		if err == nil {
			dr.Before = dr.After.AddDate(0, 0, 1)
			return
		}
	case 21:
		dr.After, err = time.ParseInLocation("2006-01-02", s[:10], time.UTC)
		if err == nil {
			dr.Before, err = time.ParseInLocation("2006-01-02", s[11:], time.UTC)
			if err == nil {
				dr.Before = dr.Before.AddDate(0, 0, 1)
				return
			}
		}
	}
	dr.set = false
	return fmt.Errorf("invalid date range:%w", err)
}

func (dr DateRange) InRange(d time.Time) bool {
	//	--------------After----------d------------Before
	return !dr.set || (d.Compare(dr.After) >= 0 && dr.Before.Compare(d) > 0)
}
