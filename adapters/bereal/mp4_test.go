package bereal

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
)

// box builds an ISO base media box.
func box(boxType string, payload ...[]byte) []byte {
	body := bytes.Join(payload, nil)
	b := make([]byte, 8, 8+len(body))
	binary.BigEndian.PutUint32(b[0:4], uint32(8+len(body)))
	copy(b[4:8], boxType)
	return append(b, body...)
}

// box64 builds a box using the 64 bits size form.
func box64(boxType string, payload []byte) []byte {
	b := make([]byte, 16, 16+len(payload))
	binary.BigEndian.PutUint32(b[0:4], 1)
	copy(b[4:8], boxType)
	binary.BigEndian.PutUint64(b[8:16], uint64(16+len(payload)))
	return append(b, payload...)
}

// track builds a 'trak' box for the given handler type ('vide', 'soun'...).
func track(handler string) []byte {
	hdlr := box("hdlr", make([]byte, 4), make([]byte, 4), []byte(handler), make([]byte, 12), []byte("Handler\x00"))
	return box("trak", box("tkhd", make([]byte, 84)), box("mdia", box("mdhd", make([]byte, 24)), hdlr))
}

func TestHasAudioTrack(t *testing.T) {
	ftyp := box("ftyp", []byte("isom"), make([]byte, 4), []byte("isomiso2"))
	mdat := box("mdat", bytes.Repeat([]byte{0xAB}, 1000))
	moovVideo := box("moov", box("mvhd", make([]byte, 100)), track("vide"))
	moovAV := box("moov", box("mvhd", make([]byte, 100)), track("vide"), track("soun"))

	tests := []struct {
		name string
		file []byte
		want bool
	}{
		{"video and audio, moov at the end", bytes.Join([][]byte{ftyp, mdat, moovAV}, nil), true},
		{"video and audio, moov first", bytes.Join([][]byte{ftyp, moovAV, mdat}, nil), true},
		{"audio track listed first", bytes.Join([][]byte{ftyp, box("moov", track("soun"), track("vide")), mdat}, nil), true},
		{"video only, moov at the end", bytes.Join([][]byte{ftyp, mdat, moovVideo}, nil), false},
		{"video only, moov first", bytes.Join([][]byte{ftyp, moovVideo, mdat}, nil), false},
		{"media data with 64 bits size", bytes.Join([][]byte{ftyp, box64("mdat", bytes.Repeat([]byte{1}, 500)), moovAV}, nil), true},
		{"media data extending to the end, no moov", bytes.Join([][]byte{ftyp, append([]byte{0, 0, 0, 0}, []byte("mdat"+"payload")...)}, nil), false},
		{"no moov", ftyp, false},
		{"empty", nil, false},
		{"not a video", []byte("this is not an MP4 file at all"), false},
		{"truncated moov", bytes.Join([][]byte{ftyp, moovAV[:len(moovAV)-20]}, nil), false},
		{"box larger than the file", append(ftyp, 0x7f, 0xff, 0xff, 0xff, 'm', 'o', 'o', 'v'), false},
		{"box smaller than its header", append(ftyp, 0, 0, 0, 4, 'm', 'o', 'o', 'v'), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, hasAudioTrack(bytes.NewReader(tt.file)))
		})
	}
}
