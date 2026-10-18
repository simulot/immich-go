package immich

import (
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJoinUploadErrors(t *testing.T) {
	otherErr := errors.New("other error")

	tests := []struct {
		name      string
		status    string
		callErr   error
		writerErr error
		wantNil   bool
		wantIs    error
	}{
		{
			name:      "duplicate ignores writer closed pipe",
			status:    UploadDuplicate,
			writerErr: io.ErrClosedPipe,
			wantNil:   true,
		},
		{
			name:    "duplicate ignores wrapped call closed pipe",
			status:  UploadDuplicate,
			callErr: fmt.Errorf("upload request: %w", io.ErrClosedPipe),
			wantNil: true,
		},
		{
			name:      "duplicate preserves unrelated writer error",
			status:    UploadDuplicate,
			writerErr: otherErr,
			wantIs:    otherErr,
		},
		{
			name:      "duplicate drops closed pipe but preserves other error",
			status:    UploadDuplicate,
			callErr:   otherErr,
			writerErr: io.ErrClosedPipe,
			wantIs:    otherErr,
		},
		{
			name:      "created upload preserves closed pipe",
			status:    UploadCreated,
			writerErr: io.ErrClosedPipe,
			wantIs:    io.ErrClosedPipe,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := joinUploadErrors(test.status, test.callErr, test.writerErr)
			if test.wantNil {
				require.NoError(t, err)
				return
			}

			require.ErrorIs(t, err, test.wantIs)
		})
	}
}
