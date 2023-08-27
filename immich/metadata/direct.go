package metadata

/*
type MetaData struct {
	DateTaken                     time.Time
	Latitude, Longitude, Altitude float64
}

func GetFileMetaData(fsys fs.FS, name string) (*MetaData, error) {
	f, err := fsys.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return GetMetaData(f)
}

// GetMetaData makes its best efforts to get the date of capture based on
// - if the name matches a at least 4 digits for the year, 2 for month, 2 for day, in this order.
//   It takes the hour, minute, second when present. Very fast
//
// - file content if the file includes some metadata, need read a part of the file
//
//

func GetMetaData(file io.Reader) (*MetaData, error) {
	meta := MetaData{}
	ext := strings.ToLower(path.Ext(l.FileName))
	var err error
	switch ext {
	case ".heic", ".heif":
		l.dateTaken, err = l.readHEIFDateTaken()
	case ".jpg", ".jpeg":
		l.dateTaken, err = l.readExifDateTaken()
	case ".mp4", ".mov":
		l.dateTaken, err = l.readMP4DateTaken()
	default:
		err = fmt.Errorf("can't determine the taken date from this file: %q", l.FileName)
	}
	return l.dateTaken, err
}
*/
// // readExifDateTaken pase the file for Exif DateTaken
// func (l *LocalAssetFile) readExifDateTaken() (time.Time, error) {

// 	// Open the file
// 	r, err := l.partialSourceReader()

// 	if err != nil {
// 		return time.Time{}, err
// 	}

// 	// Decode the EXIF data
// 	x, err := exif.Decode(r)
// 	if err != nil && exif.IsCriticalError(err) {
// 		if errors.Is(err, io.EOF) {
// 			return time.Time{}, nil
// 		}
// 		return time.Time{}, fmt.Errorf("can't get DateTaken: %w", err)
// 	}

// 	// Get the date taken from the EXIF data
// 	tm, err := x.DateTime()
// 	if err != nil {
// 		return time.Time{}, fmt.Errorf("can't get DateTaken: %w", err)
// 	}
// 	t := time.Date(tm.Year(), tm.Month(), tm.Day(), tm.Hour(), tm.Minute(), tm.Second(), tm.Nanosecond(), time.Local)
// 	return t, nil
// }

// // readHEIFDateTaken locate the Exif part and return the date of capture
// func (l *LocalAssetFile) readHEIFDateTaken() (time.Time, error) {
// 	// Open the file
// 	r, err := l.partialSourceReader()

// 	if err != nil {
// 		return time.Time{}, err
// 	}

// 	r2, err := seekReaderAtPattern(r, []byte{0x45, 0x78, 0x69, 0x66, 0, 0, 0x4d, 0x4d})
// 	if err != nil {
// 		return time.Time{}, err
// 	}

// 	filler := make([]byte, 6)
// 	r2.Read(filler)

// 	// Decode the EXIF data
// 	x, err := exif.Decode(r2)
// 	if err != nil && exif.IsCriticalError(err) {
// 		return time.Time{}, fmt.Errorf("can't get DateTaken: %w", err)
// 	}
// 	// Get the date taken from the EXIF data
// 	tm, err := x.DateTime()
// 	if err != nil {
// 		return time.Time{}, fmt.Errorf("can't get DateTaken: %w", err)
// 	}
// 	t := time.Date(tm.Year(), tm.Month(), tm.Day(), tm.Hour(), tm.Minute(), tm.Second(), tm.Nanosecond(), time.Local)
// 	return t, nil
// }

// // readMP4DateTaken locate the mvhd atom and decode the date of capture
// func (l *LocalAssetFile) readMP4DateTaken() (time.Time, error) {
// 	// Open the file
// 	r, err := l.partialSourceReader()

// 	if err != nil {
// 		return time.Time{}, err
// 	}

// 	b, err := searchPattern(r, []byte{'m', 'v', 'h', 'd'}, 60)
// 	if err != nil {
// 		return time.Time{}, err
// 	}
// 	atom, err := decodeMvhdAtom(b)
// 	if err != nil {
// 		return time.Time{}, err
// 	}
// 	return atom.CreationTime, nil
// }

/*
The mvhd atom contains metadata and information about the entire movie or presentation, such as its duration,
time scale, preferred playback rate, and more.

Here are some of the main attributes found in the mvhd atom:

- Timescale: This value indicates the time scale for the media presentation,
   which represents the number of time units per second. It allows for accurate timing of media content in the file.

- Duration: The duration is the total time the movie or presentation lasts,
	expressed in the time scale units defined in the file.

-  Preferred Rate: The preferred rate is the intended playback rate for the movie.
	It can be used to set the default playback speed when the media is played.

- Preferred Volume: The preferred volume specifies the default audio volume for the media playback.

- Matrix Structure: The mvhd atom may contain a matrix structure
		that defines transformations to be applied when rendering the video content, such as scaling or rotation.

-  Creation and Modification Time: The mvhd atom also stores the creation time and modification time
	of the movie or presentation.

In total, the minimum size of the mvhd atom is 108 bytes (version 0) or 112 bytes (version 1).
If any of the optional fields are present, the size of the atom would increase accordingly.
*/

// type MvhdAtom struct {
// 	Marker           []byte //4 bytes
// 	Version          uint8
// 	Flags            []byte // 3 bytes
// 	CreationTime     time.Time
// 	ModificationTime time.Time
// 	// ignored fields:
// 	// Timescale        uint32
// 	// Duration         uint32
// 	// Rate             float32
// 	// Volume           float32
// 	// Matrix           [9]int32
// 	// NextTrackID      uint32
// }

// func decodeMvhdAtom(b []byte) (*MvhdAtom, error) {
// 	r := &sliceReader{Reader: bytes.NewReader(b)}

// 	a := &MvhdAtom{}

// 	// Read the mvhd marker (4 bytes)
// 	a.Marker, _ = r.ReadSlice(4)

// 	// Read the mvhd version (1 byte)
// 	a.Version, _ = r.ReadByte()

// 	// Read the mvhd flags (3 bytes)
// 	a.Flags, _ = r.ReadSlice(3)

// 	if a.Version == 0 {
// 		// Read the creation time (4 bytes)
// 		b, _ := r.ReadSlice(4)
// 		a.ModificationTime = convertTime32(binary.BigEndian.Uint32(b))
// 		b, _ = r.ReadSlice(4)
// 		a.CreationTime = convertTime32(binary.BigEndian.Uint32(b))

// 	} else {
// 		// Read the creation time (4 bytes)
// 		b, _ := r.ReadSlice(8)
// 		a.ModificationTime = convertTime64(binary.BigEndian.Uint64(b))

// 		b, _ = r.ReadSlice(8)
// 		a.CreationTime = convertTime64(binary.BigEndian.Uint64(b))
// 	}

// 	return a, nil
// }

// func convertTime32(timestamp uint32) time.Time {
// 	return time.Unix(int64(timestamp)-int64(2082844800), 0).Local()
// }
// func convertTime64(timestamp uint64) time.Time {
// 	// Unix epoch starts on January 1, 1970, subtracting the number of seconds from January 1, 1904 to January 1, 1970.
// 	epochOffset := int64(2082844800)

// 	// Convert the creation time to Unix timestamp
// 	unixTimestamp := int64(timestamp>>32) - epochOffset

// 	// Convert the Unix timestamp to time.Time
// 	return time.Unix(unixTimestamp, 0).Local()
// }

// type sliceReader struct {
// 	*bytes.Reader
// }

// func (r *sliceReader) ReadSlice(l int) ([]byte, error) {
// 	b := make([]byte, l)
// 	_, err := r.Read(b)
// 	return b, err
// }
