package exif

import (
	"bytes"
	"io"
	"reflect"
	"testing"
)

func GenRandomBytes(size int) (blk []byte) {
	blk = make([]byte, size)
	for i := 0; i < size; i++ {
		blk[i] = byte(i & 0xff)
	}
	return
}

func Test_searchPattern(t *testing.T) {
	searchBufferSize := 20
	tests := []struct {
		name    string
		r       io.Reader
		pattern []byte
		found   bool
	}{
		{
			name:    "notin",
			r:       bytes.NewReader(GenRandomBytes(searchBufferSize * 3)),
			pattern: []byte{5, 4, 3, 2},
			found:   false,
		},
		{
			name:    "at end of reader",
			r:       bytes.NewReader(GenRandomBytes(searchBufferSize * 3)),
			pattern: []byte{56, 57, 58, 59},
			found:   true,
		},
		{
			name:    "at 1st buffer boundary",
			r:       bytes.NewReader(GenRandomBytes(searchBufferSize * 3)),
			pattern: []byte{18, 19, 20, 21},
			found:   true,
		},
		{
			name:    "at 2nd buffer boundary",
			r:       bytes.NewReader(GenRandomBytes(searchBufferSize * 3)),
			pattern: []byte{34, 35, 36, 37},
			found:   true,
		},
		{
			name:    "not in real",
			r:       bytes.NewReader(append(GenRandomBytes(searchBufferSize/3), "this is the date:2023-08-01T20:20:00 in the middle of the buffer"...)),
			pattern: []byte("nothere"),
			found:   false,
		},
		{
			name:    "middle",
			r:       bytes.NewReader(append(GenRandomBytes(searchBufferSize/3), "this is the date:2023-08-01T20:20:00 in the middle of the buffer"...)),
			pattern: []byte("date:"),
			found:   true,
		},
		{
			name:    "beginning",
			r:       bytes.NewReader([]byte("date:2023-08-01T20:20:00 in the middle of the buffer")),
			pattern: []byte("date:"),
			found:   true,
		},
		{
			name:    "2ndbuffer",
			r:       bytes.NewReader(append(GenRandomBytes(3*searchBufferSize), "this is the date:2023-08-01T20:20:00 in the middle of the buffer"...)),
			pattern: []byte("date:"),
			found:   true,
		},
		{
			name:    "crossing buffer boundaries",
			r:       bytes.NewReader(append(append(GenRandomBytes(2*searchBufferSize-10), "date:2023-08-01T20:20:00 in the middle of the buffer"...), GenRandomBytes(searchBufferSize-10)...)),
			pattern: []byte("date:"),
			found:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := make([]byte, searchBufferSize)
			r, err := searchPattern(tt.r, tt.pattern, b)
			if err != nil && tt.found {
				t.Errorf("Pattern %v not found", tt.pattern)
				return
			}
			if !tt.found && err == io.EOF {
				return
			}
			got, err := r.ReadSlice(len(tt.pattern))
			if err != nil {
				t.Errorf("Can't read result: %s", err)
				return
			}

			if !reflect.DeepEqual(got, tt.pattern) {
				t.Errorf("searchPattern() = %v, want %v", got, tt.pattern)
			}
		})
	}
}
