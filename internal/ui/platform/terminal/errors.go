package terminal

import "errors"

// ErrUnavailable indicates that the terminal UI shell is not compiled in.
var ErrUnavailable = errors.New("terminal ui unavailable")
