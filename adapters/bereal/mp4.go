package bereal

import (
	"encoding/binary"
	"io"
)

// maxMoovSize bounds the amount of memory used to read the 'moov' box of a video.
// The box holds the track description only; real files weigh a few kilobytes.
const maxMoovSize = 64 << 20

// hasAudioTrack reports whether the ISO base media file (MP4, MOV) read from r holds
// an audio track. It walks the top-level boxes until the 'moov' box, then looks for a
// 'trak/mdia/hdlr' box whose handler type is 'soun'. Media data is skipped, never read.
//
// The result is false when the file can't be parsed, so callers must treat false as
// "no audio found", not as a proof that the file has no sound.
func hasAudioTrack(r io.ReadSeeker) bool {
	end, err := r.Seek(0, io.SeekEnd)
	if err != nil {
		return false
	}
	if _, err = r.Seek(0, io.SeekStart); err != nil {
		return false
	}

	var pos int64
	for pos+8 <= end {
		var hdr [16]byte
		if _, err = io.ReadFull(r, hdr[:8]); err != nil {
			return false
		}
		size := int64(binary.BigEndian.Uint32(hdr[0:4]))
		boxType := string(hdr[4:8])
		hdrLen := int64(8)

		switch size {
		case 0: // the box extends to the end of the file
			size = end - pos
		case 1: // 64 bits size follows the type
			if _, err = io.ReadFull(r, hdr[8:16]); err != nil {
				return false
			}
			size = int64(binary.BigEndian.Uint64(hdr[8:16]))
			hdrLen = 16
		}
		if size < hdrLen || pos+size > end {
			return false
		}

		if boxType == "moov" {
			payload := size - hdrLen
			if payload > maxMoovSize {
				return false
			}
			buf := make([]byte, payload)
			if _, err = io.ReadFull(r, buf); err != nil {
				return false
			}
			return moovHasAudio(buf)
		}

		pos += size
		if _, err = r.Seek(pos, io.SeekStart); err != nil {
			return false
		}
	}
	return false
}

// moovHasAudio looks for an audio track in the payload of a 'moov' box.
func moovHasAudio(moov []byte) bool {
	found := false
	forEachBox(moov, func(boxType string, trak []byte) bool {
		if boxType != "trak" {
			return true
		}
		forEachBox(trak, func(boxType string, mdia []byte) bool {
			if boxType != "mdia" {
				return true
			}
			forEachBox(mdia, func(boxType string, hdlr []byte) bool {
				// hdlr: version and flags (4 bytes), pre_defined (4 bytes), handler type (4 bytes)
				if boxType == "hdlr" && len(hdlr) >= 12 && string(hdlr[8:12]) == "soun" {
					found = true
				}
				return !found
			})
			return !found
		})
		return !found
	})
	return found
}

// forEachBox calls fn with the type and the payload of each box found in b. The walk
// stops when fn returns false, or when the data doesn't hold a valid box anymore.
func forEachBox(b []byte, fn func(boxType string, payload []byte) bool) {
	for len(b) >= 8 {
		size := uint64(binary.BigEndian.Uint32(b[0:4]))
		boxType := string(b[4:8])
		hdrLen := uint64(8)

		switch size {
		case 0: // the box extends to the end of the data
			size = uint64(len(b))
		case 1: // 64 bits size follows the type
			if len(b) < 16 {
				return
			}
			size = binary.BigEndian.Uint64(b[8:16])
			hdrLen = 16
		}
		if size < hdrLen || size > uint64(len(b)) {
			return
		}

		if !fn(boxType, b[hdrLen:size]) {
			return
		}
		b = b[size:]
	}
}
