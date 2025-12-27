package web

import "errors"

// ErrUnavailable indicates that the web shell is not implemented yet.
var ErrUnavailable = errors.New("web ui unavailable")
