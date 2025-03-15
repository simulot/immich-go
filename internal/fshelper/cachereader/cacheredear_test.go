package cachereader

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"testing"
)

func makeBuffer(offset, size int) []byte {
	b := make([]byte, size)
	for i := range b {
		b[i] = byte(i + offset)
	}
	return b
}

func Test_NewReaderAtOnBuffer(t *testing.T) {
	type fields struct {
		offset   int
		length   int
		expected []byte
	}

	b := makeBuffer(0, 4096)
	cr, _, err := NewCacheReader("test", io.NopCloser(bytes.NewReader(b)))
	if err != nil {
		t.Fatalf("NewCacheReader() error = %v", err)
	}
	tests := []fields{
		{
			offset:   0,
			length:   50,
			expected: b[:50],
		},
		{
			offset:   150,
			length:   50,
			expected: b[150:200],
		},
		{
			offset:   3000,
			length:   150,
			expected: b[3000:3150],
		},
	}

	t.Parallel()
	t.Cleanup(func() {
		cr.Close()
	})
	for _, tt := range tests {
		t.Run(fmt.Sprintf("offset=%d, length=%d", tt.offset, tt.length), func(t *testing.T) {
			r, err := cr.OpenFile()
			if err != nil {
				t.Errorf("NewReaderAt() error = %v", err)
				return
			}
			defer r.Close()
			buff := make([]byte, tt.length)
			gotBytes, err := r.ReadAt(buff, int64(tt.offset))
			if err != nil {
				t.Errorf("ReadAt() error = %v", err)
				return
			}
			if !reflect.DeepEqual(buff, tt.expected) {
				t.Errorf("NewReaderAt() =\n%v,\n want\n%v", gotBytes, tt.expected)
			}
		})
	}
}

func Test_NewReaderAtOnFile(t *testing.T) {
	type fields struct {
		offset   int
		length   int
		expected []byte
	}

	b := makeBuffer(0, 4096)
	f, err := os.CreateTemp("", "immich-go_source_*")
	if err != nil {
		t.Fatalf("os.CreateTemp() error = %v", err)
		return
	}
	defer f.Close()
	_, err = f.Write(b)
	if err != nil {
		t.Fatalf("f.Write() error = %v", err)
		return
	}

	cr, _, err := NewCacheReader("test", f)
	if err != nil {
		t.Fatalf("NewCacheReader() error = %v", err)
	}
	tests := []fields{
		{
			offset:   0,
			length:   50,
			expected: b[:50],
		},
		{
			offset:   150,
			length:   50,
			expected: b[150:200],
		},
		{
			offset:   3000,
			length:   150,
			expected: b[3000:3150],
		},
	}
	t.Parallel()
	t.Cleanup(func() {
		cr.Close()
	})
	for _, tt := range tests {
		t.Run(fmt.Sprintf("offset=%d, length=%d", tt.offset, tt.length), func(t *testing.T) {
			r, err := cr.OpenFile()
			if err != nil {
				t.Errorf("NewReaderAt() error = %v", err)
				return
			}
			defer r.Close()
			buff := make([]byte, tt.length)
			gotBytes, err := r.ReadAt(buff, int64(tt.offset))
			if err != nil {
				t.Errorf("ReadAt() error = %v", err)
				return
			}
			if !reflect.DeepEqual(buff, tt.expected) {
				t.Errorf("NewReaderAt() =\n%v,\n want\n%v", gotBytes, tt.expected)
			}
		})
	}
}
