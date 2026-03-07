package immich

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"syscall"
	"testing"
	"time"
)

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"500 error", callError{status: http.StatusInternalServerError}, true},
		{"502 error", callError{status: http.StatusBadGateway}, true},
		{"503 error", callError{status: http.StatusServiceUnavailable}, true},
		{"504 error", callError{status: http.StatusGatewayTimeout}, true},
		{"408 error", callError{status: http.StatusRequestTimeout}, true},
		{"429 error", callError{status: http.StatusTooManyRequests}, true},
		{"400 error", callError{status: http.StatusBadRequest}, false},
		{"401 error", callError{status: http.StatusUnauthorized}, false},
		{"403 error", callError{status: http.StatusForbidden}, false},
		{"404 error", callError{status: http.StatusNotFound}, false},
		{"409 error", callError{status: http.StatusConflict}, false},
		{"connection refused", fmt.Errorf("connect: %w", syscall.ECONNREFUSED), true},
		{"connection reset", fmt.Errorf("read: %w", syscall.ECONNRESET), true},
		{"EOF", io.EOF, true},
		{"unexpected EOF", io.ErrUnexpectedEOF, true},
		{"net timeout", netTimeoutError{}, true},
		{"wrapped retryable", fmt.Errorf("outer: %w", callError{status: 503}), true},
		{"generic error", errors.New("something"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRetryable(tt.err)
			if got != tt.expected {
				t.Errorf("isRetryable(%v) = %v, want %v", tt.err, got, tt.expected)
			}
		})
	}
}

// netTimeoutError implements net.Error with Timeout() = true
type netTimeoutError struct{}

func (e netTimeoutError) Error() string   { return "timeout" }
func (e netTimeoutError) Timeout() bool   { return true }
func (e netTimeoutError) Temporary() bool { return true }

func TestRetryDelay(t *testing.T) {
	for attempt := 0; attempt < 3; attempt++ {
		d := retryDelay(attempt, "")
		base := time.Duration(1<<uint(attempt)) * time.Second
		if d < base || d > base+time.Second {
			t.Errorf("attempt %d: delay %v outside expected range [%v, %v]", attempt, d, base, base+time.Second)
		}
	}
}

func TestRetryDelay_RetryAfter(t *testing.T) {
	d := retryDelay(0, "5")
	if d < 5*time.Second || d > 6*time.Second {
		t.Errorf("with Retry-After=5, got %v, expected ~5s", d)
	}
}
