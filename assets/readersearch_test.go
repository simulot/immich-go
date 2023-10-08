package assets

import (
	"bytes"
	"crypto/rand"
	"io"
	"reflect"
	"testing"
)

func GenRandomBytes(size int) (blk []byte) {
	blk = make([]byte, size)
	rand.Read(blk)
	return
}

func Test_searchPattern(t *testing.T) {
	type args struct {
		r          io.Reader
		pattern    []byte
		maxDataLen int
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "notin",
			args: args{
				r:          bytes.NewReader(append(GenRandomBytes(searchBufferSize/3), "this is the date:2023-08-01T20:20:00 in the middle of the buffer"...)),
				pattern:    []byte("nothere"),
				maxDataLen: 24,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "middle",
			args: args{
				r:          bytes.NewReader(append(GenRandomBytes(searchBufferSize/3), "this is the date:2023-08-01T20:20:00 in the middle of the buffer"...)),
				pattern:    []byte("date:"),
				maxDataLen: 24,
			},
			want:    []byte("date:2023-08-01T20:20:00"),
			wantErr: false,
		},
		{
			name: "beginning",
			args: args{
				r:          bytes.NewReader([]byte("date:2023-08-01T20:20:00 in the middle of the buffer")),
				pattern:    []byte("date:"),
				maxDataLen: 24,
			},
			want:    []byte("date:2023-08-01T20:20:00"),
			wantErr: false,
		},
		{
			name: "2ndbuffer",
			args: args{
				r:          bytes.NewReader(append(GenRandomBytes(3*searchBufferSize), "this is the date:2023-08-01T20:20:00 in the middle of the buffer"...)),
				pattern:    []byte("date:"),
				maxDataLen: 24,
			},
			want:    []byte("date:2023-08-01T20:20:00"),
			wantErr: false,
		},
		{
			name: "crossing buffer boundaries",
			args: args{
				r:          bytes.NewReader(append(append(GenRandomBytes(2*searchBufferSize-10), "date:2023-08-01T20:20:00 in the middle of the buffer"...), GenRandomBytes(searchBufferSize-10)...)),
				pattern:    []byte("date:"),
				maxDataLen: 24,
			},
			want:    []byte("date:2023-08-01T20:20:00"),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := searchPattern(tt.args.r, tt.args.pattern, tt.args.maxDataLen)
			if (err != nil) != tt.wantErr {
				t.Errorf("searchPattern() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("searchPattern() = %v, want %v", got, tt.want)
			}
		})
	}
}
