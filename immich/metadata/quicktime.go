package metadata

import (
	"bytes"
	"encoding/binary"
	"time"
)

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

		b, _ = r.ReadSlice(8)
		a.CreationTime = convertTime64(binary.BigEndian.Uint64(b))
	}

	return a, nil
}

func convertTime32(timestamp uint32) time.Time {
	return time.Unix(int64(timestamp)-int64(2082844800), 0)
}
func convertTime64(timestamp uint64) time.Time {
	// Unix epoch starts on January 1, 1970, subtracting the number of seconds from January 1, 1904 to January 1, 1970.
	epochOffset := int64(2082844800)

	// Convert the creation time to Unix timestamp
	unixTimestamp := int64(timestamp>>32) - epochOffset

	// Convert the Unix timestamp to time.Time
	return time.Unix(unixTimestamp, 0)
}

type sliceReader struct {
	*bytes.Reader
}

func (r *sliceReader) ReadSlice(l int) ([]byte, error) {
	b := make([]byte, l)
	_, err := r.Read(b)
	return b, err
}
