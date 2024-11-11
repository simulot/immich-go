package metadata

import (
	"fmt"
	"time"

	etool "github.com/barasher/go-exiftool"
)

type ExifTool struct {
	flags *ExifToolFlags
	eTool *etool.Exiftool
}

func NewExifTool(flags *ExifToolFlags) (*ExifTool, error) {
	opts := []func(*etool.Exiftool) error{
		etool.Charset("filename=utf8"),
		etool.CoordFormant("%+.7f"),
	}

	if flags != nil {
		if flags.ExifPath != "" {
			opts = append(opts, etool.SetExiftoolBinaryPath(flags.ExifPath))
		}
	}

	tool, err := etool.NewExiftool(opts...)
	if err != nil {
		return nil, err
	}
	et := ExifTool{
		eTool: tool,
		flags: flags,
	}
	return &et, nil
}

func (et *ExifTool) Close() error {
	return et.eTool.Close()
}

var dateKeys = []struct {
	key    string
	format string
	isUTC  bool
}{
	{"GPSDateTime", "2006:01:02 15:04:05Z", true},      // 2023:10:06 06:29:56Z
	{"DateTimeUTC", "2006:01:02 15:04:05", false},      // 2016:11:19 20:30:52
	{"CreateDate", "2006:01:02 15:04:05", false},       // 2023:10:06 08:30:00
	{"ModifyDate", "2006:01:02 15:04:05", false},       // 2016:11:19 20:30:52
	{"MediaModifyDate", "2006:01:02 15:04:05", false},  // 2016:11:19 20:30:52
	{"DateTimeOriginal", "2006:01:02 15:04:05", false}, // 2023:10:06 08:30:00
}

// GetMetadata returns the metadata of the file. The date of capture is searched in the preferred tags first.
// missing tags or tags  with incorrect dates are skipped
//
// TODO: make a better use of time offset taken on the exif fields
// ```
// Modify Date                     : 2023:10:06 08:30:00
// Date/Time Original              : 2023:10:06 08:30:00
// Create Date                     : 2023:10:06 08:30:00
// Offset Time                     : +02:00
// Offset Time Original            : +02:00
// Offset Time Digitized           : +02:00
// Sub Sec Time                    : 139
// Sub Sec Time Original           : 139
// Sub Sec Time Digitized          : 139
// GPS Time Stamp                  : 06:29:56
// GPS Date Stamp                  : 2023:10:06
// Profile Date Time               : 2023:03:09 10:57:00
// Create Date                     : 2023:10:06 08:30:00.139+02:00
// Date/Time Original              : 2023:10:06 08:30:00.139+02:00
// Modify Date                     : 2023:10:06 08:30:00.139+02:00
// GPS Date/Time                   : 2023:10:06 06:29:56Z
// ```

func (et *ExifTool) ReadMetaData(fileName string) (*Metadata, error) {
	ms := et.eTool.ExtractMetadata(fileName)
	if len(ms) != 1 {
		return nil, fmt.Errorf("cant extract metadata from file '%s': unexpected exif-tool result", fileName)
	}
	m := ms[0]
	if m.Err != nil {
		return nil, fmt.Errorf("cant extract metadata from file '%s': %w", fileName, m.Err)
	}
	md := Metadata{}

	md.Latitude, _ = m.GetFloat("GPSLatitude")
	md.Longitude, _ = m.GetFloat("GPSLongitude")

	// get the date of capture using preferred exif tag
	for _, dk := range dateKeys {
		if s, err := m.GetString(dk.key); err == nil {
			tz := et.flags.Timezone.Location()
			if dk.isUTC {
				tz = time.UTC
			}
			t, err := time.ParseInLocation(dk.format, s, tz)
			if err == nil {
				if t.IsZero() || t.Before(time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)) || t.After(time.Now().AddDate(0, 0, 365*10)) {
					continue
				}
				md.DateTaken = t
				break
			}
		}
	}
	return &md, nil
}
