package immich

import (
	"errors"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"strconv"
	"syscall"
	"time"
)

const maxRetries = 3

// retryableStatusCodes is the set of HTTP status codes that are safe to retry.
var retryableStatusCodes = map[int]bool{
	http.StatusRequestTimeout:     true, // 408
	http.StatusTooManyRequests:    true, // 429
	http.StatusInternalServerError: true, // 500
	http.StatusBadGateway:         true, // 502
	http.StatusServiceUnavailable: true, // 503
	http.StatusGatewayTimeout:     true, // 504
}

// isRetryable returns true if the error is a transient server or network error
// that is safe to retry. Permission errors (401, 403) and client errors (400, 404, 409)
// are NOT retryable.
func isRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Check for callError with retryable status code
	var ce callError
	if errors.As(err, &ce) {
		return retryableStatusCodes[ce.status]
	}

	// Check for network errors
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	// Connection refused / reset
	if errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.ECONNRESET) {
		return true
	}

	// EOF errors (server dropped connection)
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}

	return false
}

// IsRetryable is the exported version for use by the upload layer.
func IsRetryable(err error) bool {
	return isRetryable(err)
}

// retryDelay calculates the delay before the next retry attempt.
// Uses exponential backoff (1s, 2s, 4s) with random jitter (0-1s).
// If retryAfter is a valid number of seconds (from Retry-After header),
// that value is used as the base instead.
func retryDelay(attempt int, retryAfter string) time.Duration {
	base := time.Duration(1<<uint(attempt)) * time.Second

	if retryAfter != "" {
		if seconds, err := strconv.Atoi(retryAfter); err == nil && seconds > 0 {
			base = time.Duration(seconds) * time.Second
		}
	}

	jitter := time.Duration(rand.Int64N(int64(time.Second)))
	return base + jitter
}
