package xmpsidecar

import "time"

/*
exif:DateTimeOriginalDateInternal

EXIF tags 36867, 0x9003 (primary) and 37521, 0x9291 (subseconds). Date and time when original image was generated, in ISO 8601 format. Includes the EXIF
SubSecTimeOriginal data.

Note that EXIF date-time values have no time zone information.


exif:GPSTimeStampDateInternalGPS tag 29 (date), 0x1D, and, and GPS tag 7 (time), 0x07.

Time stamp of GPS data, in Coordinated Universal

Time.

The GPSDateStamp tag is new in EXIF 2.2. The GPS
timestamp in EXIF 2.1 does not include a date. If not
present, the date component for the XMP should be
taken from exif:DateTimeOriginal, or if that is also
lacking from exif:DateTimeDigitized. If no date is
available, do not write exif:GPSTimeStamp to XMP.

*/

/*
Date
A date-time value which is represented using a subset of ISO RFC 8601 formatting, as described in
http://www.w3.org/TR/Note-datetime.html. The following formats are supported:
	YYYY
	YYYY-MM
	YYYY-MM-DD
	YYYY-MM-DDThh:mmTZD
	YYYY-MM-DDThh:mm:ssTZD
	YYYY-MM-DDThh:mm:ss.sTZD
	YYYY = four-digit year
	MM = two-digit month (01=January)
	DD = two-digit day of month (01 through 31)
	hh = two digits of hour (00 through 23)
	mm = two digits of minute (00 through 59)
	ss = two digits of second (00 through 59)
	s = one or more digits representing a decimal fraction of a second
	TZD = time zone designator (Z or +hh:mm or -hh:mm)

The time zone designator is optional in XMP. When not present, the time zone is unknown, and software
should not assume anything about the missing time zone.

It is recommended, when working with local times, that you use a time zone designator of +hh:mm or
-hh:mm instead of Z, to aid human readability. For example, if you know a file was saved at noon on
October 23 a timestamp of 2004-10-23T12:00:00-06:00 is more understandable than
2004-10-23T18:00:00Z.
*/

const xmpTimeLayout = "2006-01-02T15:04:05Z"

func TimeStringToTime(t string, l *time.Location) (time.Time, error) {
	return time.ParseInLocation(xmpTimeLayout, t, l)
}

func TimeToString(t time.Time) string {
	return t.Format(xmpTimeLayout)
}
