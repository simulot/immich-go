package immich

import (
	"bytes"
	"encoding/binary"
	"io"
	"time"
)

/*

The creation date is stored into MP4 metadata in the mvhd atom.
In baste case, the atom is found near to the file begin.
The reading is stopped.

In worst cases, the file must be completely read to get the capture time.
*/

/*
GuessTakeTimeFromMP4 return the date of capture from a reader structured as a MP4

The read is stopped as soon the date is found. Returned values are date of capture and nil

When the date isn't found, the file is completely read. Returned values are time.Time{} and nil

*/

func TakeTimeFromMP4(r io.Reader) (time.Time, error) {

	// mvhd pattern
	pattern := []byte{'m', 'v', 'h', 'd'}
	// atom max len
	structLen := 112
	pos := 0

	// Create a buffer to hold the chunk of data
	bufferSize := 16 * 1024
	buffer := make([]byte, bufferSize)

	var bytesRead int
	var offset int64
	var err error

	for {
		// Read a chunk of data into the buffer
		bytesRead, err = r.Read(buffer)
		if err != nil && err != io.EOF {
			return time.Time{}, err
		}

		// Search for the pattern within the buffer
		index := bytes.Index(buffer[:bytesRead], pattern)
		if index != -1 {
			atom, err := decodeMvhdAtom(buffer[index:])
			if err != nil {
				return time.Time{}, err
			}
			return atom.CreationTime, nil
		}

		// Slide the buffer by the pattern length minus one
		offset += int64(bytesRead - structLen + 1)

		// Check if end of file is reached
		if err == io.EOF {
			break
		}

		// Move the remaining bytes of the current buffer to the beginning
		copy(buffer, buffer[structLen-1:bytesRead])
		pos += bytesRead
	}

	// at file end, no date
	return time.Time{}, nil
}

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

type MvhdAtom struct {
	Marker           []byte //4 bytes
	Version          uint8
	Flags            []byte // 3 bytes
	CreationTime     time.Time
	ModificationTime time.Time
	// ignored fields:
	// Timescale        uint32
	// Duration         uint32
	// Rate             float32
	// Volume           float32
	// Matrix           [9]int32
	// NextTrackID      uint32
}

func decodeMvhdAtom(b []byte) (*MvhdAtom, error) {
	r := &sliceReader{Reader: bytes.NewReader(b)}

	a := &MvhdAtom{}

	// Read the mvhd marker (4 bytes)
	a.Marker, _ = r.ReadSlice(4)

	// Read the mvhd version (1 byte)
	a.Version, _ = r.ReadByte()

	// Read the mvhd flags (3 bytes)
	a.Flags, _ = r.ReadSlice(3)

	if a.Version == 0 {
		// Read the creation time (4 bytes)
		b, _ := r.ReadSlice(4)
		a.ModificationTime = convertTime32(binary.BigEndian.Uint32(b))
		b, _ = r.ReadSlice(4)
		a.CreationTime = convertTime32(binary.BigEndian.Uint32(b))

	} else {
		// Read the creation time (4 bytes)
		b, _ := r.ReadSlice(8)
		a.ModificationTime = convertTime64(binary.BigEndian.Uint64(b))

		b, _ = r.ReadSlice(4)
		a.CreationTime = convertTime64(binary.BigEndian.Uint64(b))
	}

	return a, nil
}

func convertTime32(timestamp uint32) time.Time {
	return time.Unix(int64(timestamp)-int64(2082844800), 0).Local()
}
func convertTime64(timestamp uint64) time.Time {
	// Unix epoch starts on January 1, 1970, subtracting the number of seconds from January 1, 1904 to January 1, 1970.
	epochOffset := int64(2082844800)

	// Convert the creation time to Unix timestamp
	unixTimestamp := int64(timestamp>>32) - epochOffset

	// Convert the Unix timestamp to time.Time
	return time.Unix(unixTimestamp, 0).Local()
}

type sliceReader struct {
	*bytes.Reader
}

func (r *sliceReader) ReadSlice(l int) ([]byte, error) {
	b := make([]byte, l)
	_, err := r.Read(b)
	return b, err
}
