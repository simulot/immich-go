package native

import "errors"

// ErrUnavailable indicates that the native shell has not been implemented yet.
var ErrUnavailable = errors.New("native ui unavailable")
