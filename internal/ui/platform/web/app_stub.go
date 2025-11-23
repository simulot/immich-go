package web

import (
	"context"
	"errors"

	"github.com/simulot/immich-go/internal/ui/core/messages"
)

// Config represents web-shell specific options (intentionally empty for now).
type Config struct{}

// ErrUnavailable indicates that the web shell is not implemented yet.
var ErrUnavailable = errors.New("web ui unavailable")

// Run is a placeholder to keep the package compiling until the web shell exists.
func Run(context.Context, Config, messages.Stream) error {
	return ErrUnavailable
}
